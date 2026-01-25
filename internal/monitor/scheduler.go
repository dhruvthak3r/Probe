package monitor

import (
	//"github.com/go-co-op/gocron/v2"
	"context"
	db "dhruv/probe/internal/config"
	"fmt"
)

func (mq *MonitorQueue) RunScheduler(
	ctx context.Context,
	db *db.DB,
) func() {

	return func() {
		fmt.Println("scheduler running")

		if err := mq.GetNextUrlsToPoll(ctx, db); err != nil {
			fmt.Printf("scheduler error: %v\n", err)
		}
	}
}
