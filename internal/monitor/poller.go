package monitor

import (
	"context"
	"crypto/tls"
	"net"

	db "github.com/dhruvthak3r/Probe/internal/config"
	"github.com/dhruvthak3r/Probe/internal/logger"

	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"time"
	//"github.com/PaesslerAG/jsonpath"
)

type Result struct {
	StatusCode       int
	status           string
	DNSResponseTime  time.Duration
	ConnectionTime   time.Duration
	TLSHandshakeTime time.Duration
	ResolvedIp       string
	FirstByteTime    time.Duration
	DownloadTime     time.Duration
	ResponseTime     time.Duration
	Throughput       float64
}

func (mq *MonitorQueue) PollUrls(ctx context.Context, db *db.DB) error {

	for {
		select {
		case m, ok := <-mq.UrlsToPoll:
			if !ok {

				return nil
			}
			res, err := GetResult(*m)

			if err != nil {
				return fmt.Errorf("error getting results %v", err)
			}

			LogResult(res, m.Url)

			update := `
                UPDATE monitor
                SET last_run_at = NOW(),
                next_run_at = DATE_ADD(NOW(), INTERVAL ? SECOND),
                status = 'idle'
                WHERE monitor_id = ?`

			_, Execerr := db.Pool.ExecContext(ctx, update, m.FrequencySecs, m.ID)
			if Execerr != nil {
				return fmt.Errorf("error updating monitor after poll: %v\n", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

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

	req, err := http.NewRequest(m.HttpMethod, m.Url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating the request %v\n", err)
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	setRequestHeaders(m, req)

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

	client := &http.Client{Transport: transport}

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
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
			StatusCode: resp.StatusCode,
			status:     "DOWN",
		}, nil
	}

	responseheadersValid := ValidateResponseHeaders(m.ResponseHeaders, resp.Header)
	if !responseheadersValid {
		return &Result{
			StatusCode: resp.StatusCode,
			status:     "DOWN",
		}, nil
	}

	downloadTime := end.Sub(firstByte)

	if downloadTime > 0 {
		throughput = float64(bytesRead) / downloadTime.Seconds()
	}

	return &Result{
		StatusCode: statusCode,
		status:     "UP",
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

func durationOrZero(start, end time.Time) time.Duration {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return end.Sub(start)
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

func ValidateResponseStatusCode(respStatusCode int, acceptedStatusCode []int) bool {
	for _, c := range acceptedStatusCode {
		if c == respStatusCode {
			return true
		}
	}

	return false
}

func ValidateResponseHeaders(expected map[string][]string, respHeaders http.Header) bool {
	for expectedKey, expectedValues := range expected {

		canonicalKey := textproto.CanonicalMIMEHeaderKey(expectedKey)
		actualValues := respHeaders[canonicalKey]

		if len(actualValues) == 0 {
			return false
		}

		for _, expectedVal := range expectedValues {
			expectedVal = strings.TrimSpace(expectedVal)
			matchFound := false

			for _, actualVal := range actualValues {
				actualVal = strings.TrimSpace(actualVal)

				if actualVal == expectedVal || strings.Contains(actualVal, expectedVal) {
					matchFound = true
					break
				}
			}

			if !matchFound {
				return false
			}
		}
	}

	return true
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

func LogResult(res *Result, url string) {
	logger.Log.Println("Result:")
	logger.Log.Println("URL:", url)
	logger.Log.Println("StatusCode:", res.StatusCode)
	logger.Log.Println("Status:", res.status)
	logger.Log.Println("ResolvedIp:", res.ResolvedIp)

	logger.Log.Println("DNSResponseTime:", res.DNSResponseTime)
	logger.Log.Println("ConnectionTime:", res.ConnectionTime)
	logger.Log.Println("TLSHandshakeTime:", res.TLSHandshakeTime)

	logger.Log.Println("FirstByteTime:", res.FirstByteTime)
	logger.Log.Println("DownloadTime:", res.DownloadTime)
	logger.Log.Println("ResponseTime:", res.ResponseTime)

	logger.Log.Println("Throughput (bytes/sec):", res.Throughput)
	logger.Log.Println("--------------------------------------------------")
}
