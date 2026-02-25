package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	db "github.com/dhruvthak3r/Probe/config"
)

type App struct {
	DB *db.DB
}

type CreateMonitorPayload struct {
	Name                string              `json:"name"`
	Url                 string              `json:"url"`
	FrequencySecs       int                 `json:"frequency_secs"`
	ResponseFormat      string              `json:"response_format"`
	HttpMethod          string              `json:"http_method"`
	ConnectionTimeout   int                 `json:"connection_timeout"`
	RequestHeaders      map[string][]string `json:"request_headers"`
	ResponseHeaders     map[string][]string `json:"response_headers"`
	AcceptedStatusCodes []int               `json:"accepted_status_codes"`
	RequestBody         string              `json:"request_body"`
}

type UpdateMonitorPayload struct {
	MonitorID           int                  `json:"monitor_id"`
	Name                *string              `json:"name,omitempty"`
	Url                 *string              `json:"url,omitempty"`
	FrequencySecs       *int                 `json:"frequency_secs,omitempty"`
	ResponseFormat      *string              `json:"response_format,omitempty"`
	HttpMethod          *string              `json:"http_method,omitempty"`
	ConnectionTimeout   *int                 `json:"connection_timeout,omitempty"`
	RequestHeaders      *map[string][]string `json:"request_headers,omitempty"`
	ResponseHeaders     *map[string][]string `json:"response_headers,omitempty"`
	AcceptedStatusCodes *[]int               `json:"accepted_status_codes,omitempty"`
	RequestBody         *string              `json:"request_body,omitempty"`
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the Probe API!"))
}

func (a *App) CreateMonitorhandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload CreateMonitorPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := InsertMonitorToDB(r.Context(), a.DB, payload); err != nil {
		log.Printf("error inserting to db %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	log.Println("inserting to db")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "monitor created successfully",
	})
}

func (a *App) UpdateMonitorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload UpdateMonitorPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if payload.MonitorID <= 0 {
		http.Error(w, "monitor_id is required", http.StatusBadRequest)
		return
	}

	if payload.Name == nil &&
		payload.Url == nil &&
		payload.FrequencySecs == nil &&
		payload.ResponseFormat == nil &&
		payload.HttpMethod == nil &&
		payload.ConnectionTimeout == nil &&
		payload.RequestHeaders == nil &&
		payload.ResponseHeaders == nil &&
		payload.AcceptedStatusCodes == nil &&
		payload.RequestBody == nil {
		http.Error(w, "no fields provided for update", http.StatusBadRequest)
		return
	}

	if err := UpdateMonitorInDB(r.Context(), a.DB, payload); err != nil {
		log.Printf("error updating monitor %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	log.Println("updating to db")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "monitor updated successfully",
	})

}

func (a *App) GetAllMonitorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusBadRequest)
		return
	}

	monitors, err := GetAllMonitors(r.Context(), a.DB)
	if err != nil {
		log.Printf("error fetching monitors: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"monitors": monitors,
	})
}

func (a *App) GetResultsBetweenTimestampsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	monitorIDStr := r.URL.Query().Get("monitor_id")
	if monitorIDStr == "" {
		http.Error(w, "monitor_id is required", http.StatusBadRequest)
		return
	}
	monitorID, err := strconv.Atoi(monitorIDStr)
	if err != nil || monitorID <= 0 {
		http.Error(w, "monitor_id must be a positive integer", http.StatusBadRequest)
		return
	}

	fromTSStr := r.URL.Query().Get("from_ts")
	toTSStr := r.URL.Query().Get("to_ts")
	if fromTSStr == "" || toTSStr == "" {
		http.Error(w, "from_ts and to_ts are required", http.StatusBadRequest)
		return
	}

	fromTS, err := parseTimestamp(fromTSStr)
	if err != nil {
		http.Error(w, "from_ts must be unix seconds or RFC3339", http.StatusBadRequest)
		return
	}

	toTS, err := parseTimestamp(toTSStr)
	if err != nil {
		http.Error(w, "to_ts must be unix seconds or RFC3339", http.StatusBadRequest)
		return
	}

	if fromTS.After(toTS) {
		http.Error(w, "from_ts must be before or equal to to_ts", http.StatusBadRequest)
		return
	}

	limit := 10
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	results, err := GetResultsBetweenTimestamps(r.Context(), a.DB, monitorID, fromTS, toTS, limit)
	if err != nil {
		log.Printf("error fetching results for monitor_id=%d from %s to %s: %v", monitorID, fromTS.UTC().Format(time.RFC3339), toTS.UTC().Format(time.RFC3339), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"monitor_id": monitorID,
		"from_ts":    fromTS.UTC().Format(time.RFC3339),
		"to_ts":      toTS.UTC().Format(time.RFC3339),
		"count":      len(results),
		"results":    results,
	})
}

func (a *App) GetMetricsBetweenTimestampsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	monitorIDStr := r.URL.Query().Get("monitor_id")
	if monitorIDStr == "" {
		http.Error(w, "monitor_id is required", http.StatusBadRequest)
		return
	}
	monitorID, err := strconv.Atoi(monitorIDStr)
	if err != nil || monitorID <= 0 {
		http.Error(w, "monitor_id must be a positive integer", http.StatusBadRequest)
		return
	}

	fromTSStr := r.URL.Query().Get("from_ts")
	toTSStr := r.URL.Query().Get("to_ts")
	if fromTSStr == "" || toTSStr == "" {
		http.Error(w, "from_ts and to_ts are required", http.StatusBadRequest)
		return
	}

	fromTS, err := parseTimestamp(fromTSStr)
	if err != nil {
		http.Error(w, "from_ts must be unix seconds or RFC3339", http.StatusBadRequest)
		return
	}

	toTS, err := parseTimestamp(toTSStr)
	if err != nil {
		http.Error(w, "to_ts must be unix seconds or RFC3339", http.StatusBadRequest)
		return
	}

	if fromTS.After(toTS) {
		http.Error(w, "from_ts must be before or equal to to_ts", http.StatusBadRequest)
		return
	}

	limit := 10
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	metrics, err := GetMetricsBetweenTimestamps(r.Context(), a.DB, monitorID, fromTS, toTS, limit)
	if err != nil {
		log.Printf("error fetching metrics for monitor_id=%d from %s to %s: %v", monitorID, fromTS.UTC().Format(time.RFC3339), toTS.UTC().Format(time.RFC3339), err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"monitor_id": monitorID,
		"from_ts":    fromTS.UTC().Format(time.RFC3339),
		"to_ts":      toTS.UTC().Format(time.RFC3339),
		"count":      len(metrics),
		"results":    metrics,
	})
}

func (a *App) SuspendMonitorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	monitorIDStr := r.URL.Query().Get("monitor_id")
	if monitorIDStr == "" {
		http.Error(w, "monitor_id is required", http.StatusBadRequest)
		return
	}
	monitorID, err := strconv.Atoi(monitorIDStr)
	if err != nil || monitorID <= 0 {
		http.Error(w, "monitor_id must be a positive integer", http.StatusBadRequest)
		return
	}

	err = SuspendMonitor(r.Context(), a.DB, monitorID)
	if err != nil {
		log.Printf("error suspending monitor=%d,err=%d", monitorID, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "monitor suspended successfully",
	})
}
