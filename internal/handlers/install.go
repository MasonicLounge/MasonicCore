package handlers

import (
	"errors"
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/services"
)

// Install exposes the first-run setup endpoints.
type Install struct {
	svc *services.InstallService
}

// NewInstall creates an Install handler.
func NewInstall(svc *services.InstallService) *Install {
	return &Install{svc: svc}
}

type installRequest struct {
	ForumName     string `json:"forum_name"`
	AdminUsername string `json:"admin_username"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

// GetStatus reports whether the forum is already installed.
func (h *Install) GetStatus(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context())
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// Create runs first-time setup.
func (h *Install) Create(w http.ResponseWriter, r *http.Request) {
	var req installRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "Invalid JSON body")
		return
	}

	user, err := h.svc.Setup(r.Context(), services.InstallInput{
		ForumName:     req.ForumName,
		AdminUsername: req.AdminUsername,
		AdminEmail:    req.AdminEmail,
		AdminPassword: req.AdminPassword,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAlreadyInstalled):
			writeError(w, http.StatusConflict, "already_installed", "The forum is already installed")
		case errors.Is(err, services.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "validation_error", "Check the submitted fields")
		case errors.Is(err, services.ErrConflict):
			writeError(w, http.StatusConflict, "user_exists", "That username or email already exists")
		default:
			writeServerError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"installed":  true,
		"forum_name": req.ForumName,
		"user":       user,
	})
}
