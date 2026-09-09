package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/masoniclounge/masoniccore/internal/db"
	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/store"
)

// setupDB opens a test database from TEST_DATABASE_URL, applies migrations
// and wipes all tables so each test starts from a clean schema.
func setupDB(t *testing.T) *db.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration tests")
	}
	ctx := context.Background()
	d, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(d.Close)
	if err := d.Migrate(ctx); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	if _, err := d.Pool().Exec(ctx, `TRUNCATE users, groups, settings RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate test db: %v", err)
	}
	return d
}

// createUser inserts a user with the member role and returns it.
func createUser(t *testing.T, pool *pgxpool.Pool, username, email string) *models.User {
	t.Helper()
	users := store.NewUserStore(pool)
	u, err := users.Create(context.Background(), models.NewUser{
		Username:     username,
		Email:        email,
		PasswordHash: "not-used-in-tests",
		DisplayName:  username,
	})
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	if err := users.AddMemberRole(context.Background(), u.ID); err != nil {
		t.Fatalf("add member role to %s: %v", username, err)
	}
	return u
}
