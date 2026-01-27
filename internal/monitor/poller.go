package monitor

import (
	"context"
	"crypto/tls"
	db "dhruv/probe/internal/config"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"time"
)

type result struct {
	StatusCode       int
	DNSResponseTime  time.Duration
	ConnectionTime   time.Duration
	TLSHandshakeTime time.Duration
	ResolvedIp       string
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

func GetResults(url string) {
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			fmt.Printf("DNS Start: %v\n", time.Now())
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			fmt.Printf("DNS Done: %v\n", info.Addrs[0].IP.String())
		},

		ConnectStart: func(network, addr string) {
			fmt.Printf("Connect Start: %v\n", time.Now())
		},
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("Connect Done: %v\n", time.Now())
		},

		TLSHandshakeStart: func() {
			fmt.Printf("TLS Handshake Start: %v\n", time.Now())
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			fmt.Printf("TLS Handshake Done: %v\n", cs.ServerName)
		},

		GotConn: func(gci httptrace.GotConnInfo) {
			fmt.Printf("Got Conn: %v\n", time.Now())
		},

		WroteRequest: func(wri httptrace.WroteRequestInfo) {
			fmt.Printf("Wrote Request: %v\n", time.Now())
		},

		GotFirstResponseByte: func() {
			fmt.Printf("Got First Response Byte: %v\n", time.Now())
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s\n", resp.Status)
}
