package mq

import (
	"context"
	"encoding/json"
	"fmt"
	//db "github.com/dhruvthak3r/Probe/config"
)

func (c *Consumer) ConsumeFromQueue(ctx context.Context) error {

	mssgs, err := c.ch.Consume(c.queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error consuming from rabbitmq: %v", err)
	}

	for {
		select {
		case m, ok := <-mssgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}

			res := ResultMessage{}

			err := json.Unmarshal(m.Body, &res)
			if err != nil {
				return fmt.Errorf("failed to unmarshal message body: %v", err)
			}

		case <-ctx.Done():
			return fmt.Errorf("stopping consumer %v", ctx.Err())
		}
	}
}

//func InsertResults(ctx context.Context, db *db.DB, res *ResultMessage) error {

// InsertQuery := `INSERT INTO monitor_results (monitor_id, status_code, status, dns_response_time_ms, connection_time_ms, tls_handshake_time_ms, resolved_ip, first_byte_time_ms, download_time_ms, response_time_ms, throughput, reason) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// values := []interface{}{
// 	res.MonitorID,
// 	res.StatusCode,
// 	res.Status,
// 	res.DNSResponseTime.Milliseconds(),
// 	res.ConnectionTime.Milliseconds(),
// 	res.TLSHandshakeTime.Milliseconds(),
// 	res.ResolvedIp,
// 	res.FirstByteTime.Milliseconds(),
// 	res.DownloadTime.Milliseconds(),
// 	res.ResponseTime.Milliseconds(),
// 	res.Throughput,
// 	res.Reason,
// }
// _, err := db.Pool.ExecContext(ctx, InsertQuery, values...)

// if err != nil {
// 	return fmt.Errorf("error inserting results into db: %v", err)
//}

//return nil

//}
