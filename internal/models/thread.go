package models

import (
	"time"

	"github.com/google/uuid"
)

// Thread is a discussion started by a user inside a group.
type Thread struct {
	ID         uuid.UUID  `json:"id"`
	GroupID    uuid.UUID  `json:"group_id"`
	AuthorID   uuid.UUID  `json:"author_id"`
	Title      string     `json:"title"`
	Pinned     bool       `json:"pinned"`
	Locked     bool       `json:"locked"`
	Views      int        `json:"views"`
	PostCount  int        `json:"post_count"`
	LastPostAt *time.Time `json:"last_post_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ThreadSummary is a thread joined with its author for API responses.
type ThreadSummary struct {
	Thread
	AuthorUsername string `json:"author_username"`
	AuthorDisplay  string `json:"author_display_name"`
}

// NewThread carries the fields required to create a thread.
type NewThread struct {
	GroupID  uuid.UUID
	AuthorID uuid.UUID
	Title    string
}
