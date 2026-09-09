package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/masoniclounge/masoniccore/internal/models"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a unique constraint is violated
// (e.g. duplicate username or email).
var ErrConflict = errors.New("conflict")

// isUniqueViolation reports whether err is a PostgreSQL unique violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

const userColumns = `id, username, email, password_hash, display_name,
	avatar_url, status, created_at, updated_at, last_seen_at`

// UserStore provides data access to users and roles.
type UserStore struct {
	db Pool
}

// NewUserStore creates a UserStore over the shared pool.
func NewUserStore(db Pool) *UserStore {
	return &UserStore{db: db}
}

// Create inserts a new user and returns it.
func (s *UserStore) Create(ctx context.Context, nu models.NewUser) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING `+userColumns,
		nu.Username, nu.Email, nu.PasswordHash, nu.DisplayName,
	).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastSeenAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: username or email already taken", ErrConflict)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetByUsername finds a user by username (case-insensitive).
func (s *UserStore) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.scanOne(s.db.QueryRow(ctx, `
		SELECT `+userColumns+` FROM users WHERE lower(username) = lower($1)`,
		username))
}

// GetByEmail finds a user by email (case-insensitive).
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.scanOne(s.db.QueryRow(ctx, `
		SELECT `+userColumns+` FROM users WHERE lower(email) = lower($1)`,
		email))
}

// GetByID finds a user by its primary key.
func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.scanOne(s.db.QueryRow(ctx, `
		SELECT `+userColumns+` FROM users WHERE id = $1`,
		id))
}

// RolesForUser returns the role keys assigned to a user.
func (s *UserStore) RolesForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.key
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.key`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	roles := make([]string, 0, 3)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	return roles, nil
}

// AddMemberRole assigns the default member role to a user.
func (s *UserStore) AddMemberRole(ctx context.Context, userID uuid.UUID) error {
	return s.AssignRole(ctx, userID, models.RoleMember)
}

// AssignRole assigns a role to a user by its key.
func (s *UserStore) AssignRole(ctx context.Context, userID uuid.UUID, roleKey string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE key = $2`,
		userID, roleKey)
	if err != nil {
		return fmt.Errorf("assign role %s: %w", roleKey, err)
	}
	return nil
}

// UpdateAvatar sets the avatar URL of a user and returns the updated row.
func (s *UserStore) UpdateAvatar(ctx context.Context, userID uuid.UUID, avatarURL string) (*models.User, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE users SET avatar_url = $2, updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns, userID, avatarURL)
	return s.scanOne(row)
}

func (s *UserStore) scanOne(row pgx.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.DisplayName,
		&u.AvatarURL, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastSeenAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
