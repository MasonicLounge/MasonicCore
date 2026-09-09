package models

import (
	"time"

	"github.com/google/uuid"
)

// Post is a single message inside a thread.
type Post struct {
	ID         uuid.UUID  `json:"id"`
	ThreadID   uuid.UUID  `json:"thread_id"`
	AuthorID   uuid.UUID  `json:"author_id"`
	Body       string     `json:"body"`
	EditedByID *uuid.UUID `json:"edited_by_id"`
	EditedAt   *time.Time `json:"edited_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// PostSummary is a post joined with its author for API responses.
type PostSummary struct {
	Post
	AuthorUsername string          `json:"author_username"`
	AuthorDisplay  string          `json:"author_display_name"`
	Attachments    []AttachmentWithOwner `json:"attachments"`
}

// NewPost carries the fields required to create a post.
type NewPost struct {
	ThreadID uuid.UUID
	AuthorID uuid.UUID
	Body     string
}
