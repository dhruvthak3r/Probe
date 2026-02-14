package mq

import (
	"context"
	"encoding/json"
	"fmt"

	db "github.com/dhruvthak3r/Probe/config"
)

func (c *Consumer) ConsumeFromQueue(ctx context.Context, db *db.DB) error {

	mssgs, err := c.ch.Consume(c.queue.Name, "", false, false, false, false, nil)
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

			err = InsertResults(ctx, db, &res)
			if err != nil {

				nack := m.Nack(false, true)
				if nack != nil {
					fmt.Printf("error nacking message: %v", nack)
				}

				return fmt.Errorf("error inserting results into db: %v", err)
			}

			if ack := m.Ack(false); ack != nil {
				fmt.Printf("error acknowledging message: %v", ack)
			}

		case <-ctx.Done():
			return fmt.Errorf("stopping consumer %v", ctx.Err())
		}
	}
}

func InsertResults(ctx context.Context, db *db.DB, res *ResultMessage) error {

	InsertQuery := `INSERT INTO results (monitor_id, status_code, status, dns_response_time, connection_time, tls_handshake_time, resolved_ip, first_byte_time, download_time, response_time, throughput, reason) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	values := []interface{}{
		res.MonitorID,
		res.StatusCode,
		res.Status,
		res.DNSResponseTime,
		res.ConnectionTime,
		res.TLSHandshakeTime,
		res.ResolvedIP,
		res.FirstByteTime,
		res.DownloadTime,
		res.ResponseTime,
		res.Throughput,
		res.Reason,
	}
	_, err := db.Pool.ExecContext(ctx, InsertQuery, values...)

	if err != nil {
		return fmt.Errorf("error inserting results into db: %v", err)
	}

	return nil

}
