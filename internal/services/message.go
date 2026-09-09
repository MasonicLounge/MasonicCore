package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

const (
	maxPMBodyLength = 10000
	notifTypePM     = "pm"
)

// MessageService implements private messaging backed by the store.
type MessageService struct {
	messages *store.MessageStore
	users    *store.UserStore
}

// NewMessageService creates a MessageService.
func NewMessageService(messages *store.MessageStore, users *store.UserStore) *MessageService {
	return &MessageService{messages: messages, users: users}
}

// Send delivers a private message.
func (s *MessageService) Send(ctx context.Context, sender, recipient uuid.UUID, body string) (*models.PrivateMessage, error) {
	if sender == recipient {
		return nil, ErrInvalidInput
	}
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > maxPMBodyLength {
		return nil, ErrInvalidInput
	}
	if _, err := s.users.GetByID(ctx, recipient); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return s.messages.Send(ctx, models.NewPrivateMessage{SenderID: sender, RecipientID: recipient, Body: body})
}

// Conversation returns the thread between two users, oldest first.
func (s *MessageService) Conversation(ctx context.Context, a, b uuid.UUID, limit, offset int) ([]models.PMConversation, int, error) {
	return s.messages.Conversation(ctx, a, b, limit, offset)
}

// Inbox returns the latest message per conversation, newest first.
func (s *MessageService) Inbox(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.PMConversation, int, error) {
	return s.messages.Inbox(ctx, userID, limit, offset)
}

// MarkRead marks a message read for its recipient.
func (s *MessageService) MarkRead(ctx context.Context, id, recipient uuid.UUID) error {
	if err := s.messages.MarkRead(ctx, id, recipient); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// UnreadCount returns the number of unread messages for the user.
func (s *MessageService) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.messages.CountUnread(ctx, userID)
}

// NotificationService manages per-user notifications.
type NotificationService struct {
	notifications *store.NotificationStore
}

// NewNotificationService creates a NotificationService.
func NewNotificationService(notifications *store.NotificationStore) *NotificationService {
	return &NotificationService{notifications: notifications}
}

// OnPrivateMessage records a notification for a received PM and returns it.
func (s *NotificationService) OnPrivateMessage(ctx context.Context, msg *models.PrivateMessage) (*models.Notification, error) {
	payload, _ := json.Marshal(map[string]any{
		"message_id": msg.ID,
		"sender_id":  msg.SenderID,
		"preview":    preview(msg.Body),
	})
	n, err := s.notifications.Create(ctx, models.NewNotification{
		UserID:  msg.RecipientID,
		Type:    notifTypePM,
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}
	return n, nil
}

// List returns the user's notifications, newest first.
func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Notification, int, error) {
	return s.notifications.List(ctx, userID, limit, offset)
}

// MarkRead marks one notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.notifications.MarkRead(ctx, id, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// MarkAllRead marks every notification of the user as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.notifications.MarkAllRead(ctx, userID)
}

// UnreadCount returns the number of unread notifications for the user.
func (s *NotificationService) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifications.CountUnread(ctx, userID)
}

func preview(s string) string {
	const maxPreview = 80
	if len(s) <= maxPreview {
		return s
	}
	return s[:maxPreview] + "…"
}
