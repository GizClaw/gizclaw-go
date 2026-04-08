package httputil

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the top-level JSON envelope for error responses.
type ErrorBody struct {
	Error ErrorPayload `json:"error"`
}

// ErrorPayload carries the machine-readable code and human-readable message.
type ErrorPayload struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// WriteJSON serialises v as JSON and writes it with the given HTTP status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes a standard error envelope with the given code and message.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{
		Error: ErrorPayload{Code: code, Message: message},
	})
}
