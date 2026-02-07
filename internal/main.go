package main

import (
	"context"
	"fmt"
	"log"
	"time"

	db "github.com/dhruvthak3r/Probe/config"
	"github.com/dhruvthak3r/Probe/internal/logger"
	"github.com/dhruvthak3r/Probe/internal/monitor"

	"github.com/go-co-op/gocron/v2"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {

	_ = godotenv.Load("../.env")
	err := logger.Init("probe-results.log")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := db.NewDBConnection()
	if err != nil {
		fmt.Printf("error connecting to db: %v\n", err)
		return
	}

	defer conn.Pool.Close()

	monitorq := monitor.NewMonitorQueue()

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
		return monitorq.PollUrls(ctx, conn)
	})

	g.Go(func() error {
		return monitorq.PollUrls(ctx, conn)
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("service failed: %v", err)
	}

}
