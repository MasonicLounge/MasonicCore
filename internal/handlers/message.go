package handlers

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/services"
	"github.com/masoniclounge/masoniccore/internal/ws"
)

// Messages exposes private messaging endpoints and realtime delivery.
type Messages struct {
	svc   *services.MessageService
	notif *services.NotificationService
	hub   *ws.Hub
}

// NewMessages creates a Messages handler.
func NewMessages(svc *services.MessageService, notif *services.NotificationService, hub *ws.Hub) *Messages {
	return &Messages{svc: svc, notif: notif, hub: hub}
}

type sendMessageRequest struct {
	RecipientID uuid.UUID `json:"recipient_id"`
	Body        string    `json:"body"`
}

// Send delivers a private message, creating a notification and pushing it live.
func (h *Messages) Send(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req sendMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "Invalid request body")
		return
	}
	if req.RecipientID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid recipient id")
		return
	}
	msg, err := h.svc.Send(r.Context(), user.ID, req.RecipientID, req.Body)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	n, err := h.notif.OnPrivateMessage(r.Context(), msg)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if h.hub != nil {
		h.hub.Notify(req.RecipientID, "pm", msg)
		h.hub.Notify(req.RecipientID, "notification", n)
	}
	writeJSON(w, http.StatusCreated, msg)
}

// Conversation returns the private thread between the caller and another user.
func (h *Messages) Conversation(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	other, err := uuid.Parse(r.URL.Query().Get("with"))
	if err != nil || other == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid 'with' query parameter")
		return
	}
	limit, offset := pagination(r)
	items, total, err := h.svc.Conversation(r.Context(), user.ID, other, limit, offset)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

// Inbox returns one latest message per conversation.
func (h *Messages) Inbox(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	limit, offset := pagination(r)
	items, total, err := h.svc.Inbox(r.Context(), user.ID, limit, offset)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

// MarkRead marks a received message as read.
func (h *Messages) MarkRead(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	id, ok := pathID(r, "messageID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid message id")
		return
	}
	if err := h.svc.MarkRead(r.Context(), id, user.ID); err != nil {
		writeCoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnreadCount returns the caller's unread message and notification counts.
func (h *Messages) UnreadCount(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	pm, err := h.svc.UnreadCount(r.Context(), user.ID)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	notif, err := h.notif.UnreadCount(r.Context(), user.ID)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"messages": pm, "notifications": notif})
}

// Notifications lists the caller's notifications, newest first.
func (h *Messages) Notifications(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	limit, offset := pagination(r)
	items, total, err := h.notif.List(r.Context(), user.ID, limit, offset)
	if err != nil {
		writeCoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

// MarkNotificationRead marks one notification as read.
func (h *Messages) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	id, ok := pathID(r, "notificationID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid notification id")
		return
	}
	if err := h.notif.MarkRead(r.Context(), id, user.ID); err != nil {
		writeCoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllNotificationsRead marks every notification of the caller as read.
func (h *Messages) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := authUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := h.notif.MarkAllRead(r.Context(), user.ID); err != nil {
		writeServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Online exposes the currently present user IDs.
func (h *Messages) Online(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil {
		writeJSON(w, http.StatusOK, []uuid.UUID{})
		return
	}
	writeJSON(w, http.StatusOK, h.hub.OnlineUsers())
}

// Presence returns whether a user is online.
func (h *Messages) Presence(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "userID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid user id")
		return
	}
	if h.hub == nil {
		writeJSON(w, http.StatusOK, map[string]any{"user_id": id, "online": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": id, "online": h.hub.IsOnline(id)})
}
