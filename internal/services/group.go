package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const (
	maxGroupNameLen        = 100
	maxGroupDescriptionLen = 2000
	maxSlugLen             = 80
)

// GroupInput carries the mutable fields of a group.
type GroupInput struct {
	Name        string
	Slug        string
	Description string
	ParentID    *uuid.UUID
	SortOrder   int
}

// GroupService contains business logic for groups.
type GroupService struct {
	groups  *store.GroupStore
	threads *store.ThreadStore
}

// NewGroupService creates a GroupService.
func NewGroupService(groups *store.GroupStore, threads *store.ThreadStore) *GroupService {
	return &GroupService{groups: groups, threads: threads}
}

// List returns all groups ordered for display.
func (s *GroupService) List(ctx context.Context) ([]models.Group, error) {
	return s.groups.List(ctx)
}

// Get returns a single group by ID.
func (s *GroupService) Get(ctx context.Context, id uuid.UUID) (*models.Group, error) {
	g, err := s.groups.GetByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	return g, err
}

// Create validates and inserts a new group.
func (s *GroupService) Create(ctx context.Context, in GroupInput) (*models.Group, error) {
	if err := validateGroup(in); err != nil {
		return nil, err
	}
	g, err := s.groups.Create(ctx, models.NewGroup{
		Name:        in.Name,
		Slug:        in.Slug,
		Description: in.Description,
		ParentID:    in.ParentID,
		SortOrder:   in.SortOrder,
	})
	if errors.Is(err, store.ErrConflict) {
		return nil, ErrConflict
	}
	return g, err
}

// Update validates and applies new values to a group.
func (s *GroupService) Update(ctx context.Context, id uuid.UUID, in GroupInput) (*models.Group, error) {
	if err := validateGroup(in); err != nil {
		return nil, err
	}
	g, err := s.groups.Update(ctx, id, models.NewGroup{
		Name:        in.Name,
		Slug:        in.Slug,
		Description: in.Description,
		ParentID:    in.ParentID,
		SortOrder:   in.SortOrder,
	})
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrGroupNotFound
	}
	if errors.Is(err, store.ErrConflict) {
		return nil, ErrConflict
	}
	return g, err
}

// Delete removes a group if it contains no threads.
func (s *GroupService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.groups.GetByID(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrGroupNotFound
		}
		return err
	}
	n, err := s.threads.CountByGroup(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrNotEmpty
	}
	if err := s.groups.Delete(ctx, id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrGroupNotFound
		}
		return err
	}
	return nil
}

func validateGroup(in GroupInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxGroupNameLen {
		return ErrInvalidInput
	}
	slug := strings.TrimSpace(in.Slug)
	if !slugRe.MatchString(slug) || len(slug) > maxSlugLen {
		return ErrInvalidInput
	}
	if utf8.RuneCountInString(in.Description) > maxGroupDescriptionLen {
		return ErrInvalidInput
	}
	if in.ParentID != nil && *in.ParentID == uuid.Nil {
		return ErrInvalidInput
	}
	return nil
}
