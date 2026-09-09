package services

import "errors"

// Sentinel errors for the core (groups/threads/posts) domain.
var (
	ErrGroupNotFound  = errors.New("group not found")
	ErrThreadNotFound = errors.New("thread not found")
	ErrPostNotFound   = errors.New("post not found")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidInput   = errors.New("validation failed")
	ErrConflict       = errors.New("conflict")
	ErrThreadLocked   = errors.New("thread locked")
	ErrNotEmpty       = errors.New("resource not empty")
)

// hasRole reports whether the role list contains any of the given keys.
func hasRole(roles []string, keys ...string) bool {
	for _, r := range roles {
		for _, k := range keys {
			if r == k {
				return true
			}
		}
	}
	return false
}
