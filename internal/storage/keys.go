package storage

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

// object column prefixes
const (
	avatarDir = "avatars"
	attachDir = "attachments"
	extMaxLen = 8
)

// sanitizeExt extracts a lowercase extension from a filename and caps its length.
func sanitizeExt(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	if len(ext) > extMaxLen {
		ext = ext[:extMaxLen]
	}
	return ext
}

// AvatarKey builds the storage key for a user avatar.
func AvatarKey(userID uuid.UUID, filename string) string {
	return fmt.Sprintf("%s/%s/%s%s", avatarDir, userID, uuid.NewString(), sanitizeExt(filename))
}

// AttachmentKey builds the storage key for a post attachment.
func AttachmentKey(filename string) string {
	return fmt.Sprintf("%s/%s%s", attachDir, uuid.NewString(), sanitizeExt(filename))
}
