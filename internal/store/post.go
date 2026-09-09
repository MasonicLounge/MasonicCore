package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const postColumns = `id, thread_id, author_id, body, edited_by_id, edited_at, created_at, updated_at`

const postSummarySelect = `
SELECT p.id, p.thread_id, p.author_id, p.body, p.edited_by_id, p.edited_at, p.created_at, p.updated_at,
       u.username, COALESCE(u.display_name, u.username)
FROM posts p
JOIN users u ON u.id = p.author_id`

// PostStore provides SQL access to the posts table.
type PostStore struct {
	db Pool
}

// NewPostStore creates a PostStore backed by the given Pool.
func NewPostStore(db Pool) *PostStore {
	return &PostStore{db: db}
}

// Create inserts a new post and returns it.
func (s *PostStore) Create(ctx context.Context, p models.NewPost) (*models.Post, error) {
	row := s.db.QueryRow(ctx,
		`INSERT INTO posts (thread_id, author_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING `+postColumns,
		p.ThreadID, p.AuthorID, p.Body)
	return scanPost(row)
}

// ListByThread returns posts in chronological order with pagination.
func (s *PostStore) ListByThread(ctx context.Context, threadID uuid.UUID, limit, offset int) ([]models.PostSummary, int, error) {
	rows, err := s.db.Query(ctx,
		postSummarySelect+`
		WHERE p.thread_id = $1
		ORDER BY p.created_at ASC
		LIMIT $2 OFFSET $3`,
		threadID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]models.PostSummary, 0)
	for rows.Next() {
		ps, err := scanPostSummary(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *ps)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM posts WHERE thread_id = $1`, threadID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a post by its ID.
func (s *PostStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	row := s.db.QueryRow(ctx, `SELECT `+postColumns+` FROM posts WHERE id = $1`, id)
	return scanPost(row)
}

// SummaryByID returns a post joined with its author.
func (s *PostStore) SummaryByID(ctx context.Context, id uuid.UUID) (*models.PostSummary, error) {
	row := s.db.QueryRow(ctx, postSummarySelect+` WHERE p.id = $1`, id)
	return scanPostSummary(row)
}

// UpdateBody replaces the body of a post and records the editor.
func (s *PostStore) UpdateBody(ctx context.Context, id uuid.UUID, body string, editorID uuid.UUID) (*models.Post, error) {
	row := s.db.QueryRow(ctx,
		`UPDATE posts
		 SET body = $2, edited_by_id = $3, edited_at = now(), updated_at = now()
		 WHERE id = $1
		 RETURNING `+postColumns,
		id, body, editorID)
	return scanPost(row)
}

// Delete removes a post by its ID. Returns ErrNotFound if it does not exist.
func (s *PostStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM posts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanPost(row pgx.Row) (*models.Post, error) {
	var p models.Post
	err := row.Scan(&p.ID, &p.ThreadID, &p.AuthorID, &p.Body, &p.EditedByID,
		&p.EditedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func scanPostSummary(row pgx.Row) (*models.PostSummary, error) {
	var ps models.PostSummary
	err := row.Scan(&ps.ID, &ps.ThreadID, &ps.AuthorID, &ps.Body, &ps.EditedByID,
		&ps.EditedAt, &ps.CreatedAt, &ps.UpdatedAt, &ps.AuthorUsername, &ps.AuthorDisplay)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ps, nil
}
