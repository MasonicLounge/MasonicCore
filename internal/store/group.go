package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const groupColumns = `id, name, slug, description, parent_id, sort_order, created_at, updated_at`

// GroupStore provides SQL access to the groups table.
type GroupStore struct {
	db Pool
}

// NewGroupStore creates a GroupStore backed by the given Pool.
func NewGroupStore(db Pool) *GroupStore {
	return &GroupStore{db: db}
}

// Create inserts a new group and returns it.
func (s *GroupStore) Create(ctx context.Context, g models.NewGroup) (*models.Group, error) {
	row := s.db.QueryRow(ctx,
		`INSERT INTO groups (name, slug, description, parent_id, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+groupColumns,
		g.Name, g.Slug, g.Description, g.ParentID, g.SortOrder)
	return scanGroup(row)
}

// List returns all groups ordered by sort order then name.
func (s *GroupStore) List(ctx context.Context) ([]models.Group, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+groupColumns+` FROM groups ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := make([]models.Group, 0)
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.Description, &g.ParentID,
			&g.SortOrder, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// GetByID returns a group by its ID.
func (s *GroupStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Group, error) {
	row := s.db.QueryRow(ctx, `SELECT `+groupColumns+` FROM groups WHERE id = $1`, id)
	return scanGroup(row)
}

// GetBySlug returns a group by its slug.
func (s *GroupStore) GetBySlug(ctx context.Context, slug string) (*models.Group, error) {
	row := s.db.QueryRow(ctx, `SELECT `+groupColumns+` FROM groups WHERE slug = $1`, slug)
	return scanGroup(row)
}

// Update replaces the mutable fields of a group and returns it.
func (s *GroupStore) Update(ctx context.Context, id uuid.UUID, g models.NewGroup) (*models.Group, error) {
	row := s.db.QueryRow(ctx,
		`UPDATE groups
		 SET name = $2, slug = $3, description = $4, parent_id = $5, sort_order = $6, updated_at = now()
		 WHERE id = $1
		 RETURNING `+groupColumns,
		id, g.Name, g.Slug, g.Description, g.ParentID, g.SortOrder)
	return scanGroup(row)
}

// Delete removes a group by its ID. Returns ErrNotFound if it does not exist.
func (s *GroupStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanGroup(row pgx.Row) (*models.Group, error) {
	var g models.Group
	err := row.Scan(&g.ID, &g.Name, &g.Slug, &g.Description, &g.ParentID,
		&g.SortOrder, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
