package handlers

import (
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/services"
)

// Posts exposes the post REST endpoints.
type Posts struct {
	svc *services.PostService
}

// NewPosts creates a Posts handler.
func NewPosts(svc *services.PostService) *Posts {
	return &Posts{svc: svc}
}

type postRequest struct {
	Body          string   `json:"body"`
	AttachmentIDs []string `json:"attachment_ids"`
}

// ListByThread lists posts of a thread in chronological order.
func (h *Posts) ListByThread(w http.ResponseWriter, r *http.Request) {
	threadID, ok := pathID(r, "threadID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid thread id")
		return
	}
	limit, offset := pagination(r)
	items, total, err := h.svc.ListByThread(r.Context(), threadID, limit, offset)
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

// Create adds a reply to a thread.
func (h *Posts) Create(w http.ResponseWriter, r *http.Request) {
	threadID, ok := pathID(r, "threadID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid thread id")
		return
	}
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req postRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	attachmentIDs, err := parseIDs(req.AttachmentIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid attachment id")
		return
	}
	p, err := h.svc.Create(r.Context(), threadID, user.ID, req.Body, attachmentIDs)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// Update edits a post owned by the caller.
func (h *Posts) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "postID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid post id")
		return
	}
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req postRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	p, err := h.svc.Update(r.Context(), id, user.ID, req.Body)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Delete removes a post.
func (h *Posts) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "postID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid post id")
		return
	}
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := h.svc.Delete(r.Context(), id, user.ID, user.Roles); err != nil {
		writeCoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
