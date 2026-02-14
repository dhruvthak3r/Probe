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
	MonitorID        int           `json:"monitor_id"`
	MonitorUrl       string        `json:"monitor_url"`
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

			func(m *Monitor) {

				res, err := GetResult(*m)
				if err != nil {
					fmt.Printf("error getting results: %v\n", err)
					return
				}

				resconv := ToResultMessage(*res)

				payload, err := json.Marshal(resconv)
				if err != nil {
					fmt.Printf("error marshalling monitor data: %v\n", err)
					return
				}

				// if m.FrequencySecs >= 1800 {
				// 	fmt.Printf("publishing results for %d", m.ID)
				// }

				if err := rmq.PublishToQueue(ctx, payload); err != nil {
					fmt.Printf("error publishing monitor data to queue: %v\n", err)
					return
				}

				if err := SetStatusToIdle(ctx, db, m); err != nil {
					fmt.Printf("error setting monitor status to idle: %v\n", err)
				}

			}(m)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func ToResultMessage(res Result) *resultq.ResultMessage {
	return &resultq.ResultMessage{
		MonitorID:        res.MonitorID,
		MonitorUrl:       res.MonitorUrl,
		StatusCode:       res.StatusCode,
		Status:           res.Status,
		DNSResponseTime:  res.DNSResponseTime.Milliseconds(),
		ConnectionTime:   res.ConnectionTime.Milliseconds(),
		TLSHandshakeTime: res.TLSHandshakeTime.Milliseconds(),
		ResolvedIP:       res.ResolvedIp,
		FirstByteTime:    res.FirstByteTime.Milliseconds(),
		DownloadTime:     res.DownloadTime.Milliseconds(),
		ResponseTime:     res.ResponseTime.Milliseconds(),
		Throughput:       res.Throughput,
		Reason:           res.Reason,
	}
}
