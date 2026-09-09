package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

const (
	maxThreadTitleLen = 200
	maxBodyLen        = 50000
)

// CreateThreadInput carries the fields required to start a thread.
type CreateThreadInput struct {
	GroupID       uuid.UUID
	AuthorID      uuid.UUID
	Title         string
	Body          string
	AttachmentIDs []uuid.UUID
}

// UpdateThreadInput carries optional thread mutations. Pinning and locking
// are only honoured when the caller is a moderator or administrator.
type UpdateThreadInput struct {
	Title  *string
	Pinned *bool
	Locked *bool
}

// ThreadService contains business logic for threads.
type ThreadService struct {
	groups      *store.GroupStore
	threads     *store.ThreadStore
	attachments *store.AttachmentStore
}

// NewThreadService creates a ThreadService.
func NewThreadService(groups *store.GroupStore, threads *store.ThreadStore, attachments *store.AttachmentStore) *ThreadService {
	return &ThreadService{groups: groups, threads: threads, attachments: attachments}
}

// ListByGroup returns threads of a group with pagination.
func (s *ThreadService) ListByGroup(ctx context.Context, groupID uuid.UUID, limit, offset int) ([]models.ThreadSummary, int, error) {
	if _, err := s.groups.GetByID(ctx, groupID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, 0, ErrGroupNotFound
		}
		return nil, 0, err
	}
	return s.threads.ListByGroup(ctx, groupID, limit, offset)
}

// Get returns a thread by ID and bumps its view counter.
func (s *ThreadService) Get(ctx context.Context, id uuid.UUID) (*models.ThreadSummary, error) {
	ts, err := s.threads.SummaryByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrThreadNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.threads.IncrementViews(ctx, id); err != nil {
		return nil, err
	}
	return ts, nil
}

// Create validates the input and starts a thread with its first post.
func (s *ThreadService) Create(ctx context.Context, in CreateThreadInput) (*models.ThreadSummary, error) {
	if _, err := s.groups.GetByID(ctx, in.GroupID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	if err := validateTitle(in.Title); err != nil {
		return nil, err
	}
	if err := validateBody(in.Body); err != nil {
		return nil, err
	}
	if len(in.AttachmentIDs) > maxPostAttachments {
		return nil, ErrInvalidInput
	}

	th, firstPostID, err := s.threads.Create(ctx,
		models.NewThread{GroupID: in.GroupID, AuthorID: in.AuthorID, Title: in.Title},
		models.NewPost{AuthorID: in.AuthorID, Body: in.Body})
	if err != nil {
		return nil, err
	}

	if len(in.AttachmentIDs) > 0 {
		linked, err := s.attachments.LinkToPost(ctx, firstPostID, in.AuthorID, in.AttachmentIDs)
		if err != nil {
			return nil, err
		}
		if linked != int64(len(in.AttachmentIDs)) {
			// roll back the just-created thread so a failed link does not
			// leave an empty thread behind
			_ = s.threads.Delete(ctx, th.ID)
			return nil, ErrForbidden
		}
	}

	return s.threads.SummaryByID(ctx, th.ID)
}

// Update applies the requested mutations, honouring the caller's permissions.
func (s *ThreadService) Update(ctx context.Context, id uuid.UUID, in UpdateThreadInput, callerID uuid.UUID, roles []string) (*models.ThreadSummary, error) {
	t, err := s.threads.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrThreadNotFound
	}
	if err != nil {
		return nil, err
	}

	moderator := hasRole(roles, models.RoleAdmin, models.RoleModerator)
	if t.AuthorID != callerID && !moderator {
		return nil, ErrForbidden
	}

	if in.Title != nil {
		if t.AuthorID != callerID {
			return nil, ErrForbidden
		}
		if err := validateTitle(*in.Title); err != nil {
			return nil, err
		}
		t.Title = *in.Title
	}
	if moderator && in.Pinned != nil {
		t.Pinned = *in.Pinned
	}
	if moderator && in.Locked != nil {
		t.Locked = *in.Locked
	}

	updated, err := s.threads.Update(ctx, id, t.Title, t.Pinned, t.Locked)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrThreadNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.threads.SummaryByID(ctx, updated.ID)
}

// Delete removes a thread if the caller is its author or a moderator.
func (s *ThreadService) Delete(ctx context.Context, id uuid.UUID, callerID uuid.UUID, roles []string) error {
	t, err := s.threads.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return ErrThreadNotFound
	}
	if err != nil {
		return err
	}
	if t.AuthorID != callerID && !hasRole(roles, models.RoleAdmin, models.RoleModerator) {
		return ErrForbidden
	}
	if err := s.threads.Delete(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrThreadNotFound
		}
		return err
	}
	return nil
}

func validateTitle(title string) error {
	t := strings.TrimSpace(title)
	if t == "" || utf8.RuneCountInString(t) > maxThreadTitleLen {
		return ErrInvalidInput
	}
	return nil
}

func validateBody(body string) error {
	b := strings.TrimSpace(body)
	if b == "" || utf8.RuneCountInString(b) > maxBodyLen {
		return ErrInvalidInput
	}
	return nil
}
