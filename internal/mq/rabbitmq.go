package mq

import (
	"fmt"

	config "github.com/dhruvthak3r/Probe/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch    *amqp.Channel
	queue amqp.Queue
}

type Consumer struct {
	ch    *amqp.Channel
	queue amqp.Queue
}

func NewRabbitMQQueue(ch *amqp.Channel) (amqp.Queue, error) {
	q, err := ch.QueueDeclare("monitor_results", true, false, false, false, nil)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("error declaring rabbitmq queue: %v", err)
	}
	return q, nil
}

func NewRabbitMQPublisher(rmq config.RabbitMQ) (*Publisher, error) {
	ch, err := rmq.NewRabbitMQChannel()
	if err != nil {
		return nil, fmt.Errorf("error creating rabbitmq channel: %v", err)
	}

	q, err := NewRabbitMQQueue(ch)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("error creating rabbitmq queue: %v", err)
	}

	return &Publisher{
		ch:    ch,
		queue: q,
	}, nil
}

func (p *Publisher) Close() error {
	if p == nil || p.ch == nil {
		return nil
	}
	return p.ch.Close()
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

func (c *Consumer) Close() error {
	if c == nil || c.ch == nil {
		return nil
	}
	return c.ch.Close()
}
