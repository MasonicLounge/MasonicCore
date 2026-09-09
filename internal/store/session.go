package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const sessionColumns = `id, user_id, token_hash, user_agent, ip::text AS ip, created_at, expires_at, revoked_at`

// SessionStore provides data access to refresh sessions.
type SessionStore struct {
	db Pool
}

// NewSessionStore creates a SessionStore over the shared pool.
func NewSessionStore(db Pool) *SessionStore {
	return &SessionStore{db: db}
}

// Create persists a new refresh session.
func (s *SessionStore) Create(ctx context.Context, ns models.NewSession) (*models.Session, error) {
	sess := &models.Session{}
	var ip *string
	if ns.IP != "" {
		ip = &ns.IP
	}
	err := s.db.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+sessionColumns,
		ns.UserID, ns.TokenHash, ns.UserAgent, ip, ns.ExpiresAt,
	).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.UserAgent,
		&sess.IP, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

// GetByTokenHash finds an active (non-revoked, non-expired) session.
func (s *SessionStore) GetByTokenHash(ctx context.Context, hash string) (*models.Session, error) {
	sess := &models.Session{}
	err := s.db.QueryRow(ctx, `
		SELECT `+sessionColumns+`
		FROM user_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > now()`,
		hash,
	).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash, &sess.UserAgent,
		&sess.IP, &sess.CreatedAt, &sess.ExpiresAt, &sess.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("query session: %w", err)
	}
	return sess, nil
}

// Revoke marks a session as revoked.
func (s *SessionStore) Revoke(ctx context.Context, id uuid.UUID, at time.Time) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`,
		id, at)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteExpired removes expired or revoked sessions. Returns rows deleted.
func (s *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM user_sessions WHERE expires_at < now() OR revoked_at IS NOT NULL`)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Touch updates a session's expiry (sliding expiration on refresh).
func (s *SessionStore) Touch(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE user_sessions SET expires_at = $2 WHERE id = $1 AND revoked_at IS NULL`,
		id, expiresAt)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
