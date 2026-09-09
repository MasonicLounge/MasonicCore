package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/masoniclounge/masoniccore/internal/models"
)

const attachmentColumns = `id, owner_id, post_id, filename, content_type, size_bytes, storage_key, public_url, created_at, updated_at`

// AttachmentStore provides SQL access to the attachments table.
type AttachmentStore struct {
	db Pool
}

// NewAttachmentStore creates an AttachmentStore backed by the given Pool.
func NewAttachmentStore(db Pool) *AttachmentStore {
	return &AttachmentStore{db: db}
}

// Create inserts a new attachment and returns it.
func (s *AttachmentStore) Create(ctx context.Context, a models.NewAttachment) (*models.Attachment, error) {
	row := s.db.QueryRow(ctx,
		`INSERT INTO attachments (owner_id, post_id, filename, content_type, size_bytes, storage_key, public_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+attachmentColumns,
		a.OwnerID, a.PostID, a.Filename, a.ContentType, a.SizeBytes, a.StorageKey, a.PublicURL)
	return scanAttachment(row)
}

// GetByID returns an attachment by its ID.
func (s *AttachmentStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Attachment, error) {
	row := s.db.QueryRow(ctx, `SELECT `+attachmentColumns+` FROM attachments WHERE id = $1`, id)
	return scanAttachment(row)
}

// Delete removes an attachment row by its ID. Returns ErrNotFound if it does not exist.
func (s *AttachmentStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM attachments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAttachment(row pgx.Row) (*models.Attachment, error) {
	var a models.Attachment
	err := row.Scan(&a.ID, &a.OwnerID, &a.PostID, &a.Filename, &a.ContentType,
		&a.SizeBytes, &a.StorageKey, &a.PublicURL, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}
