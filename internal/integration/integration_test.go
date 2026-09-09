package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/masoniclounge/masoniccore/internal/auth"
	"github.com/masoniclounge/masoniccore/internal/config"
	"github.com/masoniclounge/masoniccore/internal/models"
	"github.com/masoniclounge/masoniccore/internal/services"
	"github.com/masoniclounge/masoniccore/internal/store"
)

// TestInstallValidatorRejects covers input validation paths of InstallService.
// Each test creates a fresh database via setupDB, so the forum is never installed.
func TestInstallValidatorRejects(t *testing.T) {
	d := setupDB(t)
	ctx := context.Background()
	pool := d.Pool()

	users := store.NewUserStore(pool)
	settings := store.NewSettingsStore(pool)
	svc := services.NewInstallService(users, settings, auth.NewPasswordHasher())

	cases := []struct {
		name  string
		input services.InstallInput
	}{
		{"empty forum name", services.InstallInput{ForumName: "  ", AdminUsername: "root", AdminEmail: "r@e.com", AdminPassword: "password123"}},
		{"short password", services.InstallInput{ForumName: "F", AdminUsername: "root", AdminEmail: "r@e.com", AdminPassword: "short"}},
		{"bad email", services.InstallInput{ForumName: "F", AdminUsername: "root", AdminEmail: "not-an-email", AdminPassword: "password123"}},
		{"short username", services.InstallInput{ForumName: "F", AdminUsername: "ab", AdminEmail: "r@e.com", AdminPassword: "password123"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Setup(ctx, tc.input); !errors.Is(err, services.ErrInvalidInput) {
				t.Fatalf("Setup(%q) error = %v, want ErrInvalidInput", tc.name, err)
			}
		})
	}
}

// TestInstallFlow exercises the happy path and the already-installed guard.
func TestInstallFlow(t *testing.T) {
	d := setupDB(t)
	ctx := context.Background()
	pool := d.Pool()

	users := store.NewUserStore(pool)
	settings := store.NewSettingsStore(pool)
	svc := services.NewInstallService(users, settings, auth.NewPasswordHasher())

	st, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.Installed {
		t.Fatal("Status.Installed = true, want false on fresh db")
	}

	u, err := svc.Setup(ctx, services.InstallInput{
		ForumName:     "Test Forum",
		AdminUsername: "rootadmin",
		AdminEmail:    "root@example.com",
		AdminPassword: "password123",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if u.Username != "rootadmin" {
		t.Fatalf("Setup user = %q, want rootadmin", u.Username)
	}
	roles, err := users.RolesForUser(ctx, u.ID)
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if !contains(roles, models.RoleAdmin) || !contains(roles, models.RoleMember) {
		t.Fatalf("admin user roles = %v, want admin+member", roles)
	}

	st2, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("Status after setup: %v", err)
	}
	if !st2.Installed || st2.ForumName == nil || *st2.ForumName != "Test Forum" {
		t.Fatalf("Status after setup = %+v, want installed with forum name", st2)
	}

	if _, err := svc.Setup(ctx, services.InstallInput{
		ForumName: "Again", AdminUsername: "other", AdminEmail: "o@e.com", AdminPassword: "password123",
	}); !errors.Is(err, services.ErrAlreadyInstalled) {
		t.Fatalf("second Setup error = %v, want ErrAlreadyInstalled", err)
	}
}

// TestAuthFlow covers registration, login and session validation end to end.
func TestAuthFlow(t *testing.T) {
	d := setupDB(t)
	ctx := context.Background()
	pool := d.Pool()

	users := store.NewUserStore(pool)
	sessions := store.NewSessionStore(pool)
	hasher := auth.NewPasswordHasher()
	jwtm := auth.NewManager("test-secret-0123456789abcdef0123456789", "test", time.Minute)
	cfg := config.Config{AccessTTL: time.Minute, RefreshTTL: 24 * time.Hour}
	svc := services.NewAuthService(users, sessions, hasher, jwtm, cfg)

	u, err := svc.Register(ctx, "alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if u.Username != "alice" {
		t.Fatalf("Register username = %q", u.Username)
	}

	if _, err := svc.Register(ctx, "alice", "other@example.com", "password123"); !errors.Is(err, services.ErrUsernameTaken) {
		t.Fatalf("duplicate username error = %v, want ErrUsernameTaken", err)
	}
	if _, err := svc.Register(ctx, "bob", "alice@example.com", "password123"); !errors.Is(err, services.ErrUsernameTaken) {
		t.Fatalf("duplicate email error = %v, want ErrUsernameTaken", err)
	}
	if _, err := svc.Register(ctx, "carol", "carol@example.com", "short"); !errors.Is(err, services.ErrValidation) {
		t.Fatalf("short password error = %v, want ErrValidation", err)
	}

	toks, err := svc.Login(ctx, "alice", "password123", "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if toks.AccessToken == "" || toks.RefreshToken == "" {
		t.Fatal("Login returned empty tokens")
	}
	if toks.User.ID != u.ID {
		t.Fatalf("Login user id = %v, want %v", toks.User.ID, u.ID)
	}
	if !contains(toks.Roles, models.RoleMember) {
		t.Fatalf("Login roles = %v, want member", toks.Roles)
	}

	if _, err := svc.Login(ctx, "alice", "wrong-password", "test-agent", "127.0.0.1"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Login(ctx, "missing", "password123", "ag", "ip"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Fatalf("missing user error = %v, want ErrInvalidCredentials", err)
	}

	// Banned users must be rejected at login.
	if _, err := users.UpdateStatus(ctx, u.ID, models.UserStatusBanned); err != nil {
		t.Fatalf("UpdateStatus banned: %v", err)
	}
	if _, err := svc.Login(ctx, "alice", "password123", "a", "ip"); !errors.Is(err, services.ErrUserNotActive) {
		t.Fatalf("banned login error = %v, want ErrUserNotActive", err)
	}
}

// TestCoreCRUD covers groups, threads and posts including moderations rules.
func TestCoreCRUD(t *testing.T) {
	d := setupDB(t)
	ctx := context.Background()
	pool := d.Pool()

	admin := createUser(t, pool, "root", "root@e.com")
	if err := store.NewUserStore(pool).AssignRole(ctx, admin.ID, models.RoleAdmin); err != nil {
		t.Fatalf("assign admin: %v", err)
	}

	groupSvc := services.NewGroupService(store.NewGroupStore(pool), store.NewThreadStore(pool))
	threadSvc := services.NewThreadService(store.NewGroupStore(pool), store.NewThreadStore(pool))
	postSvc := services.NewPostService(store.NewPostStore(pool), store.NewThreadStore(pool))

	// group validation
	if _, err := groupSvc.Create(ctx, services.GroupInput{Name: "A", Slug: "BAD_SLUG", Description: "d"}); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("bad slug error = %v, want ErrInvalidInput", err)
	}

	g, err := groupSvc.Create(ctx, services.GroupInput{Name: "General", Slug: "general", Description: "Main group"})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	if _, err := groupSvc.Create(ctx, services.GroupInput{Name: "General", Slug: "general"}); !errors.Is(err, services.ErrConflict) {
		t.Fatalf("duplicate slug error = %v, want ErrConflict", err)
	}

	th, err := threadSvc.Create(ctx, services.CreateThreadInput{
		GroupID: g.ID, AuthorID: admin.ID, Title: "Welcome", Body: "First post",
	})
	if err != nil {
		t.Fatalf("Create thread: %v", err)
	}
	if th.PostCount != 1 {
		t.Fatalf("thread post_count = %d, want 1", th.PostCount)
	}
	got, err := threadSvc.Get(ctx, th.ID)
	if err != nil {
		t.Fatalf("Get thread: %v", err)
	}
	if _, err := threadSvc.Get(ctx, th.ID); err != nil {
		t.Fatalf("Get thread again: %v", err)
	}
	got, err = threadSvc.Get(ctx, th.ID)
	if err != nil {
		t.Fatalf("Get thread thrice: %v", err)
	}
	if got.Views != 2 {
		t.Fatalf("thread views = %d, want 2 after three Gets", got.Views)
	}

	// reply
	reply, err := postSvc.Create(ctx, th.ID, admin.ID, "A reply")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	summary, err := threadSvc.Get(ctx, th.ID)
	if err != nil {
		t.Fatalf("Get thread after reply: %v", err)
	}
	if summary.PostCount != 2 {
		t.Fatalf("post_count = %d, want 2", summary.PostCount)
	}

	// edit own post
	edited, err := postSvc.Update(ctx, reply.ID, admin.ID, "Edited reply")
	if err != nil {
		t.Fatalf("Update post: %v", err)
	}
	if edited.Body != "Edited reply" || edited.EditedAt == nil {
		t.Fatalf("edited post = %+v", edited)
	}

	// lock thread -> new replies rejected
	locked := true
	if _, err := threadSvc.Update(ctx, th.ID, services.UpdateThreadInput{Locked: &locked}, admin.ID, []string{models.RoleAdmin, models.RoleMember}); err != nil {
		t.Fatalf("lock thread: %v", err)
	}
	if _, err := postSvc.Create(ctx, th.ID, admin.ID, "Locked out"); !errors.Is(err, services.ErrThreadLocked) {
		t.Fatalf("reply to locked thread error = %v, want ErrThreadLocked", err)
	}

	// group cannot be deleted while a thread exists; thread delete frees it
	if err := groupSvc.Delete(ctx, g.ID); !errors.Is(err, services.ErrNotEmpty) {
		t.Fatalf("delete non-empty group error = %v, want ErrNotEmpty", err)
	}
	if err := threadSvc.Delete(ctx, th.ID, admin.ID, []string{models.RoleAdmin}); err != nil {
		t.Fatalf("delete thread: %v", err)
	}
	if err := groupSvc.Delete(ctx, g.ID); err != nil {
		t.Fatalf("delete empty group: %v", err)
	}
}

// TestAdminService covers user management and forum settings updates.
func TestAdminService(t *testing.T) {
	d := setupDB(t)
	ctx := context.Background()
	pool := d.Pool()

	users := store.NewUserStore(pool)
	admin := createUser(t, pool, "boss", "boss@e.com")
	if err := users.AssignRole(ctx, admin.ID, models.RoleAdmin); err != nil {
		t.Fatalf("assign admin: %v", err)
	}
	member := createUser(t, pool, "worker", "worker@e.com")

	svc := services.NewAdminService(users, store.NewAttachmentStore(pool), store.NewSettingsStore(pool))

	items, total, err := svc.ListUsers(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if int(total) != 2 || len(items) != 2 {
		t.Fatalf("ListUsers total = %d items = %d, want 2/2", total, len(items))
	}

	roles := []string{models.RoleModerator, models.RoleMember}
	status := models.UserStatusBanned
	updated, err := svc.UpdateUser(ctx, member.ID, &roles, &status)
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.Status != models.UserStatusBanned || !contains(updated.Roles, models.RoleModerator) {
		t.Fatalf("updated user = status %q roles %v", updated.Status, updated.Roles)
	}

	if _, err := svc.UpdateUser(ctx, member.ID, &[]string{"superuser"}, nil); !errors.Is(err, services.ErrInvalidInput) {
		t.Fatalf("bad role error = %v, want ErrInvalidInput", err)
	}

	fs, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if fs.ForumName != "" {
		t.Fatalf("GetSettings fresh = %q, want empty", fs.ForumName)
	}
	fs, err = svc.UpdateSettings(ctx, services.ForumSettings{ForumName: "Renamed Forum"})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if fs.ForumName != "Renamed Forum" {
		t.Fatalf("UpdateSettings = %q", fs.ForumName)
	}
	fs, err = svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings after update: %v", err)
	}
	if fs.ForumName != "Renamed Forum" {
		t.Fatalf("GetSettings = %q, want Renamed Forum", fs.ForumName)
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
