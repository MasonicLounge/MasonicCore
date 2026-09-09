package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

// maxPostAttachments caps how many files may be attached to a single post.
const maxPostAttachments = 10

// PostService contains business logic for posts.
type PostService struct {
	posts       *store.PostStore
	threads     *store.ThreadStore
	attachments *store.AttachmentStore
}

// NewPostService creates a PostService.
func NewPostService(posts *store.PostStore, threads *store.ThreadStore, attachments *store.AttachmentStore) *PostService {
	return &PostService{posts: posts, threads: threads, attachments: attachments}
}

// ListByThread returns posts of a thread in chronological order.
func (s *PostService) ListByThread(ctx context.Context, threadID uuid.UUID, limit, offset int) ([]models.PostSummary, int, error) {
	if _, err := s.threads.GetByID(ctx, threadID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, 0, ErrThreadNotFound
		}
		return nil, 0, err
	}
	return s.posts.ListByThread(ctx, threadID, limit, offset)
}

// Create adds a reply to a thread, optionally linking owned attachments.
func (s *PostService) Create(ctx context.Context, threadID uuid.UUID, authorID uuid.UUID, body string, attachmentIDs []uuid.UUID) (*models.PostSummary, error) {
	th, err := s.threads.GetByID(ctx, threadID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrThreadNotFound
	}
	if err != nil {
		return nil, err
	}
	if th.Locked {
		return nil, ErrThreadLocked
	}
	if err := validateBody(body); err != nil {
		return nil, err
	}
	if len(attachmentIDs) > maxPostAttachments {
		return nil, ErrInvalidInput
	}

	p, err := s.posts.Create(ctx, models.NewPost{ThreadID: threadID, AuthorID: authorID, Body: body})
	if err != nil {
		return nil, err
	}
	if len(attachmentIDs) > 0 {
		linked, err := s.attachments.LinkToPost(ctx, p.ID, authorID, attachmentIDs)
		if err != nil {
			return nil, err
		}
		if linked != int64(len(attachmentIDs)) {
			return nil, ErrForbidden
		}
	}
	if err := s.threads.BumpPost(ctx, threadID, p.CreatedAt); err != nil {
		return nil, err
	}
	return s.posts.SummaryByID(ctx, p.ID)
}

// Update edits the body of a post. Only the author may edit.
func (s *PostService) Update(ctx context.Context, id uuid.UUID, authorID uuid.UUID, body string) (*models.PostSummary, error) {
	p, err := s.posts.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	if p.AuthorID != authorID {
		return nil, ErrForbidden
	}
	if err := validateBody(body); err != nil {
		return nil, err
	}

	if _, err := s.posts.UpdateBody(ctx, id, body, authorID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return s.posts.SummaryByID(ctx, id)
}

// Delete removes a post if the caller is its author or a moderator.
func (s *PostService) Delete(ctx context.Context, id uuid.UUID, callerID uuid.UUID, roles []string) error {
	p, err := s.posts.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return ErrPostNotFound
	}
	if err != nil {
		return err
	}
	if p.AuthorID != callerID && !hasRole(roles, models.RoleAdmin, models.RoleModerator) {
		return ErrForbidden
	}

	if err := s.posts.Delete(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	return s.threads.RecalcStats(ctx, p.ThreadID)
}
