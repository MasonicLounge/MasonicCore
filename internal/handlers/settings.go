package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/store"
)

// PublicSettings exposes read-only forum settings (e.g. the forum name).
type PublicSettings struct {
	settings *store.SettingsStore
}

// NewPublicSettings creates a PublicSettings handler.
func NewPublicSettings(settings *store.SettingsStore) *PublicSettings {
	return &PublicSettings{settings: settings}
}

// Get returns publicly visible forum settings.
func (h *PublicSettings) Get(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"forum_name": nil}
	if raw, err := h.settings.Get(r.Context(), "forum.name"); err == nil {
		var name string
		if json.Unmarshal(raw, &name) == nil && name != "" {
			out["forum_name"] = name
		}
	}
	writeJSON(w, http.StatusOK, out)
}