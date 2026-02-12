package monitor

import (
	"context"
	"database/sql"
	"fmt"

	db "github.com/dhruvthak3r/Probe/config"
)

type Monitor struct {
	ID                  int
	Url                 string
	FrequencySecs       int
	LastRunAt           sql.NullTime
	NextRunAt           sql.NullTime
	ResponseFormat      string
	HttpMethod          string
	ConnectionTimeout   sql.NullInt64
	RequestHeaders      map[string][]string
	ResponseHeaders     map[string][]string
	AcceptedStatusCodes []int
	RequestBody         sql.NullString
}

type MonitorQueue struct {
	UrlsToPoll chan *Monitor
}

func NewMonitor(ID int, Url string, FrequencySecs int, LastRunAt sql.NullTime, NextRunAt sql.NullTime, ResponseFormat string, RequestBody sql.NullString, HttpMethod string, ConnectionTimeout sql.NullInt64) *Monitor {
	return &Monitor{
		ID:                ID,
		Url:               Url,
		FrequencySecs:     FrequencySecs,
		LastRunAt:         LastRunAt,
		NextRunAt:         NextRunAt,
		ResponseFormat:    ResponseFormat,
		RequestBody:       RequestBody,
		HttpMethod:        HttpMethod,
		ConnectionTimeout: ConnectionTimeout,
	}
}

func NewMonitorQueue() *MonitorQueue {

	return &MonitorQueue{
		UrlsToPoll: make(chan *Monitor, 100),
	}
}

func (mq *MonitorQueue) EnqueueNextMonitorsToChan(ctx context.Context, db *db.DB) error {

	tx, err := db.Pool.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})

	if err != nil {
		return fmt.Errorf("error starting transaction: %v\n", err)
	}

	defer tx.Rollback()

	monitors, ids, err := GetNextMonitors(ctx, tx)
	if err != nil {
		return fmt.Errorf("error querying next monitors :%v\n", err)
	}

	fmt.Printf("no of monitor enqueued %d\n", len(monitors))

	if len(ids) == 0 {
		return tx.Commit()
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}

	err = UpdateMonitorStatus(ctx, tx, placeholders, ids)
	if err != nil {
		return fmt.Errorf("updating monitor status failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	requestheadersByMonitor, err := GetRequestHeadersForMonitor(ctx, db, ids, placeholders)
	if err != nil {
		return fmt.Errorf("failed getting request headers..%w", err)
	}

	responseheadersByMonitor, err := GetResponseHeadersForMonitor(ctx, db, ids, placeholders)
	if err != nil {
		return fmt.Errorf("failed getting response headers..%w", err)
	}

	acceptedCodesByMonitor, err := GetAcceptedStatusCodeForMonitor(ctx, db, ids, placeholders)
	if err != nil {
		return fmt.Errorf("failed getting status codes..%w", err)
	}

	for _, m := range monitors {
		m.RequestHeaders = requestheadersByMonitor[m.ID]
		if m.RequestHeaders == nil {
			m.RequestHeaders = map[string][]string{}
		}

		m.ResponseHeaders = responseheadersByMonitor[m.ID]
		if m.ResponseHeaders == nil {
			m.ResponseHeaders = map[string][]string{}
		}

		m.AcceptedStatusCodes = acceptedCodesByMonitor[m.ID]
		if len(m.AcceptedStatusCodes) == 0 {
			m.AcceptedStatusCodes = []int{200}
		}

		select {
		case mq.UrlsToPoll <- m:
			fmt.Println("monitor enqueued from db")
		case <-ctx.Done():
			return fmt.Errorf("oops error with enqueuing: %v\n", ctx.Err())
		}
	}

	return nil
}
