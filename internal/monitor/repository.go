package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	db "github.com/dhruvthak3r/Probe/config"
)

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
        SET status = "running",
		last_run_at = NOW()
        WHERE monitor_id IN (%s)
        `, strings.Join(placeholders, ","))

	if _, err := tx.ExecContext(ctx, updateq, ids...); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

func GetNextMonitors(ctx context.Context, tx *sql.Tx) ([]*Monitor, []interface{}, error) {

	query := `SELECT monitor_id, url, frequency_seconds, last_run_at, next_run_at, response_format, request_body, http_method, connection_timeout
              FROM monitor 
              WHERE is_active = 1 
              AND is_mock = 1 
              AND (
                    status = 'idle'
                    OR (
                         status = 'running'
                         AND (
                            last_run_at IS NULL 
                            OR last_run_at <= DATE_SUB(NOW(), INTERVAL frequency_seconds SECOND)
                            )
                        )
                )
              AND next_run_at <= NOW()
              ORDER BY next_run_at
              FOR UPDATE SKIP LOCKED;`

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

func SetStatusToIdle(ctx context.Context, db *db.DB, m *Monitor) error {
	update := `
        UPDATE monitor
        SET next_run_at = DATE_ADD(NOW(), INTERVAL ? SECOND),
            status = 'idle'
        WHERE monitor_id = ?`

	_, err := db.Pool.ExecContext(ctx, update, m.FrequencySecs, m.ID)
	if err != nil {
		return fmt.Errorf("error updating monitor after poll: %w", err)
	}
	return nil
}
