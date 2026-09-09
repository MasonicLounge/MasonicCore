package models

import (
	"time"

	"github.com/google/uuid"
)

// Session is a long-lived refresh session bound to a user.
type Session struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	UserAgent string     `json:"user_agent"`
	IP        *string    `json:"ip"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

// NewSession describes a refresh session being created.
type NewSession struct {
	UserID    uuid.UUID
	TokenHash string
	UserAgent string
	IP        string
	ExpiresAt time.Time
}
