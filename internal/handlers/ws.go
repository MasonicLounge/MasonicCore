package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WS exposes the realtime endpoint. Authenticates via the access_token query parameter.
func WS(hub *ws.Hub, jwtm *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("access_token")
		if raw == "" {
			writeError(w, http.StatusUnauthorized, "missing_token", "access_token query parameter is required")
			return
		}
		claims, err := jwtm.Parse(raw)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid access token")
			return
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid access token")
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client := ws.NewClient(hub, conn, userID)
		go client.WritePump()
		client.ReadPump(func(c *ws.Client) {
			c.Close()
		})
	}
}
