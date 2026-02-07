package config

import (
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQConnection() (*amqp.Connection, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))

	if err != nil {
		return nil, fmt.Errorf("error connecting rabbitmq : %v", err)
	}

	return conn, nil
}

func NewRabbitMQChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("error creating rabbitmq channel: %v", err)
	}
	return ch, nil
}
