package monitor

import (
	"context"
	"fmt"

	db "github.com/dhruvthak3r/Probe/config"
)

func (mq *MonitorQueue) RunScheduler(
	ctx context.Context,
	db *db.DB,
) func() {

	return func() {

		if err := mq.EnqueueNextMonitorsToChan(ctx, db); err != nil {
			fmt.Printf("scheduler error: %v\n", err)
		}
	}
}
