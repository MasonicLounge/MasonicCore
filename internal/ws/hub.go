package ws

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Event is a JSON message pushed to connected clients.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type notify struct {
	userID uuid.UUID
	event  Event
}

// Hub tracks live connections and routes events to their recipients.
// A notify with userID == uuid.Nil is broadcast to every connected client.
type Hub struct {
	mu       sync.RWMutex
	clients  map[uuid.UUID]map[*Client]struct{}
	online   map[uuid.UUID]struct{}
	notifyCh chan notify
}

// NewHub creates an empty Hub. Run must be called to dispatch events.
func NewHub() *Hub {
	return &Hub{
		clients:  make(map[uuid.UUID]map[*Client]struct{}),
		online:   make(map[uuid.UUID]struct{}),
		notifyCh: make(chan notify, 256),
	}
}

// Run dispatches queued events until the context is cancelled.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case n := <-h.notifyCh:
			h.deliver(n)
		}
	}
}

func (h *Hub) deliver(n notify) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if n.userID == uuid.Nil {
		for _, conns := range h.clients {
			for c := range conns {
				h.queue(c, n.event)
			}
		}
		return
	}
	for c := range h.clients[n.userID] {
		h.queue(c, n.event)
	}
}

func (h *Hub) queue(c *Client, ev Event) {
	select {
	case c.send <- ev:
	default:
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*Client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
	_, already := h.online[c.userID]
	h.online[c.userID] = struct{}{}
	h.mu.Unlock()

	if !already {
		h.broadcastPresence(c.userID, true)
	}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	if conns := h.clients[c.userID]; conns != nil {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.clients, c.userID)
		}
	}
	_, stillOnline := h.online[c.userID]
	if stillOnline {
		if conns, ok := h.clients[c.userID]; !ok || len(conns) == 0 {
			delete(h.online, c.userID)
			h.mu.Unlock()
			h.broadcastPresence(c.userID, false)
			return
		}
	}
	h.mu.Unlock()
}

// IsOnline reports whether the user has any active connection.
func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.online[userID]
	return ok
}

// OnlineUsers returns the currently present user IDs.
func (h *Hub) OnlineUsers() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]uuid.UUID, 0, len(h.online))
	for id := range h.online {
		out = append(out, id)
	}
	return out
}

// Notify queues an event for a specific user (non-blocking).
func (h *Hub) Notify(userID uuid.UUID, typ string, data any) {
	h.notifyCh <- notify{userID: userID, event: Event{Type: typ, Data: data}}
}

// Broadcast queues an event for every connected client (non-blocking).
func (h *Hub) Broadcast(typ string, data any) {
	h.notifyCh <- notify{userID: uuid.Nil, event: Event{Type: typ, Data: data}}
}

func (h *Hub) broadcastPresence(userID uuid.UUID, online bool) {
	h.Broadcast("presence", map[string]any{"user_id": userID, "online": online})
}
