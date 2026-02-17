package api

import (
	"context"
	"fmt"
)

func HttpRequestWorkers(ctx context.Context, a *App) {
	go func() {
		for {
			select {
			case req, ok := <-a.RequestChan:
				if !ok {
					fmt.Println("Request channel closed, stopping workers")
					return
				}

				switch req.JobType {
				case "CreateMonitor":
					payload := req.Payload.(CreateMonitorPayload)
					fmt.Printf("Processing CreateMonitor job: %+v\n", payload)
				}
			case <-ctx.Done():
				fmt.Println("Context cancelled, stopping workers")
				return
			}
		}
	}()
}
