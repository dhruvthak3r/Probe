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

	conn, conn_err := db.NewConnection()
	if conn_err != nil {
		fmt.Printf("error connecting to db: %v\n", conn_err)
		return
	}

	defer conn.Pool.Close()

	urlq := monitor.NewMonitorQueue()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return urlq.GetNextUrlsToPoll(ctx, conn)
	})
	g.Go(func() error {
		return urlq.GetNextUrlsToPoll(ctx, conn)
	})

	g.Go(func() error {
		return urlq.PollUrls(ctx)
	})

	if err := g.Wait(); err != nil {
		log.Fatalf("service failed: %v", err)
	}

}
