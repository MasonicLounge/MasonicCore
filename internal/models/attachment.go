package models

import (
	"time"

	"github.com/google/uuid"
)

// Attachment is S3 metadata for a file uploaded to a post.
type Attachment struct {
	ID          uuid.UUID  `json:"id"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	PostID      *uuid.UUID `json:"post_id"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"content_type"`
	SizeBytes   int64      `json:"size_bytes"`
	StorageKey  string     `json:"storage_key"`
	PublicURL   string     `json:"public_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewAttachment carries the columns required to create an attachment row.
type NewAttachment struct {
	OwnerID     uuid.UUID
	PostID      *uuid.UUID
	Filename    string
	ContentType string
	SizeBytes   int64
	StorageKey  string
	PublicURL   string
}

// AttachmentWithOwner embeds Attachment and adds the owner's username.
type AttachmentWithOwner struct {
	Attachment
	OwnerUsername string `json:"owner_username"`
}
