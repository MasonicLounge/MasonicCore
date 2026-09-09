package models

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus values.
const (
	UserStatusActive  = "active"
	UserStatusBanned  = "banned"
	UserStatusPending = "pending"
)

// Role keys matching the seeded roles in migration 00001.
const (
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleMember    = "member"
)

// User is a registered account.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	AvatarURL    string     `json:"avatar_url"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastSeenAt   *time.Time `json:"last_seen_at"`
}

// NewUser describes a user being created.
type NewUser struct {
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
}

// Role is a named permission group.
type Role struct {
	ID          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}
