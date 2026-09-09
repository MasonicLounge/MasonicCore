package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/services"
)

// Groups exposes the group REST endpoints.
type Groups struct {
	svc *services.GroupService
}

// NewGroups creates a Groups handler.
func NewGroups(svc *services.GroupService) *Groups {
	return &Groups{svc: svc}
}

type groupRequest struct {
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
	SortOrder   int        `json:"sort_order"`
}

// List groups for display.
func (h *Groups) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.svc.List(r.Context())
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": groups})
}

// Create a new group. Admin only.
func (h *Groups) Create(w http.ResponseWriter, r *http.Request) {
	var req groupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	g, err := h.svc.Create(r.Context(), services.GroupInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

// Get a single group.
func (h *Groups) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "groupID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid group id")
		return
	}
	g, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// Update a group. Admin only.
func (h *Groups) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "groupID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid group id")
		return
	}
	var req groupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	g, err := h.svc.Update(r.Context(), id, services.GroupInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// Delete a group. Admin only.
func (h *Groups) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "groupID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid group id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeCoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
