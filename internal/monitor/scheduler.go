package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	db "github.com/dhruvthak3r/Probe/internal/config"
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

	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}

	requestheadersByMonitor, err := GetRequestHeadersForMonitor(ctx, tx, ids, placeholders)
	if err != nil {
		return fmt.Errorf("failed getting request headers..%w", err)
	}

	responseheadersByMonitor, err := GetResponseHeadersForMonitor(ctx, tx, ids, placeholders)
	if err != nil {
		return fmt.Errorf("failed getting response headers..%w", err)
	}

	acceptedCodesByMonitor, err := GetAcceptedStatusCodeForMonitor(ctx, tx, ids, placeholders)
	if err != nil {
		return fmt.Errorf("failed getting status codes..%w", err)
	}

	u_err := UpdateMonitorStatus(ctx, tx, placeholders, ids)
	if u_err != nil {
		return fmt.Errorf("updating monitor status failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
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

func GetHeadersForMonitor(ctx context.Context, tx *sql.Tx, table string, ids []interface{}, placeholders []string) (map[int]map[string][]string, error) {

	query := fmt.Sprintf(`
		SELECT monitor_id, name, value
		FROM %s
		WHERE monitor_id IN (%s)
	`,
		table,
		strings.Join(placeholders, ","),
	)

	rows, err := tx.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, fmt.Errorf("failed getting headers from %s: %w", table, err)
	}
	defer rows.Close()

	headersByMonitor := make(map[int]map[string][]string)

	for rows.Next() {
		var monitorID int
		var key, value string

		if err := rows.Scan(&monitorID, &key, &value); err != nil {
			return nil, err
		}

		if _, ok := headersByMonitor[monitorID]; !ok {
			headersByMonitor[monitorID] = make(map[string][]string)
		}

		headersByMonitor[monitorID][key] = append(headersByMonitor[monitorID][key], value)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return headersByMonitor, nil
}

func GetRequestHeadersForMonitor(ctx context.Context, tx *sql.Tx, ids []interface{}, placeholders []string) (map[int]map[string][]string, error) {

	return GetHeadersForMonitor(
		ctx,
		tx,
		"monitor_request_headers",
		ids,
		placeholders,
	)
}

func GetResponseHeadersForMonitor(ctx context.Context, tx *sql.Tx, ids []interface{}, placeholders []string) (map[int]map[string][]string, error) {

	return GetHeadersForMonitor(
		ctx,
		tx,
		"monitor_response_headers",
		ids,
		placeholders,
	)
}

func GetAcceptedStatusCodeForMonitor(ctx context.Context, tx *sql.Tx, ids []interface{}, placeholders []string) (map[int][]int, error) {
	acceptedstatuscodesQuery := fmt.Sprintf(`SELECT monitor_id,status_code FROM monitor_accepted_status_codes WHERE monitor_id IN (%s)`, strings.Join(placeholders, ","))
	statuscodesRows, err := tx.QueryContext(ctx, acceptedstatuscodesQuery, ids...)

	if err != nil {
		return nil, fmt.Errorf("failed getting status codes..%w", err)
	}
	defer statuscodesRows.Close()

	acceptedCodesByMonitor := make(map[int][]int)

	for statuscodesRows.Next() {
		var monitor_id int
		var statuscode int

		if err := statuscodesRows.Scan(&monitor_id, &statuscode); err != nil {
			return nil, err
		}

		acceptedCodesByMonitor[monitor_id] =
			append(acceptedCodesByMonitor[monitor_id], statuscode)
	}

	if err := statuscodesRows.Err(); err != nil {
		return nil, err
	}

	return acceptedCodesByMonitor, nil

}

func UpdateMonitorStatus(ctx context.Context, tx *sql.Tx, placeholders []string, ids []interface{}) error {
	updateq := fmt.Sprintf(`
        UPDATE monitor
        SET status = "running"
        WHERE monitor_id IN (%s)
        `, strings.Join(placeholders, ","))

	if _, err := tx.ExecContext(ctx, updateq, ids...); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

func GetNextMonitors(ctx context.Context, tx *sql.Tx) ([]*Monitor, []interface{}, error) {
	query := `SELECT monitor_id, url, frequency_seconds, last_run_at, next_run_at, response_format, request_body, http_method,connection_timeout
        FROM monitor 
        WHERE is_active = 1 
        AND is_mock = 1 
		AND next_run_at <= NOW()
		AND (
			status = 'idle'
			OR (status = 'running' AND next_run_at <= DATE_SUB(NOW(), INTERVAL frequency_seconds SECOND))
		)
        ORDER BY next_run_at
        FOR UPDATE SKIP LOCKED`

	rows, err := tx.QueryContext(ctx, query)

	if err != nil {
		return nil, nil, fmt.Errorf("error querying %v\n", err)
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
		var RequestBody sql.NullString
		var HttpMethod string
		var ConnectionTimeout sql.NullInt64

		err := rows.Scan(&ID, &Url, &FrequencySecs, &LastRunAt, &NextRunAt, &ResponseFormat, &RequestBody, &HttpMethod, &ConnectionTimeout)

		if err != nil {
			return nil, nil, fmt.Errorf("error scanning rows: %v\n", err)
		}

		m := NewMonitor(ID, Url, FrequencySecs, LastRunAt, NextRunAt, ResponseFormat, RequestBody, HttpMethod, ConnectionTimeout)

		monitors = append(monitors, m)
		ids = append(ids, ID)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return monitors, ids, nil

}
