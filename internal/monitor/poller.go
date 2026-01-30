package monitor

import (
	"context"
	"crypto/tls"
	db "dhruv/probe/internal/config"
	"dhruv/probe/internal/logger"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
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

	start := time.Now()

	for key, val := range m.RequestHeaders {
		req.Header.Set(key, val)
	}

	transport := &http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting the response from client.Do %v", err)
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode
	statusValid := validateResponseStatusCode(statusCode, m.AcceptedStatusCodes)
	if !statusValid {
		return &Result{
			StatusCode: resp.StatusCode,
			status:     "DOWN",
		}, nil
	}

	bytesRead, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading the response body %v", err)
	}

	end := time.Now()

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

func validateResponseStatusCode(respStatusCode int, acceptedStatusCode []int) bool {
	for _, c := range acceptedStatusCode {
		if c == respStatusCode {
			return true
		}
	}

	return false
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
