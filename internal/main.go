package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dhruvthak3r/Probe/config"
	db "github.com/dhruvthak3r/Probe/config"

	"github.com/dhruvthak3r/Probe/internal/monitor"
	"github.com/dhruvthak3r/Probe/internal/mq"

	"github.com/go-co-op/gocron/v2"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {

	_ = godotenv.Load("../.env")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := db.NewDBConnection()
	if err != nil {
		fmt.Printf("error connecting to db: %v\n", err)
		return
	}

	defer conn.Pool.Close()

	monitorq := monitor.NewMonitorQueue()

	rmqconn, err := config.NewRabbitMQConnection()
	if err != nil {
		fmt.Printf("error connecting to rabbitmq: %v\n", err)
		return
	}
	defer rmqconn.Close()

	publisher, err := mq.NewRabbitMQPublisher(*rmqconn)
	if err != nil {
		fmt.Printf("error creating rabbitmq publisher: %v\n", err)
		return
	}
	defer publisher.Close()

	// Create handler for processing consumed messages
	messageHandler := func(msg *mq.ResultMessage) error {
		result := &monitor.Result{
			MonitorID:       msg.MonitorID,
			MonitorUrl:      msg.MonitorUrl,
			StatusCode:      msg.StatusCode,
			Status:          msg.Status,
			ResolvedIp:      msg.ResolvedIP,
			Reason:          msg.Reason,
			Throughput:      msg.Throughput,
		}
		return monitor.InsertResults(ctx, conn, result)
	}

	consumer, err := mq.NewConsumer(*rmqconn, messageHandler)
	if err != nil {
		fmt.Printf("error creating rabbitmq consumer: %v\n", err)
		return
	}
	defer consumer.Close()

	g, ctx := errgroup.WithContext(ctx)

	s, err := gocron.NewScheduler()

	if err != nil {
		fmt.Printf("error creating scheduler: %v\n", err)
	}

	_, j_err := s.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			monitorq.RunScheduler(ctx, conn),
		),
	)

	if j_err != nil {
		fmt.Printf("error creating job: %v\n", err)
	}

	s.Start()

	g.Go(func() error {
		return monitorq.PollUrls(ctx, conn, publisher)
	})

	g.Go(func() error {
		return consumer.ConsumeFromQueue(ctx)
	})
	g.Go(func() error {
		return consumer.ConsumeFromQueue(ctx)
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("service failed: %v", err)
	}

}
