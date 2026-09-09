package handlers

import (
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/services"
)

// Threads exposes the thread REST endpoints.
type Threads struct {
	svc *services.ThreadService
}

// NewThreads creates a Threads handler.
func NewThreads(svc *services.ThreadService) *Threads {
	return &Threads{svc: svc}
}

type createThreadRequest struct {
	Title         string   `json:"title"`
	Body          string   `json:"body"`
	AttachmentIDs []string `json:"attachment_ids"`
}

type updateThreadRequest struct {
	Title  *string `json:"title"`
	Pinned *bool   `json:"pinned"`
	Locked *bool   `json:"locked"`
}

// ListByGroup lists threads of a group.
func (h *Threads) ListByGroup(w http.ResponseWriter, r *http.Request) {
	groupID, ok := pathID(r, "groupID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid group id")
		return
	}
	limit, offset := pagination(r)
	items, total, err := h.svc.ListByGroup(r.Context(), groupID, limit, offset)
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

// Create starts a thread with its first post.
func (h *Threads) Create(w http.ResponseWriter, r *http.Request) {
	groupID, ok := pathID(r, "groupID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid group id")
		return
	}
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req createThreadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	attachmentIDs, err := parseIDs(req.AttachmentIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid attachment id")
		return
	}
	th, err := h.svc.Create(r.Context(), services.CreateThreadInput{
		GroupID:       groupID,
		AuthorID:      user.ID,
		Title:         req.Title,
		Body:          req.Body,
		AttachmentIDs: attachmentIDs,
	})
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, th)
}

// Get returns a thread and bumps its view counter.
func (h *Threads) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "threadID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid thread id")
		return
	}
	th, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, th)
}

// Update applies title/pin/lock changes.
func (h *Threads) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "threadID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid thread id")
		return
	}
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req updateThreadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
		return
	}
	th, err := h.svc.Update(r.Context(), id, services.UpdateThreadInput{
		Title:  req.Title,
		Pinned: req.Pinned,
		Locked: req.Locked,
	}, user.ID, user.Roles)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, th)
}

// Delete removes a thread.
func (h *Threads) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "threadID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid thread id")
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
