package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/config"
	"github.com/masoniclounge/masoniccore/internal/services"
)

// Auth wraps the authentication endpoints.
type Auth struct {
	svc *services.AuthService
	cfg config.Config
}

// NewAuth creates the auth handler set.
func NewAuth(svc *services.AuthService, cfg config.Config) *Auth {
	return &Auth{svc: svc, cfg: cfg}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	AccessToken string   `json:"access_token"`
	User        any      `json:"user"`
	Roles       []string `json:"roles,omitempty"`
}

// Register creates a new member account.
func (h *Auth) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "request body is invalid")
		return
	}

	user, err := h.svc.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrValidation):
			writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, services.ErrUsernameTaken), errors.Is(err, services.ErrEmailTaken):
			writeError(w, http.StatusConflict, "user_exists", "username or email is already registered")
		default:
			writeServerError(w, err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

// Login authenticates a user and issues an access token plus refresh cookie.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "request body is invalid")
		return
	}

	tokens, err := h.svc.Login(r.Context(), req.Identifier, req.Password, r.UserAgent(), clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "username or password is incorrect")
		case errors.Is(err, services.ErrUserNotActive):
			writeError(w, http.StatusForbidden, "account_disabled", "account is not active")
		default:
			writeServerError(w, err)
		}
		return
	}

	h.setRefreshCookie(w, tokens.RefreshToken)
	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: tokens.AccessToken,
		User:        tokens.User,
		Roles:       tokens.Roles,
	})
}

// Refresh rotates the session and mints a fresh access token.
func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cfg.RefreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is missing")
		return
	}

	tokens, err := h.svc.Refresh(r.Context(), cookie.Value, r.UserAgent(), clientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefresh):
			h.clearRefreshCookie(w)
			writeError(w, http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
		case errors.Is(err, services.ErrUserNotActive):
			h.clearRefreshCookie(w)
			writeError(w, http.StatusForbidden, "account_disabled", "account is not active")
		default:
			writeServerError(w, err)
		}
		return
	}

	h.setRefreshCookie(w, tokens.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{
		"access_token": tokens.AccessToken,
	})
}

// Logout revokes the refresh session and clears the cookie.
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cfg.RefreshCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user profile.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	info, err := auth.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	user, roles, err := h.svc.Profile(r.Context(), info.ID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user_not_found", "user does not exist")
			return
		}
		writeServerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": user, "roles": roles})
}

func (h *Auth) setRefreshCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.RefreshCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(h.cfg.RefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Auth) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.RefreshCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
	})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

func clientIP(r *http.Request) string {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}
