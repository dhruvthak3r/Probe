package mq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	ch    *amqp.Channel
	queue amqp.Queue
}

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
			fmt.Println("body received:", string(m.Body))

		case <-ctx.Done():
			return fmt.Errorf("stopping consumer %v", ctx.Err())
		}
	}
}
