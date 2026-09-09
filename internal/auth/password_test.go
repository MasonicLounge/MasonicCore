package auth

import (
	"strings"
	"testing"
)

func TestPasswordHasherRoundTrip(t *testing.T) {
	h := NewPasswordHasher()

	encoded, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("unexpected hash format: %s", encoded)
	}

	ok, err := h.Verify("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("Verify() = false, want true for correct password")
	}

	ok, err = h.Verify("wrong password", encoded)
	if err != nil {
		t.Fatalf("Verify(wrong) error = %v", err)
	}
	if ok {
		t.Fatal("Verify(wrong) = true, want false")
	}
}

func TestPasswordHasherUniqueSalts(t *testing.T) {
	h := NewPasswordHasher()

	a, err := h.Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	b, err := h.Hash("same-password")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if a == b {
		t.Fatal("two hashes of the same password are identical; salt is not random")
	}
}

func TestPasswordHasherRejectsMalformed(t *testing.T) {
	h := NewPasswordHasher()

	ok, err := h.Verify("password", "not-a-hash")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if ok {
		t.Fatal("Verify() = true for malformed hash, want false")
	}
}
