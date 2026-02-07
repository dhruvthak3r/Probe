package mq

import (
	"context"
	"fmt"

	"github.com/rabbitmq/amqp091-go"

	"github.com/dhruvthak3r/Probe/config"
)

func PublishToQueue(ctx context.Context, payload []byte) error {
	conn, err := config.NewRabbitMQConnection()
	if err != nil {
		return fmt.Errorf("error connecting to rabbitmq: %v", err)
	}
	defer conn.Close()

	ch, err := config.NewRabbitMQChannel(conn)
	if err != nil {
		return fmt.Errorf("error creating rabbitmq channel: %v", err)
	}
	defer ch.Close()

	q, err := NewRabbitMQQueue(ch)
	if err != nil {
		return fmt.Errorf("error creating rabbitmq queue: %v", err)
	}

	err = ch.PublishWithContext(ctx, "", q.Name, false, false, amqp091.Publishing{
		ContentType: "application/json",
		Body:        payload,
	})

	if err != nil {
		return fmt.Errorf("error publishing to rabbitmq: %v", err)
	}
	return nil
}
