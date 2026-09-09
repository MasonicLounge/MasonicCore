package store

import (
	"context"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const moderationColumns = `ml.id, ml.moderator_id, ml.action, ml.target_type, ml.target_id, ml.reason, ml.created_at`

// ModerationStore provides SQL access to the moderation log.
type ModerationStore struct {
	db Pool
}

// NewModerationStore creates a ModerationStore backed by the given Pool.
func NewModerationStore(db Pool) *ModerationStore {
	return &ModerationStore{db: db}
}

// Create inserts a moderation log entry and returns it with its identity.
func (s *ModerationStore) Create(ctx context.Context, m models.ModerationEntry) (*models.ModerationEntry, error) {
	row := s.db.QueryRow(ctx, `
		INSERT INTO moderation_log (moderator_id, action, target_type, target_id, reason)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, moderator_id, action, target_type, target_id, reason, created_at`,
		m.ModeratorID, m.Action, m.TargetType, m.TargetID, m.Reason)
	var out models.ModerationEntry
	if err := row.Scan(&out.ID, &out.ModeratorID, &out.Action, &out.TargetType, &out.TargetID, &out.Reason, &out.CreatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns moderation log entries with moderator names, newest first.
func (s *ModerationStore) List(ctx context.Context, limit, offset int) ([]models.ModerationEntry, int, error) {
	total := 0
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM moderation_log`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+moderationColumns+`, u.username, COALESCE(u.display_name, u.username)
		FROM moderation_log ml
		JOIN users u ON u.id = ml.moderator_id
		ORDER BY ml.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []models.ModerationEntry{}
	for rows.Next() {
		var e models.ModerationEntry
		if err := rows.Scan(&e.ID, &e.ModeratorID, &e.Action, &e.TargetType, &e.TargetID, &e.Reason, &e.CreatedAt,
			&e.ModeratorUsername, &e.ModeratorDisplay); err != nil {
			return nil, 0, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
