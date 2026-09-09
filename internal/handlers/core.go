package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/services"
)

// authUser returns the authenticated user from the request context or nil.
func authUser(r *http.Request) *auth.UserInfo {
	ui, err := auth.FromContext(r.Context())
	if err != nil {
		return nil
	}
	return ui
}

// pathID parses a UUID routing parameter.
func pathID(r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	return id, err == nil
}

// pagination parses limit/offset query parameters with sensible bounds.
func pagination(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

// writeCoreError maps core service errors to stable HTTP responses.
func writeCoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request")
	case errors.Is(err, services.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Forbidden")
	case errors.Is(err, services.ErrGroupNotFound):
		writeError(w, http.StatusNotFound, "group_not_found", "Group not found")
	case errors.Is(err, services.ErrThreadNotFound):
		writeError(w, http.StatusNotFound, "thread_not_found", "Thread not found")
	case errors.Is(err, services.ErrPostNotFound):
		writeError(w, http.StatusNotFound, "post_not_found", "Post not found")
	case errors.Is(err, services.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Resource not found")
	case errors.Is(err, services.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user_not_found", "User not found")
	case errors.Is(err, services.ErrAttachmentNotFound):
		writeError(w, http.StatusNotFound, "attachment_not_found", "Attachment not found")
	case errors.Is(err, services.ErrFileTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "File is too large")
	case errors.Is(err, services.ErrUnsupportedType):
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_content_type", "File type is not supported")
	case errors.Is(err, services.ErrThreadLocked):
		writeError(w, http.StatusConflict, "thread_locked", "Thread is locked")
	case errors.Is(err, services.ErrNotEmpty):
		writeError(w, http.StatusConflict, "not_empty", "Resource is not empty")
	case errors.Is(err, services.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "Resource already exists")
	default:
		writeServerError(w, err)
	}
}
