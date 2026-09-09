package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const threadColumns = `id, group_id, author_id, title, pinned, locked, views, post_count, last_post_at, created_at, updated_at`

const threadSummarySelect = `
SELECT t.id, t.group_id, t.author_id, t.title, t.pinned, t.locked, t.views, t.post_count,
       t.last_post_at, t.created_at, t.updated_at,
       u.username, COALESCE(u.display_name, u.username)
FROM threads t
JOIN users u ON u.id = t.author_id`

// ThreadStore provides SQL access to the threads table.
type ThreadStore struct {
	db Pool
}

// NewThreadStore creates a ThreadStore backed by the given Pool.
func NewThreadStore(db Pool) *ThreadStore {
	return &ThreadStore{db: db}
}

// Create inserts a new thread. The first post is created in the same transaction,
// so the thread always starts with exactly one post.
func (s *ThreadStore) Create(ctx context.Context, t models.NewThread, first models.NewPost) (*models.Thread, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`INSERT INTO threads (group_id, author_id, title)
		 VALUES ($1, $2, $3)
		 RETURNING `+threadColumns,
		t.GroupID, t.AuthorID, t.Title)
	th, err := scanThread(row)
	if err != nil {
		return nil, err
	}

	first.ThreadID = th.ID
	row = tx.QueryRow(ctx,
		`INSERT INTO posts (thread_id, author_id, body)
		 VALUES ($1, $2, $3)
		 RETURNING created_at`,
		th.ID, first.AuthorID, first.Body)
	var postCreatedAt time.Time
	if err := row.Scan(&postCreatedAt); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE threads SET post_count = post_count + 1, last_post_at = $2, updated_at = now() WHERE id = $1`,
		th.ID, postCreatedAt); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	th.PostCount = 1
	th.LastPostAt = &postCreatedAt
	return th, nil
}

// ListByGroup returns thread summaries in a group ordered by pinned then most recent activity.
func (s *ThreadStore) ListByGroup(ctx context.Context, groupID uuid.UUID, limit, offset int) ([]models.ThreadSummary, int, error) {
	rows, err := s.db.Query(ctx,
		threadSummarySelect+`
		WHERE t.group_id = $1
		ORDER BY t.pinned DESC, t.last_post_at DESC NULLS LAST, t.created_at DESC
		LIMIT $2 OFFSET $3`,
		groupID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]models.ThreadSummary, 0)
	for rows.Next() {
		ts, err := scanThreadSummary(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *ts)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM threads WHERE group_id = $1`, groupID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID returns a thread by its ID.
func (s *ThreadStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	row := s.db.QueryRow(ctx, `SELECT `+threadColumns+` FROM threads WHERE id = $1`, id)
	return scanThread(row)
}

// SummaryByID returns a thread joined with its author.
func (s *ThreadStore) SummaryByID(ctx context.Context, id uuid.UUID) (*models.ThreadSummary, error) {
	row := s.db.QueryRow(ctx, threadSummarySelect+` WHERE t.id = $1`, id)
	return scanThreadSummary(row)
}

// Update sets the mutable fields of a thread and returns it.
func (s *ThreadStore) Update(ctx context.Context, id uuid.UUID, title string, pinned, locked bool) (*models.Thread, error) {
	row := s.db.QueryRow(ctx,
		`UPDATE threads
		 SET title = $2, pinned = $3, locked = $4, updated_at = now()
		 WHERE id = $1
		 RETURNING `+threadColumns,
		id, title, pinned, locked)
	return scanThread(row)
}

// Delete removes a thread by its ID. Returns ErrNotFound if it does not exist.
func (s *ThreadStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM threads WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// IncrementViews bumps the view counter of a thread.
func (s *ThreadStore) IncrementViews(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `UPDATE threads SET views = views + 1 WHERE id = $1`, id)
	return err
}

// CountByGroup counts threads belonging to a group.
func (s *ThreadStore) CountByGroup(ctx context.Context, groupID uuid.UUID) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `SELECT count(*) FROM threads WHERE group_id = $1`, groupID).Scan(&n)
	return n, err
}

// BumpPost increments the post counter and refreshes the last activity time.
func (s *ThreadStore) BumpPost(ctx context.Context, threadID uuid.UUID, lastPostAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`UPDATE threads SET post_count = post_count + 1, last_post_at = $2, updated_at = now() WHERE id = $1`,
		threadID, lastPostAt)
	return err
}

// RecalcStats recomputes the post counter and last activity time from the posts table.
func (s *ThreadStore) RecalcStats(ctx context.Context, threadID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		`UPDATE threads t
		 SET post_count = COALESCE((SELECT count(*) FROM posts p WHERE p.thread_id = t.id), 0),
		     last_post_at = (SELECT max(created_at) FROM posts p WHERE p.thread_id = t.id),
		     updated_at = now()
		 WHERE t.id = $1`,
		threadID)
	return err
}

func scanThread(row pgx.Row) (*models.Thread, error) {
	var t models.Thread
	err := row.Scan(&t.ID, &t.GroupID, &t.AuthorID, &t.Title, &t.Pinned, &t.Locked,
		&t.Views, &t.PostCount, &t.LastPostAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func scanThreadSummary(row pgx.Row) (*models.ThreadSummary, error) {
	var ts models.ThreadSummary
	err := row.Scan(&ts.ID, &ts.GroupID, &ts.AuthorID, &ts.Title, &ts.Pinned, &ts.Locked,
		&ts.Views, &ts.PostCount, &ts.LastPostAt, &ts.CreatedAt, &ts.UpdatedAt,
		&ts.AuthorUsername, &ts.AuthorDisplay)
	if err != nil {
		if isNoRows(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ts, nil
}
