package api

import (
	"encoding/json"
	"net/http"

	db "github.com/dhruvthak3r/Probe/config"
)

type Job struct {
	JobType string
	Payload interface{}
}

type App struct {
	DB          *db.DB
	RequestChan chan Job
}

type CreateMonitorPayload struct {
	Url                 string            `json:"url"`
	FrequencySecs       int               `json:"frequency_secs"`
	ResponseFormat      string            `json:"response_format"`
	HttpMethod          string            `json:"http_method"`
	ConnectionTimeout   int               `json:"connection_timeout"`
	RequestHeaders      map[string]string `json:"request_headers"`
	ResponseHeaders     map[string]string `json:"response_headers"`
	AcceptedStatusCodes []int             `json:"accepted_status_codes"`
	RequestBody         string            `json:"request_body"`
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
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

	job := Job{
		JobType: "CreateMonitor",
		Payload: payload,
	}

	select {
	case a.RequestChan <- job:
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Monitor creation request accepted"))
	case <-r.Context().Done():
		http.Error(w, "Request cancelled", http.StatusRequestTimeout)
	default:
		http.Error(w, "Server is busy, try again later", http.StatusServiceUnavailable)
	}

}
