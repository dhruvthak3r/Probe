package monitor

import (
	"context"
	"database/sql"
	db "dhruv/probe/internal/config"
	"fmt"
	//"time"
)

type Monitor struct {
	ID             int
	Url            string
	FrequencySecs  int
	LastRunAt      sql.NullTime
	NextRunAt      sql.NullTime
	ResponseFormat string
	HttpMethod     string
}

type MonitorQueue struct {
	UrlsToPoll chan *Monitor
}

func NewMonitor(ID int, Url string, FrequencySecs int, LastRunAt sql.NullTime, NextRunAt sql.NullTime, ResponseFormat string, HttpMethod string) *Monitor {
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

func (mq *MonitorQueue) GetNextUrlsToPoll(ctx context.Context, db *db.DB) error {

	query := "SELECT monitor_id, url, frequency_seconds, last_run_at, next_run_at, response_format, http_method FROM monitor WHERE is_active = 1 AND is_mock = 1 AND next_run_at <= NOW() ORDER BY next_run_at"
	rows, err := db.Pool.QueryContext(ctx, query)

	if err != nil {
		return fmt.Errorf("error querying %v\n", err)
	}

	defer rows.Close()

	for rows.Next() {
		var ID int
		var Url string
		var FrequencySecs int
		var LastRunAt sql.NullTime
		var NextRunAt sql.NullTime
		var ResponseFormat string
		var HttpMethod string

		err := rows.Scan(&ID, &Url, &FrequencySecs, &LastRunAt, &NextRunAt, &ResponseFormat, &HttpMethod)

		if err != nil {
			return fmt.Errorf("error scanning rows: %v\n", err)
		}

		m := NewMonitor(ID, Url, FrequencySecs, LastRunAt, NextRunAt, ResponseFormat, HttpMethod)

		select {
		case mq.UrlsToPoll <- m:
			fmt.Println("monitor enqueued from db")
		case <-ctx.Done():
			return fmt.Errorf("oops error with enqueuing: %v\n", ctx.Err())
		}

	}

	return nil
}

func (mq *MonitorQueue) PollUrls(ctx context.Context) error {
	for {
		select {
		case monitor, ok := <-mq.UrlsToPoll:
			if !ok {

				return nil
			}

			fmt.Printf(
				"Polling URL: %s (Monitor ID: %d)\n",
				monitor.Url,
				monitor.ID,
			)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
