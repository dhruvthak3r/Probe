package mq

import (
	"context"
	"fmt"

	"github.com/dhruvthak3r/Probe/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	ch    *amqp.Channel
	queue amqp.Queue
}

func NewConsumer(rmq config.RabbitMQ) (*Consumer, error) {
	ch, err := rmq.NewRabbitMQChannel()
	if err != nil {
		return nil, fmt.Errorf("error creating rabbitmq channel: %v", err)
	}

	q, err := NewRabbitMQQueue(ch)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("error creating rabbitmq queue: %v", err)
	}

	return &Consumer{
		ch:    ch,
		queue: q,
	}, nil
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

func (c *Consumer) Close() error {
	if c == nil || c.ch == nil {
		return nil
	}
	return c.ch.Close()
}
