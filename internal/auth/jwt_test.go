package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAccessTokenLifecycle(t *testing.T) {
	m := NewManager("test-secret-test-secret-test-secret-0000", "masoniccore", time.Minute)

	uid := uuid.New()
	raw, err := m.NewAccessToken(uid, "alice", []string{"member", "admin"})
	if err != nil {
		t.Fatalf("NewAccessToken() error = %v", err)
	}

	claims, err := m.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Subject != uid.String() {
		t.Errorf("Subject = %q, want %q", claims.Subject, uid.String())
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want alice", claims.Username)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "member" || claims.Roles[1] != "admin" {
		t.Errorf("Roles = %v, want [member admin]", claims.Roles)
	}
	if claims.Issuer != "masoniccore" {
		t.Errorf("Issuer = %q, want masoniccore", claims.Issuer)
	}
}

func TestAccessTokenWrongSecret(t *testing.T) {
	mint := NewManager("test-secret-test-secret-test-secret-0000", "masoniccore", time.Minute)
	verify := NewManager("different-secret-different-secret-00", "masoniccore", time.Minute)

	raw, err := mint.NewAccessToken(uuid.New(), "alice", nil)
	if err != nil {
		t.Fatalf("NewAccessToken() error = %v", err)
	}

	if _, err := verify.Parse(raw); err == nil {
		t.Fatal("Parse() with wrong secret succeeded, want error")
	}
}

func TestAccessTokenWrongIssuer(t *testing.T) {
	mint := NewManager("test-secret-test-secret-test-secret-0000", "other-issuer", time.Minute)
	verify := NewManager("test-secret-test-secret-test-secret-0000", "masoniccore", time.Minute)
	raw, err := mint.NewAccessToken(uuid.New(), "alice", nil)
	if err != nil {
		t.Fatalf("NewAccessToken() error = %v", err)
	}

	if _, err := verify.Parse(raw); err == nil {
		t.Fatal("Parse() with mismatched issuer succeeded, want error")
	}
}

func TestAccessTokenExpired(t *testing.T) {
	m := NewManager("test-secret-test-secret-test-secret-0000", "masoniccore", -time.Minute)
	raw, err := m.NewAccessToken(uuid.New(), "alice", nil)
	if err != nil {
		t.Fatalf("NewAccessToken() error = %v", err)
	}

	if _, err := m.Parse(raw); err == nil {
		t.Fatal("Parse() of expired token succeeded, want error")
	}
}

func TestRefreshTokenFormat(t *testing.T) {
	token, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	if token == "" || hash == "" || token == hash {
		t.Fatalf("unexpected token/hash: token=%q hash=%q", token, hash)
	}

	recomputed, err := HashRefreshToken(token)
	if err != nil {
		t.Fatalf("HashRefreshToken() error = %v", err)
	}
	if recomputed != hash {
		t.Fatalf("HashRefreshToken() = %q, want %q", recomputed, hash)
	}
}
