package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// writeJSON encodes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError encodes an error envelope with a stable machine-readable code.
// Clients localize messages based on the code (see project i18n convention).
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}

// writeServerError logs the unexpected error and responds with a generic 500.
func writeServerError(w http.ResponseWriter, err error) {
	slog.Error("internal server error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}
