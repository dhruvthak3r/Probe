package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	handlers "github.com/dhruvthak3r/Probe/api"
	"github.com/dhruvthak3r/Probe/config"
	db "github.com/dhruvthak3r/Probe/config"
	"github.com/dhruvthak3r/Probe/internal/monitor"
	"github.com/dhruvthak3r/Probe/internal/mq"
	"github.com/dhruvthak3r/Probe/migrations"
	"github.com/go-co-op/gocron/v2"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	_ = godotenv.Load()

	conn, err := db.NewDBConnection()
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}
	defer conn.Pool.Close()

	migrations.Run(db.MigrationURL())

	rmqconn, err := config.NewRabbitMQConnection()
	if err != nil {
		log.Fatalf("error connecting to rabbitmq: %v", err)
	}
	defer rmqconn.Close()

	publisher, err := mq.NewRabbitMQPublisher(*rmqconn)
	if err != nil {
		log.Fatalf("error creating rabbitmq publisher: %v", err)
	}
	defer publisher.Close()

	consumer, err := mq.NewConsumer(*rmqconn)
	if err != nil {
		log.Fatalf("error creating rabbitmq consumer: %v", err)
	}
	defer consumer.Close()

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(rootCtx)
	monitorq := monitor.NewMonitorQueue()

	g.Go(func() error {
		<-ctx.Done()
		stop()
		return nil
	})

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("error creating scheduler: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, jErr := s.NewJob(
			gocron.DurationJob(5*time.Second),
			gocron.NewTask(monitorq.RunScheduler(ctx, conn)),
		)
		if jErr != nil {
			log.Fatalf("error creating scheduler job: %v", jErr)
		}
	}
	s.Start()
	defer s.Shutdown()

	for i := 0; i < 10; i++ {
		g.Go(func() error {
			return monitorq.PollUrls(ctx, conn, publisher)
		})
	}

	for i := 0; i < 10; i++ {
		g.Go(func() error {
			return consumer.ConsumeFromQueue(ctx, conn)
		})
	}

	a := &handlers.App{DB: conn}
	mux := http.NewServeMux()
	mux.HandleFunc("/", handlers.HomeHandler)
	mux.HandleFunc("/create-monitor", a.CreateMonitorhandler)
	mux.HandleFunc("/update-monitor", a.UpdateMonitorHandler)
	mux.HandleFunc("/suspend-monitor", a.SuspendMonitorHandler)
	mux.HandleFunc("/get-all-monitors", a.GetAllMonitorsHandler)
	mux.HandleFunc("/get-results", a.GetResultsBetweenTimestampsHandler)
	mux.HandleFunc("/get-metrics", a.GetMetricsBetweenTimestampsHandler)

	srv := &http.Server{Addr: ":8080", Handler: mux}

	g.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("http server error: %w", err)
		}
		return nil
	})

	fmt.Println("combined server running: API on :8080 and scheduler workers active")

	<-rootCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("http shutdown error: %v\n", err)
	}

	if err := g.Wait(); err != nil && rootCtx.Err() == nil {
		log.Fatalf("service failed: %v", err)
	}

	fmt.Println("combined server gracefully stopped")
}
