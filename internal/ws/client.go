package ws

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4 << 10 // 4 KiB
	// sendBuffer bounds queued events per connection.
	sendBuffer = 64
)

// Client represents one authenticated WebSocket connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan Event
	userID uuid.UUID
}

// NewClient wraps a connection for the user and registers it with the hub.
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID) *Client {
	c := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan Event, sendBuffer),
		userID: userID,
	}
	hub.register(c)
	return c
}

// Close unregisters the client and closes its output channel.
func (c *Client) Close() {
	c.hub.unregister(c)
	close(c.send)
}

// UserID returns the authenticated user of this connection.
func (c *Client) UserID() uuid.UUID {
	return c.userID
}

// ReadPump forwards incoming frames; it blocks until the connection closes.
// The onClose callback lets the caller finalize connection teardown.
func (c *Client) ReadPump(onClose func(*Client)) {
	defer func() {
		c.conn.Close()
		if onClose != nil {
			onClose(c)
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

// WritePump writes queued events and keep-alive pings until the connection closes.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case ev, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(ev); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
