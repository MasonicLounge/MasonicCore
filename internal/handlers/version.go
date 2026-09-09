package handlers

import (
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/db"
)

// Version reports backend and database schema versions so the frontend
// can verify compatibility at startup.
type Version struct {
	db      *db.DB
	backend string
}

// NewVersion creates a Version handler.
func NewVersion(d *db.DB, backendVersion string) *Version {
	return &Version{db: d, backend: backendVersion}
}

// Get returns the current component versions.
func (h *Version) Get(w http.ResponseWriter, r *http.Request) {
	dbVersion, err := h.db.SchemaVersion(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "schema_version_failed", "failed to read schema version")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"backend": h.backend,
		"db":      dbVersion,
	})
}
