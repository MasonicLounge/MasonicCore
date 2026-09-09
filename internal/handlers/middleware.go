package handlers

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/auth"
)

// RequireAuth validates the Bearer access token and attaches the user identity.
func RequireAuth(jwtm *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r.Header.Get("Authorization"))
			if raw == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}

			claims, err := jwtm.Parse(raw)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_token", "access token is invalid or expired")
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_token", "access token is invalid or expired")
				return
			}

			ctx := auth.WithUser(r.Context(), &auth.UserInfo{
				ID:       userID,
				Username: claims.Username,
				Roles:    claims.Roles,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}

// RequireRole restricts a route to users holding any of the given roles.
// It must be mounted after RequireAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ui := authUser(r)
			if ui == nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			for _, rl := range ui.Roles {
				for _, want := range roles {
					if rl == want {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
		})
	}
}
