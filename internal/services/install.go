package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

const (
	keyInstalled = "system.installed"
	keyForumName = "forum.name"
)

// ErrAlreadyInstalled is returned when the install wizard runs twice.
var ErrAlreadyInstalled = errors.New("already installed")

// InstallStatus reports whether the forum is set up.
type InstallStatus struct {
	Installed bool    `json:"installed"`
	ForumName *string `json:"forum_name"`
}

// InstallInput describes a first-run setup.
type InstallInput struct {
	ForumName     string
	AdminUsername string
	AdminEmail    string
	AdminPassword string
}

// InstallService bootstraps a forum: admin user plus base settings.
type InstallService struct {
	users    *store.UserStore
	settings *store.SettingsStore
	hasher   *auth.PasswordHasher
}

// NewInstallService builds an InstallService.
func NewInstallService(users *store.UserStore, settings *store.SettingsStore, hasher *auth.PasswordHasher) *InstallService {
	return &InstallService{users: users, settings: settings, hasher: hasher}
}

// Status reports the current installation state.
func (s *InstallService) Status(ctx context.Context) (*InstallStatus, error) {
	st := &InstallStatus{Installed: false}

	raw, err := s.settings.Get(ctx, keyInstalled)
	switch {
	case err == nil:
		var installed bool
		if err := json.Unmarshal(raw, &installed); err != nil {
			return nil, err
		}
		st.Installed = installed
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	if name, err := s.settings.Get(ctx, keyForumName); err == nil {
		var forumName string
		if err := json.Unmarshal(name, &forumName); err != nil {
			return nil, err
		}
		st.ForumName = &forumName
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	return st, nil
}

// Setup creates the first admin and marks the forum installed.
func (s *InstallService) Setup(ctx context.Context, in InstallInput) (*models.User, error) {
	st, err := s.Status(ctx)
	if err != nil {
		return nil, err
	}
	if st.Installed {
		return nil, ErrAlreadyInstalled
	}

	in.ForumName = strings.TrimSpace(in.ForumName)
	in.AdminUsername = strings.TrimSpace(in.AdminUsername)
	in.AdminEmail = strings.TrimSpace(in.AdminEmail)
	if in.ForumName == "" || utf8.RuneCountInString(in.ForumName) > 100 {
		return nil, ErrInvalidInput
	}
	if len(in.AdminPassword) < 8 {
		return nil, ErrInvalidInput
	}
	if len([]rune(in.AdminUsername)) < 3 || len([]rune(in.AdminUsername)) > 32 || !usernameRe.MatchString(in.AdminUsername) {
		return nil, ErrInvalidInput
	}
	if !emailRe.MatchString(in.AdminEmail) {
		return nil, ErrInvalidInput
	}

	hash, err := s.hasher.Hash(in.AdminPassword)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, models.NewUser{
		Username:     in.AdminUsername,
		Email:        in.AdminEmail,
		PasswordHash: hash,
		DisplayName:  in.AdminUsername,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrConflict
		}
		return nil, err
	}
	if err := s.users.AddMemberRole(ctx, user.ID); err != nil {
		return nil, err
	}
	if err := s.users.AssignRole(ctx, user.ID, models.RoleAdmin); err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, keyInstalled, true); err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, keyForumName, in.ForumName); err != nil {
		return nil, err
	}
	return user, nil
}
