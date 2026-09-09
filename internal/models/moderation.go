package models

import (
	"time"

	"github.com/google/uuid"
)

// Moderation log action names. Target type is "thread" or "post".
const (
	ModActionThreadPin    = "thread_pin"
	ModActionThreadUnpin  = "thread_unpin"
	ModActionThreadLock   = "thread_lock"
	ModActionThreadUnlock = "thread_unlock"
	ModActionThreadDelete = "thread_delete"
	ModActionPostDelete   = "post_delete"

	ModTargetThread = "thread"
	ModTargetPost   = "post"
)

// ModerationEntry is a single record in the moderation log.
type ModerationEntry struct {
	ID                uuid.UUID `json:"id"`
	ModeratorID       uuid.UUID `json:"moderator_id"`
	Action            string    `json:"action"`
	TargetType        string    `json:"target_type"`
	TargetID          uuid.UUID `json:"target_id"`
	Reason            string    `json:"reason"`
	CreatedAt         time.Time `json:"created_at"`
	ModeratorUsername string    `json:"moderator_username"`
	ModeratorDisplay  string    `json:"moderator_display"`
}
