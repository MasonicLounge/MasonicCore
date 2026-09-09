package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/masoniclounge/masoniccore/internal/db"
)

// Health serves liveness and readiness probes.
type Health struct {
	db *db.DB
}

// NewHealth creates a Health handler.
func NewHealth(d *db.DB) *Health {
	return &Health{db: d}
}

// Health reports the process as alive.
func (h *Health) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready reports whether the service can serve requests (DB reachable).
func (h *Health) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database is not reachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
