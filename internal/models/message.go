package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PrivateMessage is a one-to-one message between two users.
type PrivateMessage struct {
	ID          uuid.UUID  `json:"id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	RecipientID uuid.UUID  `json:"recipient_id"`
	Body        string     `json:"body"`
	ReadAt      *time.Time `json:"read_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NewPrivateMessage carries the columns required to insert a private message.
type NewPrivateMessage struct {
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Body        string
}

// PMConversation adds sender display info to a private message.
type PMConversation struct {
	PrivateMessage
	SenderUsername string `json:"sender_username"`
	SenderDisplay  string `json:"sender_display"`
}

// Notification is a per-user event notification delivered via the app.
type Notification struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	ReadAt    *time.Time      `json:"read_at"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewNotification carries the columns required to insert a notification.
type NewNotification struct {
	UserID  uuid.UUID
	Type    string
	Payload json.RawMessage
}
