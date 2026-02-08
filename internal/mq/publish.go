package mq

import (
	"context"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

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
