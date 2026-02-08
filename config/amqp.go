package config

import (
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn *amqp.Connection
}

func NewRabbitMQConnection() (*RabbitMQ, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))

	if err != nil {
		return nil, fmt.Errorf("error connecting rabbitmq : %v", err)
	}

	return &RabbitMQ{conn: conn}, nil
}

func (mq *RabbitMQ) NewRabbitMQChannel() (*amqp.Channel, error) {
	ch, err := mq.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error creating rabbitmq channel: %v", err)
	}
	return ch, nil
}

func (mq *RabbitMQ) Close() error {
	if mq == nil || mq.conn == nil {
		return nil
	}
	return mq.conn.Close()
}
