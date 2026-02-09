package monitor

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"time"
)

func GetResult(m Monitor) (*Result, error) {

	var (
		dnsStart, dnsEnd         time.Time
		connectStart, connectEnd time.Time
		tlsStart, tlsEnd         time.Time
		firstByte                time.Time
		resolvedIP               string
		statusCode               int
		bytesRead                int64
		throughput               float64
	)

	trace := BuildTrace(&dnsStart, &dnsEnd, &resolvedIP, &connectStart, &connectEnd, &tlsStart, &tlsEnd, &firstByte)

	req, err := Buildreq(m, trace)
	if err != nil {
		return nil, fmt.Errorf("error building the request %v", err)
	}

	if m.RequestHeaders != nil {
		setRequestHeaders(m, req)
	}

	client := BuildClient(m)

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return &Result{
				MonitorID:  m.ID,
				MonitorUrl: m.Url,
				Status:     "DOWN",
				Reason:     "connection timed out",
			}, nil
		}
		return nil, fmt.Errorf("error getting the response from client.Do %v", err)
	}
	defer resp.Body.Close()

	bytesRead, err = io.CopyN(io.Discard, resp.Body, 1024)
	if err != nil {
		return nil, fmt.Errorf("error reading the response body %v", err)
	}

	end := time.Now()

	statusCode = resp.StatusCode

	statusValid := ValidateResponseStatusCode(statusCode, m.AcceptedStatusCodes)
	if !statusValid {
		return &Result{
			MonitorID:  m.ID,
			MonitorUrl: m.Url,
			StatusCode: resp.StatusCode,
			Status:     "DOWN",
			Reason:     fmt.Sprintf("status code %d not in accepted list", resp.StatusCode),
		}, nil
	}

	responseheadersValid := ValidateResponseHeaders(m.ResponseHeaders, resp.Header)
	if !responseheadersValid {
		return &Result{
			MonitorID:  m.ID,
			MonitorUrl: m.Url,
			StatusCode: resp.StatusCode,
			Status:     "DOWN",
			Reason:     "response headers did not match expected values",
		}, nil
	}

	downloadTime := end.Sub(firstByte)

	if downloadTime > 0 {
		throughput = float64(bytesRead) / downloadTime.Seconds()
	}

	return &Result{
		MonitorID:  m.ID,
		MonitorUrl: m.Url,
		StatusCode: statusCode,
		Status:     "UP",
		ResolvedIp: resolvedIP,

		DNSResponseTime:  durationOrZero(dnsStart, dnsEnd),
		ConnectionTime:   durationOrZero(connectStart, connectEnd),
		TLSHandshakeTime: durationOrZero(tlsStart, tlsEnd),

		FirstByteTime: firstByte.Sub(start),
		DownloadTime:  downloadTime,
		ResponseTime:  end.Sub(start),

		Throughput: throughput,
	}, nil
}

func Buildreq(m Monitor, trace *httptrace.ClientTrace) (*http.Request, error) {
	var bodyReader io.Reader

	if m.HttpMethod != "GET" && m.RequestBody.Valid {
		bodyReader = bytes.NewReader([]byte(m.RequestBody.String))
	}

	req, err := http.NewRequest(m.HttpMethod, m.Url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating the request: %w", err)
	}

	req = req.WithContext(
		httptrace.WithClientTrace(req.Context(), trace),
	)

	if m.RequestBody.Valid {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(m.RequestBody.String))
	}

	return req, nil
}

func setRequestHeaders(m Monitor, req *http.Request) {
	for key, values := range m.RequestHeaders {
		if key == "" {
			continue
		}

		canonicalKey := textproto.CanonicalMIMEHeaderKey(key)

		for _, val := range values {
			val = strings.TrimSpace(val)
			if val == "" {
				continue
			}

			switch canonicalKey {
			case "Host":

				req.Host = val

			case "Content-Length", "Transfer-Encoding", "Connection":

				continue

			case "Cookie", "Set-Cookie", "Accept", "Accept-Encoding":

				req.Header.Add(canonicalKey, val)

			default:
				req.Header.Add(canonicalKey, val)
			}
		}
	}
}

func BuildClient(m Monitor) *http.Client {
	const defaultConnectionTimeout = 10 * time.Second

	dialer := &net.Dialer{
		Timeout: defaultConnectionTimeout,
	}
	if m.ConnectionTimeout.Valid && m.ConnectionTimeout.Int64 > 0 {
		dialer.Timeout = time.Duration(m.ConnectionTimeout.Int64) * time.Second
	}

	transport := &http.Transport{
		DialContext:       dialer.DialContext,
		DisableKeepAlives: true,
	}

	return &http.Client{Transport: transport}
}

func BuildTrace(dnsStart *time.Time, dnsEnd *time.Time, resolvedIP *string, connectStart *time.Time, connectEnd *time.Time, tlsStart *time.Time, tlsEnd *time.Time, firstByte *time.Time) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			*dnsStart = time.Now()
		},
		DNSDone: func(di httptrace.DNSDoneInfo) {
			*dnsEnd = time.Now()

			for _, addr := range di.Addrs {
				if ipv4 := addr.IP.To4(); ipv4 != nil {
					*resolvedIP = ipv4.String()
					return
				}
			}

			if len(di.Addrs) > 0 {
				*resolvedIP = di.Addrs[0].IP.String()
			}
		},

		ConnectStart: func(_, _ string) {
			*connectStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			*connectEnd = time.Now()
		},

		TLSHandshakeStart: func() {
			*tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			*tlsEnd = time.Now()
		},

		GotFirstResponseByte: func() {
			*firstByte = time.Now()
		},
	}
}

func durationOrZero(start, end time.Time) time.Duration {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return end.Sub(start)
}
