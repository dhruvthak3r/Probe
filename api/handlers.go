package api

import (
	"encoding/json"
	"log"
	"net/http"

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
