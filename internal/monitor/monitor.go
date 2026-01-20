package monitor

import (
	"context"
	db "dhruv/probe/internal/config"
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

type MonitorQueue struct {
	UrlsToPoll chan *Monitor
}

func NewMonitor(ID int, Url string, FrequencySecs int, LastRunAt time.Time, NextRunAt time.Time, ResponseFormat string, HttpMethod string) *Monitor {
	return &Monitor{
		ID:             ID,
		Url:            Url,
		FrequencySecs:  FrequencySecs,
		LastRunAt:      LastRunAt,
		NextRunAt:      NextRunAt,
		ResponseFormat: ResponseFormat,
		HttpMethod:     HttpMethod,
	}
}

func NewMonitorQueue() *MonitorQueue {
	return &MonitorQueue{
		UrlsToPoll: make(chan *Monitor),
	}
}

func (uq *MonitorQueue) GetNextUrlsToPoll(ctx context.Context, wg *sync.WaitGroup, db *db.DB) error {
	defer wg.Done()

	query := "SELECT monitor_id, monitor_name, url, frequency_seconds, last_run_at, next_run_at, response_format, http_method FROM monitor WHERE is_active = 1 AND is_mock = 1 AND next_run_at <= NOW() ORDER BY next_run_at"
	rows, err := db.Pool.QueryContext(ctx, query)

	if err != nil {
		return fmt.Errorf("error querying %v\n", err)
	}

	defer rows.Close()

	for rows.Next() {
		var ID int
		var Url string
		var FrequencySecs int
		var LastRunAt time.Time
		var NextRunAt time.Time
		var ResponseFormat string
		var HttpMethod string

		err := rows.Scan(&ID, &Url, &FrequencySecs, &LastRunAt, &NextRunAt, &ResponseFormat, &HttpMethod)

		if err != nil {
			return fmt.Errorf("error scanning rows: %v\n", err)
		}

		m := NewMonitor(ID, Url, FrequencySecs, LastRunAt, NextRunAt, ResponseFormat, HttpMethod)

		select {
		case uq.UrlsToPoll <- m:
			fmt.Println("monitor enqueued from db")
		case <-ctx.Done():
			return fmt.Errorf("oops error with context: %v\n", ctx.Err())
		}

	}

	close(uq.UrlsToPoll)

	return nil
}
