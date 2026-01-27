package main

import (
	"context"
	db "dhruv/probe/internal/config"
	"dhruv/probe/internal/monitor"
	"fmt"
	"log"

	"github.com/go-co-op/gocron/v2"

	//"sync"
	"time"
	//"net/http"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	_ = godotenv.Load("../.env")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := db.NewConnection()
	if err != nil {
		fmt.Printf("error connecting to db: %v\n", err)
		return
	}

	defer conn.Pool.Close()

	urlq := monitor.NewMonitorQueue()

	g, ctx := errgroup.WithContext(ctx)

	s, err := gocron.NewScheduler()

	if err != nil {
		fmt.Printf("error creating scheduler: %v\n", err)
	}

	_, j_err := s.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			urlq.RunScheduler(ctx, conn),
		),
	)

	if j_err != nil {
		fmt.Printf("error creating job: %v\n", err)
	}

	s.Start()

	g.Go(func() error {
		return urlq.PollUrls(ctx, conn)
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("service failed: %v", err)
	}

	monitor.GetResults("https://roadmap.sh/golang")

}
