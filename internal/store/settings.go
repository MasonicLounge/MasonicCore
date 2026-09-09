package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// SettingsStore provides access to the key/value settings table (JSONB values).
type SettingsStore struct {
	db Pool
}

// NewSettingsStore creates a SettingsStore over the shared pool.
func NewSettingsStore(db Pool) *SettingsStore {
	return &SettingsStore{db: db}
}

// Get returns the raw JSON value stored under key.
func (s *SettingsStore) Get(ctx context.Context, key string) (json.RawMessage, error) {
	var value json.RawMessage
	err := s.db.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, nil
}

// Set upserts a JSON value under key.
func (s *SettingsStore) Set(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal setting %q: %w", key, err)
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2::jsonb, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, string(raw))
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
