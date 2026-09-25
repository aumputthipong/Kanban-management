package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		// The status is already sent; a failure can only be logged.
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			slog.Warn("encode JSON response", "status", status, "err", err)
		}
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, ErrorResponse{Error: message})
}
