package monitor

import (
	"context"
	"dhruv/probe/internal/db"
	"fmt"
	"sync"
	"time"
)

type Monitor struct {
	ID             int
	Url            string
	FrequencySecs  int
	LastRunAt      time.Time
	NextRunAt      time.Time
	ResponseFormat string
	HttpMethod     string
}

type UrlQueue struct {
	UrlsToPoll chan *Monitor
}

func GetNextUrlsToPoll(ctx context.Context, db *db.DB) {

	query := "SELECT monitor_id, monitor_name, url FROM monitor WHERE is_active = 1 AND is_mock = 1 AND next_run_at <= now() ORDER BY next_run_at"
	rows, err := db.Pool.QueryContext(ctx, query)

	if err != nil {
		fmt.Printf("error querying %v\n", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var id int
		var monitor_name string
		var url string
		err := rows.Scan(&id, &monitor_name, &url)

		if err != nil {
			fmt.Printf("error scanning rows: %v\n", err)
		}

		fmt.Printf("id: %d, monitor_name: %s, url: %s\n", id, monitor_name, url)

	}
}

func (uq *UrlQueue) EnqueueMonitors(ctx context.Context, wg *sync.WaitGroup, m []*Monitor) {

	defer wg.Done()

	for _, m := range m {
		select {
		case uq.UrlsToPoll <- m:
			fmt.Println("monitor enqueued")
		case <-ctx.Done():
			fmt.Printf("oops error with context: %v\n", ctx.Err())
		}
	}
}
