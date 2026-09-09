package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/config"
	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

// Auth errors. Handlers map them to stable error codes.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrUserNotActive      = errors.New("user is not active")
	ErrUserNotFound       = errors.New("user not found")
	ErrValidation         = errors.New("validation failed")
	ErrUsernameTaken      = errors.New("username taken")
	ErrEmailTaken         = errors.New("email taken")
	ErrWrongPassword      = errors.New("wrong password")
)

// Tokens is the result of a successful login or refresh.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	User         *models.User
	Roles        []string
}

var (
	usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
	emailRe    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// AuthService orchestrates registration, login, refresh and logout.
type AuthService struct {
	users    *store.UserStore
	sessions *store.SessionStore
	hasher   *auth.PasswordHasher
	jwt      *auth.Manager
	cfg      config.Config
}

// NewAuthService builds an AuthService.
func NewAuthService(users *store.UserStore, sessions *store.SessionStore, hasher *auth.PasswordHasher, jwt *auth.Manager, cfg config.Config) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
		jwt:      jwt,
		cfg:      cfg,
	}
}

// Register creates a new member account.
func (s *AuthService) Register(ctx context.Context, username, email, password string) (*models.User, error) {
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)

	if !usernameRe.MatchString(username) {
		return nil, fmt.Errorf("%w: username must be 3-32 chars [a-zA-Z0-9_]", ErrValidation)
	}
	if !emailRe.MatchString(email) {
		return nil, fmt.Errorf("%w: invalid email", ErrValidation)
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 chars", ErrValidation)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, models.NewUser{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		DisplayName:  username,
	})
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrUsernameTaken // or ErrEmailTaken; both map to conflict
		}
		return nil, err
	}

	if err := s.users.AddMemberRole(ctx, user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

// Login verifies credentials and issues an access token plus a refresh session.
func (s *AuthService) Login(ctx context.Context, identifier, password, userAgent, ip string) (*Tokens, error) {
	identifier = strings.TrimSpace(identifier)

	user, err := s.users.GetByUsername(ctx, identifier)
	if errors.Is(err, store.ErrNotFound) {
		user, err = s.users.GetByEmail(ctx, identifier)
		if err != nil {
			return nil, ErrInvalidCredentials
		}
	} else if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != models.UserStatusActive {
		return nil, ErrUserNotActive
	}

	valid, err := s.hasher.Verify(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user, userAgent, ip)
}

// Refresh rotates a refresh session and mints a fresh access token.
// Returns the new tokens and the session's user.
func (s *AuthService) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (*Tokens, error) {
	hash, err := auth.HashRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefresh
	}

	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			_ = s.sessions.Revoke(ctx, session.ID, time.Now())
			return nil, ErrInvalidRefresh
		}
		return nil, err
	}
	if user.Status != models.UserStatusActive {
		return nil, ErrUserNotActive
	}

	// Rotate: revoke the presented session, create a fresh one.
	if err := s.sessions.Revoke(ctx, session.ID, time.Now()); err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, userAgent, ip)
}

// Logout revokes a refresh session.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash, err := auth.HashRefreshToken(refreshToken)
	if err != nil {
		return nil // token is already unusable
	}

	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		return nil // nothing to revoke
	}

	_ = s.sessions.Revoke(ctx, session.ID, time.Now())
	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *models.User, userAgent, ip string) (*Tokens, error) {
	roles, err := s.users.RolesForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	access, err := s.jwt.NewAccessToken(user.ID, user.Username, roles)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refresh, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = s.sessions.Create(ctx, models.NewSession{
		UserID:    user.ID,
		TokenHash: refreshHash,
		UserAgent: userAgent,
		IP:        normalizeIP(ip),
		ExpiresAt: time.Now().Add(s.cfg.RefreshTTL),
	})
	if err != nil {
		return nil, err
	}

	return &Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         user,
		Roles:        roles,
	}, nil
}

// Profile returns a user and its roles for API consumption.
func (s *AuthService) Profile(ctx context.Context, id uuid.UUID) (*models.User, []string, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	roles, err := s.users.RolesForUser(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, roles, nil
}

const maxDisplayNameLen = 50

// UpdateProfile updates the caller's editable profile fields. Currently only
// the display name is supported; an empty name resets it to the username.
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, displayName string) (*models.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	displayName = strings.TrimSpace(displayName)
	if len(displayName) > maxDisplayNameLen {
		return nil, fmt.Errorf("%w: display name must be at most %d chars", ErrValidation, maxDisplayNameLen)
	}
	if displayName == "" {
		displayName = user.Username
	}

	return s.users.UpdateDisplayName(ctx, userID, displayName)
}

// ChangePassword verifies the current password, stores a new hash and revokes
// every refresh session so the user must log in again.
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: password must be at least 8 chars", ErrValidation)
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	valid, err := s.hasher.Verify(currentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !valid {
		return ErrWrongPassword
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if _, err := s.users.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	return s.sessions.RevokeAllForUser(ctx, userID)
}

func normalizeIP(ip string) string {
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		return host
	}
	return ip
}
