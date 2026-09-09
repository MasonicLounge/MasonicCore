package handlers

import (
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/services"
)

// Admin exposes administrator endpoints.
type Admin struct {
	svc *services.AdminService
}

// NewAdmin creates an Admin handler.
func NewAdmin(svc *services.AdminService) *Admin {
	return &Admin{svc: svc}
}

type updateUserRequest struct {
	Roles  *[]string `json:"roles"`
	Status *string   `json:"status"`
}

type updateSettingsRequest struct {
	ForumName string `json:"forum_name"`
}

// ListUsers returns a page of users with their roles.
func (h *Admin) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	items, total, err := h.svc.ListUsers(r.Context(), limit, offset)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateUser changes roles and/or status of a user.
func (h *Admin) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "userID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid user id")
		return
	}
	var req updateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "Malformed request body")
		return
	}
	if req.Roles == nil && req.Status == nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Provide roles or status")
		return
	}
	u, err := h.svc.UpdateUser(r.Context(), id, req.Roles, req.Status)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// GetSettings returns the forum settings.
func (h *Admin) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.GetSettings(r.Context())
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// UpdateSettings persists the forum settings.
func (h *Admin) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "Malformed request body")
		return
	}
	s, err := h.svc.UpdateSettings(r.Context(), services.ForumSettings{ForumName: req.ForumName})
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ListMedia returns a page of attachments with owner usernames.
func (h *Admin) ListMedia(w http.ResponseWriter, r *http.Request) {
	limit, offset := pagination(r)
	items, total, err := h.svc.ListMedia(r.Context(), limit, offset)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
