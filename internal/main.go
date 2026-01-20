package main

import (
	"context"
	db "dhruv/probe/internal/config"
	"dhruv/probe/internal/monitor"
	"fmt"
	"sync"
	"time"

	//"net/http"
	"github.com/joho/godotenv"
)

type Urls struct {
	url string
}

func main() {
	_ = godotenv.Load("../.env")
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := db.NewConnection()
	if err != nil {
		fmt.Printf("error connecting to db: %v\n", err)
		return
	}

	fmt.Println("db connected:", conn)

	urlq := monitor.NewMonitorQueue()
	urlq.GetNextUrlsToPoll(ctx, &wg, conn)

}
