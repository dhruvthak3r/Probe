package monitor

import (
	//"github.com/go-co-op/gocron/v2"
	"context"
	"fmt"

	db "github.com/dhruvthak3r/Probe/internal/config"
)

func (mq *MonitorQueue) RunScheduler(
	ctx context.Context,
	db *db.DB,
) func() {

	return func() {
		fmt.Println("scheduler running")

		if err := mq.EnqueueNextMonitorsToChan(ctx, db); err != nil {
			fmt.Printf("scheduler error: %v\n", err)
		}
	}
}
