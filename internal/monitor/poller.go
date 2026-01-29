package monitor

import (
	"context"
	"crypto/tls"
	db "dhruv/probe/internal/config"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

type Result struct {
	StatusCode       int
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

			fmt.Printf(
				"Polling URL: %s (Monitor ID: %d)\n",
				m.Url,
				m.ID,
			)

			update := `
                UPDATE monitor
                SET last_run_at = NOW(),
                next_run_at = DATE_ADD(NOW(), INTERVAL ? SECOND),
                status = 'idle'
                WHERE monitor_id = ?`

			_, err := db.Pool.ExecContext(ctx, update, m.FrequencySecs, m.ID)
			if err != nil {
				return fmt.Errorf("error updating monitor after poll: %v\n", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func GetResult(url string) *Result {

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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("ooopss")
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start := time.Now()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ooopss")
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode

	bytesRead, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		fmt.Println("ooopss")
	}

	end := time.Now()

	downloadTime := end.Sub(firstByte)

	if downloadTime > 0 {
		throughput = float64(bytesRead) / downloadTime.Seconds()
	}

	return &Result{
		StatusCode: statusCode,
		ResolvedIp: resolvedIP,

		DNSResponseTime:  durationOrZero(dnsStart, dnsEnd),
		ConnectionTime:   durationOrZero(connectStart, connectEnd),
		TLSHandshakeTime: durationOrZero(tlsStart, tlsEnd),

		FirstByteTime: firstByte.Sub(start),
		DownloadTime:  downloadTime,
		ResponseTime:  end.Sub(start),

		Throughput: throughput,
	}
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
