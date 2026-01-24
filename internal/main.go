package main

import (
	"context"
	db "dhruv/probe/internal/config"
	"dhruv/probe/internal/monitor"
	"fmt"
	"log"

	//"sync"
	"time"

	//"net/http"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	_ = godotenv.Load("../.env")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := db.NewConnection()
	if err != nil {
		fmt.Printf("error connecting to db: %v\n", err)
		return
	}

	defer conn.Pool.Close()

	urlq := monitor.NewMonitorQueue()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return urlq.GetNextUrlsToPoll(ctx, conn)
	})

	g.Go(func() error {
		return urlq.PollUrls(ctx, conn)
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("service failed: %v", err)
	}

}
