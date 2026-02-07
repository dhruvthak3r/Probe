package mq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQQueue(ch *amqp.Channel) (amqp.Queue, error) {
	q, err := ch.QueueDeclare("monitor_results", true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("error declaring rabbitmq queue: %v", err)
	}
	return q, nil
}
