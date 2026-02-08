package monitor

import (
	"context"
	"encoding/json"

	"fmt"
	"time"

	db "github.com/dhruvthak3r/Probe/config"
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

func (mq *MonitorQueue) PollUrls(ctx context.Context, db *db.DB, rmq *resultq.Publisher) error {

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

			err = rmq.PublishToQueue(ctx, payload)
			if err != nil {
				return fmt.Errorf("error publishing monitor data to queue: %v", err)
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
