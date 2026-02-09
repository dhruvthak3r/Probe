package mq

import (
	"context"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type ResultMessage struct {
	MonitorID        int     `json:"monitor_id"`
	MonitorUrl       string  `json:"monitor_url"`
	StatusCode       int     `json:"status_code"`
	Status           string  `json:"status"`
	DNSResponseTime  int64   `json:"dns_response_time_ms,omitempty"`
	ConnectionTime   int64   `json:"connection_time_ms,omitempty"`
	TLSHandshakeTime int64   `json:"tls_handshake_time_ms,omitempty"`
	ResolvedIP       string  `json:"resolved_ip,omitempty"`
	FirstByteTime    int64   `json:"first_byte_time_ms,omitempty"`
	DownloadTime     int64   `json:"download_time_ms,omitempty"`
	ResponseTime     int64   `json:"response_time_ms,omitempty"`
	Throughput       float64 `json:"throughput,omitempty"`
	Reason           string  `json:"reason,omitempty"`
}

func (rmq *Publisher) PublishToQueue(ctx context.Context, payload []byte) error {

	err := rmq.ch.PublishWithContext(ctx, "", rmq.queue.Name, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        payload,
	})

	if err != nil {
		return fmt.Errorf("error publishing to rabbitmq: %v", err)
	}
	return nil
}
