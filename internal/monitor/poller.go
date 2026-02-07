package monitor

import (
	"context"
	"encoding/json"

	"fmt"
	"time"

	db "github.com/dhruvthak3r/Probe/config"
	"github.com/dhruvthak3r/Probe/internal/logger"
	resultq "github.com/dhruvthak3r/Probe/internal/mq"
)

type Result struct {
	StatusCode       int           `json:"status_code"`
	Status           string        `json:"status"`
	DNSResponseTime  time.Duration `json:"dns_response_time,omitempty"`
	ConnectionTime   time.Duration `json:"connection_time,omitempty"`
	TLSHandshakeTime time.Duration `json:"tls_handshake_time,omitempty"`
	ResolvedIp       string        `json:"resolved_ip,omitempty"`
	FirstByteTime    time.Duration `json:"first_byte_time,omitempty"`
	DownloadTime     time.Duration `json:"download_time,omitempty"`
	ResponseTime     time.Duration `json:"response_time,omitempty"`
	Throughput       float64       `json:"throughput,omitempty"`
	Reason           string        `json:"reason,omitempty"`
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

			payload, err := json.Marshal(res)
			if err != nil {
				return fmt.Errorf("error marshalling monitor data: %v", err)
			}

			err = resultq.PublishToQueue(ctx, payload)
			if err != nil {
				return fmt.Errorf("error publishing monitor data to queue: %v", err)
			}

			err = resultq.ConsumeFromQueue(ctx)
			if err != nil {
				return fmt.Errorf("error consuming monitor data from queue: %v", err)
			}

			update := `
                UPDATE monitor
                SET last_run_at = NOW(),
                next_run_at = DATE_ADD(NOW(), INTERVAL ? SECOND),
                status = 'idle'
                WHERE monitor_id = ?`

			_, Execerr := db.Pool.ExecContext(ctx, update, m.FrequencySecs, m.ID)
			if Execerr != nil {
				return fmt.Errorf("error updating monitor after poll: %v\n", Execerr)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func LogResult(res *Result, url string) {
	logger.Log.Println("Result:")
	logger.Log.Println("URL:", url)
	logger.Log.Println("StatusCode:", res.StatusCode)
	logger.Log.Println("Status:", res.Status)
	if res.Reason != "" {
		logger.Log.Println("Reason:", res.Reason)
	}
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
