package api

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/dhruvthak3r/Probe/config"
)

func HttpRequestWorkers(ctx context.Context, a *App) {
	go func() {
		for {
			select {
			case req, ok := <-a.RequestChan:
				if !ok {
					fmt.Println("Request channel closed, stopping workers")
					return
				}

				switch req.JobType {
				case "CreateMonitor":
					payload := req.Payload.(CreateMonitorPayload)
					fmt.Println("inserting into db")
					if err := InsertMonitorToDB(ctx, a.DB, payload); err != nil {
						fmt.Printf("Error inserting monitor to DB: %v\n", err)
					}

				}
			case <-ctx.Done():
				fmt.Println("Context cancelled, stopping workers")
				return
			}
		}
	}()
}

func InsertMonitorToDB(ctx context.Context, db *config.DB, payload CreateMonitorPayload) error {

	tx, err := db.Pool.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: false})
	if err != nil {
		return fmt.Errorf("error starting transaction: %v\n", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO monitor (monitor_name,url,frequency_seconds,response_format,http_method,connection_timeout,request_body) VALUES (?,?,?,?,?,?,?)`
	values := []interface{}{
		payload.Name,
		payload.Url,
		payload.FrequencySecs,
		payload.ResponseFormat,
		payload.HttpMethod,
		payload.ConnectionTimeout,
		payload.RequestBody,
	}

	res, err := tx.ExecContext(ctx, query, values...)

	if err != nil {
		return fmt.Errorf("error inserting monitor: %v\n", err)
	}

	newMonitorID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("error getting last insert id: %v\n", err)
	}

	codes := payload.AcceptedStatusCodes
	if len(codes) == 0 {
		codes = []int{200}
	}

	if err := InsertAcceptedStatusCodes(ctx, tx, newMonitorID, codes); err != nil {
		return fmt.Errorf("error inserting accepted status codes: %v\n", err)
	}

	if err := InsertHeaders(ctx, tx, newMonitorID, payload.RequestHeaders, "monitor_request_headers"); err != nil {
		return fmt.Errorf("error inserting request headers: %v\n", err)
	}

	if err := InsertHeaders(ctx, tx, newMonitorID, payload.ResponseHeaders, "monitor_response_headers"); err != nil {
		return fmt.Errorf("error inserting response headers: %v\n", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v\n", err)
	}

	return nil

}

func InsertHeaders(ctx context.Context, tx *sql.Tx, monitorID int64, headers map[string][]string, tableName string) error {

	if len(headers) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(headers))
	args := make([]interface{}, 0, len(headers)*3)
	for key, values := range headers {
		for _, value := range values {
			placeholders = append(placeholders, "(?, ?, ?)")
			args = append(args, monitorID, key, value)
		}
	}

	query := fmt.Sprintf(`INSERT INTO %s (monitor_id, name, value) VALUES %s`, tableName, strings.Join(placeholders, ","))
	_, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error inserting headers: %v\n", err)
	}
	return nil
}

func InsertAcceptedStatusCodes(ctx context.Context, tx *sql.Tx, newMonitorID int64, codes []int) error {

	if len(codes) == 0 {
		return nil
	}

	if len(codes) == 0 {
		codes = []int{200}
	}

	placeholders := make([]string, 0, len(codes))
	args := make([]interface{}, 0, len(codes)*2)
	for _, code := range codes {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, newMonitorID, code)
	}

	acceptedCodesQuery := fmt.Sprintf(`INSERT INTO monitor_accepted_status_codes (monitor_id, status_code) VALUES %s`, strings.Join(placeholders, ","))
	_, err := tx.ExecContext(ctx, acceptedCodesQuery, args...)
	if err != nil {
		return fmt.Errorf("error inserting accepted status codes: %v\n", err)

	}
	return nil
}
