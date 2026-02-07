package mq

import (
	"context"
	"fmt"

	"github.com/dhruvthak3r/Probe/config"
)

func ConsumeFromQueue(ctx context.Context) error {
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

	mssgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
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
