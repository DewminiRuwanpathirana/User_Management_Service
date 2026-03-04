package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Message string `json:"message"`
}

// helper functions to write the HTTP JSON payload with status code
// write a JSON response with the given status code and payload
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode json response", "status_code", statusCode, "error", err)
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

// write an error response with the given status code and message
func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, errorResponse{Message: message})
}
