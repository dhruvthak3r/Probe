package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	handlers "github.com/dhruvthak3r/Probe/api"
	db "github.com/dhruvthak3r/Probe/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	conn, err := db.NewDBConnection()
	if err != nil {
		panic(err)
	}
	defer conn.Pool.Close()

	a := &handlers.App{
		DB: conn,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/create-monitor", a.CreateMonitorhandler)
	http.HandleFunc("/update-monitor", a.UpdateMonitorHandler)
	http.HandleFunc("/get-all-monitors", a.GetAllMonitorsHandler)

	srv := &http.Server{Addr: ":8080", Handler: nil}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()
	fmt.Println("API server running on :8080")

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	fmt.Println("API server gracefully stopped")

}
