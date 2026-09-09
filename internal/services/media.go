package services

import (
	"context"
	"errors"
	"io"
	"path"
	"strings"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/config"
	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/storage"
	"github.com/masoniclounge/masoniccore/internal/store"
)

const (
	maxAvatarSize     = 2 << 20  // 2 MiB
	maxAttachmentSize = 20 << 20 // 20 MiB
	maxFilenameLength = 160
)

// MediaService handles avatar and attachment uploads backed by S3 (MinIO).
type MediaService struct {
	attachments *store.AttachmentStore
	users       *store.UserStore
	posts       *store.PostStore
	s3          *storage.S3
	cfg         config.Config
}

// NewMediaService creates a MediaService.
func NewMediaService(attachments *store.AttachmentStore, users *store.UserStore, posts *store.PostStore, s3 *storage.S3, cfg config.Config) *MediaService {
	return &MediaService{attachments: attachments, users: users, posts: posts, s3: s3, cfg: cfg}
}

// UploadAvatar stores an image as the user's avatar and updates the profile.
func (s *MediaService) UploadAvatar(ctx context.Context, userID uuid.UUID, filename, contentType string, size int64, r io.Reader) (*models.User, error) {
	if size <= 0 {
		return nil, ErrInvalidInput
	}
	if size > maxAvatarSize {
		return nil, ErrFileTooLarge
	}
	contentType = strings.TrimSpace(contentType)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, ErrUnsupportedType
	}

	key := storage.AvatarKey(userID, filename)
	if err := s.s3.Upload(ctx, key, r, size, contentType); err != nil {
		return nil, err
	}
	return s.users.UpdateAvatar(ctx, userID, s.mediaURL(key))
}

// UploadAttachment stores a file attached to a post (if postID is given).
func (s *MediaService) UploadAttachment(ctx context.Context, userID uuid.UUID, postID *uuid.UUID, filename, contentType string, size int64, r io.Reader) (*models.Attachment, error) {
	if size <= 0 {
		return nil, ErrInvalidInput
	}
	if size > maxAttachmentSize {
		return nil, ErrFileTooLarge
	}
	if postID != nil {
		if _, err := s.posts.GetByID(ctx, *postID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, ErrPostNotFound
			}
			return nil, err
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	key := storage.AttachmentKey(filename)
	if err := s.s3.Upload(ctx, key, r, size, contentType); err != nil {
		return nil, err
	}
	return s.attachments.Create(ctx, models.NewAttachment{
		OwnerID:     userID,
		PostID:      postID,
		Filename:    sanitizeFilename(filename),
		ContentType: contentType,
		SizeBytes:   size,
		StorageKey:  key,
		PublicURL:   s.mediaURL(key),
	})
}

// DeleteAttachment removes an attachment if the caller is its owner or a moderator.
func (s *MediaService) DeleteAttachment(ctx context.Context, id uuid.UUID, callerID uuid.UUID, roles []string) error {
	a, err := s.attachments.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return ErrAttachmentNotFound
	}
	if err != nil {
		return err
	}
	if a.OwnerID != callerID && !hasRole(roles, models.RoleAdmin, models.RoleModerator) {
		return ErrForbidden
	}
	if err := s.s3.Remove(ctx, a.StorageKey); err != nil {
		return err
	}
	return s.attachments.Delete(ctx, id)
}

func (s *MediaService) mediaURL(key string) string {
	base := strings.TrimRight(s.cfg.MediaBaseURL, "/")
	return base + "/" + s.s3.Bucket() + "/" + key
}

func sanitizeFilename(name string) string {
	name = path.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	if len(name) > maxFilenameLength {
		name = name[:maxFilenameLength]
	}
	return name
}
