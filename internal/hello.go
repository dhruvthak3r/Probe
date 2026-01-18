package main

import (
	"context"
	"dhruv/probe/internal/db"
	"dhruv/probe/internal/monitor"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Urls struct {
	url string
}

type UrlQueue struct {
	UrlsToPoll chan *Urls
}

func main() {
	// var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := db.NewConnection()
	if err != nil {
		fmt.Printf("error connecting to db: %v\n", err)
		return
	}

	monitor.GetNextUrlsToPoll(ctx, conn)
	// const workers = 3
	// wg.Add(workers)

	// Urls := []*Urls{
	// 	{url: "https://google.com"},
	// 	{url: "https://github.com"},
	// 	{url: "https://stackoverflow.com"},
	// 	{url: "https://reddit.com"},
	// 	{url: "https://news.ycombinator.com"},
	// 	{url: "https://golang.org"},
	// 	{url: "https://medium.com"},
	// 	{url: "https://twitter.com"},
	// 	{url: "roadmap.sh/golang"},
	// 	{url: "https://linkedin.com"},
	// }

	// urlq := NewUrlQueue()
	// go urlq.EnqueueUrls(ctx, &wg, Urls)

	// for i := 1; i <= workers; i++ {
	// 	go urlq.DeQueueUrls(ctx, &wg)
	// }

	// wg.Wait()

	// db, err := db.NewConnection()
	// if err != nil {
	// 	fmt.Printf("oops: %v\n", err)
	// }

	// rows, err := db.QueryContext(ctx, "SELECT monitor_id,monitor_name FROM monitor")
	// if err != nil {
	// 	fmt.Printf("error querying %v\n", err)
	// }
	// defer rows.Close()

	// for rows.Next() {
	// 	var id int
	// 	var monitor_name string

	// 	err := rows.Scan(&id, &monitor_name)
	// 	if err != nil {
	// 		fmt.Printf("error scanning %v\n", err)
	// 	}

	// 	fmt.Printf("id: %d, monitor_name: %s\n", id, monitor_name)
	// }
	// defer db.Close()
}

func NewUrlQueue() *UrlQueue {
	return &UrlQueue{
		UrlsToPoll: make(chan *Urls),
	}
}

func (uq *UrlQueue) EnqueueUrls(ctx context.Context, wg *sync.WaitGroup, urls []*Urls) {
	defer wg.Done()

	for _, u := range urls {
		select {
		case uq.UrlsToPoll <- u:
			fmt.Println("url Enqueued")

		case <-ctx.Done():
			fmt.Printf("oops: %v\n", ctx.Err())
			return
		}
	}
	close(uq.UrlsToPoll)
}

func (uq *UrlQueue) DeQueueUrls(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case url, ok := <-uq.UrlsToPoll:
			if !ok {
				return
			}

			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url.url, nil)
			resp, err := http.DefaultClient.Do(req)

			if err != nil {
				fmt.Println("error with the url", err)

			}

			fmt.Println("response:", resp)
			resp.Body.Close()

		case <-ctx.Done():
			fmt.Printf("oops: %v\n", ctx.Err())
			return
		}
	}

}

//func GoRoutineEg(ctx context.Context, wg *sync.WaitGroup, ch chan string) {
//defer wg.Done()

//select {
//case ch <- "hello Go Routines":
//fmt.Printf("sent to the channel")
//case <-ctx.Done():
//fmt.Println("oops", ctx.Err())
//}
//close(ch)
//}

//func ReceivFromChannel(ctx context.Context, wg *sync.WaitGroup, ch chan string) {
//defer wg.Done()

//select {
//case mssg := <-ch:
//fmt.Println("received mssg from channel :", mssg)
//case <-ctx.Done():
//fmt.Println("oops", ctx.Err())
//}

//}
