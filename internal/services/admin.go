package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

var (
	allowedRoles  = map[string]bool{models.RoleAdmin: true, models.RoleModerator: true, models.RoleMember: true}
	allowedStatus = map[string]bool{models.UserStatusActive: true, models.UserStatusBanned: true, models.UserStatusPending: true}
	maxForumName  = 100
)

// AdminService provides administrator operations: user and role management,
// forum settings, media administration and the moderation log.
type AdminService struct {
	users       *store.UserStore
	attachments *store.AttachmentStore
	settings    *store.SettingsStore
	moderation  *store.ModerationStore
}

// NewAdminService creates an AdminService.
func NewAdminService(users *store.UserStore, attachments *store.AttachmentStore, settings *store.SettingsStore, moderation *store.ModerationStore) *AdminService {
	return &AdminService{users: users, attachments: attachments, settings: settings, moderation: moderation}
}

// ListUsers returns a page of users with their roles.
func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]*models.UserWithRoles, int64, error) {
	return s.users.List(ctx, limit, offset)
}

// UpdateUser applies role and/or status changes. Returns the updated user.
func (s *AdminService) UpdateUser(ctx context.Context, id uuid.UUID, roles *[]string, status *string) (*models.UserWithRoles, error) {
	if status != nil {
		if !allowedStatus[*status] {
			return nil, ErrInvalidInput
		}
		if _, err := s.users.UpdateStatus(ctx, id, *status); err != nil {
			return nil, mapUserErr(err)
		}
	}
	if roles != nil {
		if len(*roles) == 0 {
			return nil, ErrInvalidInput
		}
		for _, r := range *roles {
			if !allowedRoles[strings.ToLower(r)] {
				return nil, ErrInvalidInput
			}
		}
		keys := make([]string, 0, len(*roles))
		for _, r := range *roles {
			keys = append(keys, strings.ToLower(r))
		}
		if err := s.users.UpdateRoles(ctx, id, keys); err != nil {
			return nil, mapUserErr(err)
		}
	}

	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, mapUserErr(err)
	}
	rolesList, err := s.users.RolesForUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.UserWithRoles{User: *u, Roles: rolesList}, nil
}

// ListMedia returns a page of attachments with owner usernames.
func (s *AdminService) ListMedia(ctx context.Context, limit, offset int) ([]*models.AttachmentWithOwner, int64, error) {
	return s.attachments.List(ctx, limit, offset)
}

// ListModerationLog returns a page of moderation log entries with moderator names.
func (s *AdminService) ListModerationLog(ctx context.Context, limit, offset int) ([]models.ModerationEntry, int, error) {
	return s.moderation.List(ctx, limit, offset)
}

// ForumSettings holds editable forum configuration.
type ForumSettings struct {
	ForumName string `json:"forum_name"`
}

// GetSettings returns the current forum settings.
func (s *AdminService) GetSettings(ctx context.Context) (*ForumSettings, error) {
	out := &ForumSettings{}
	if raw, err := s.settings.Get(ctx, keyForumName); err == nil {
		var name string
		if json.Unmarshal(raw, &name) == nil {
			out.ForumName = name
		}
	}
	return out, nil
}

// UpdateSettings validates and persists forum settings.
func (s *AdminService) UpdateSettings(ctx context.Context, in ForumSettings) (*ForumSettings, error) {
	name := strings.TrimSpace(in.ForumName)
	if name == "" || utf8.RuneCountInString(name) > maxForumName {
		return nil, ErrInvalidInput
	}
	out := &ForumSettings{ForumName: name}
	if err := s.settings.Set(ctx, keyForumName, name); err != nil {
		return nil, err
	}
	return out, nil
}

func mapUserErr(err error) error {
	if errors.Is(err, store.ErrNotFound) {
		return ErrUserNotFound
	}
	if errors.Is(err, store.ErrConflict) {
		return ErrConflict
	}
	return err
}
