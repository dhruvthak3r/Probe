package monitor

import (
	"context"
	"database/sql"
	db "dhruv/probe/internal/config"
	"fmt"
	"strings"
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
		UrlsToPoll: make(chan *Monitor, 100),
	}
}

func (mq *MonitorQueue) GetNextUrlsToPoll(ctx context.Context, db *db.DB) error {

	tx, tx_err := db.Pool.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})

	if tx_err != nil {
		return fmt.Errorf("error starting transaction: %v\n", tx_err)
	}

	defer tx.Rollback()

	query := `SELECT monitor_id, url, frequency_seconds, last_run_at, next_run_at, response_format, http_method
        FROM monitor 
        WHERE is_active = 1 
        AND is_mock = 1 
		AND status = 'idle'
        AND next_run_at <= NOW()
        ORDER BY next_run_at
        FOR UPDATE SKIP LOCKED`

	rows, err := tx.QueryContext(ctx, query)

	if err != nil {
		return fmt.Errorf("error querying %v\n", err)
	}

	defer rows.Close()

	var (
		monitors []*Monitor
		ids      []interface{}
	)

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

		monitors = append(monitors, m)
		ids = append(ids, ID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if len(ids) > 0 {

		placeholders := make([]string, len(ids))
		for i := range ids {
			placeholders[i] = "?"
		}

		updateq := fmt.Sprintf(`
        UPDATE monitor
        SET status = "running"
        WHERE monitor_id IN (%s)
        `, strings.Join(placeholders, ","))

		if _, err := tx.ExecContext(ctx, updateq, ids...); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	for _, m := range monitors {
		select {
		case mq.UrlsToPoll <- m:
			fmt.Println("monitor enqueued from db")
		case <-ctx.Done():
			return fmt.Errorf("oops error with enqueuing: %v\n", ctx.Err())
		}
	}

	return nil
}

func (mq *MonitorQueue) PollUrls(ctx context.Context, db *db.DB) error {

	for {
		select {
		case m, ok := <-mq.UrlsToPoll:
			if !ok {

				return nil
			}

			fmt.Printf(
				"Polling URL: %s (Monitor ID: %d)\n",
				m.Url,
				m.ID,
			)

			update := `
                UPDATE monitor
                SET last_run_at = NOW(),
                next_run_at = DATE_ADD(NOW(), INTERVAL ? SECOND),
                status = 'idle'
                WHERE monitor_id = ?`

			_, err := db.Pool.ExecContext(ctx, update, m.FrequencySecs, m.ID)
			if err != nil {
				return fmt.Errorf("error updating monitor after poll: %v\n", err)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
