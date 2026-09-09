package models

import (
	"time"

	"github.com/google/uuid"
)

// Group represents a category or forum that threads belong to.
type Group struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewGroup carries the fields required to create a group.
type NewGroup struct {
	Name        string
	Slug        string
	Description string
	ParentID    *uuid.UUID
	SortOrder   int
}
