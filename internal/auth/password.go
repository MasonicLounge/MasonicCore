package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// ErrInvalidPassword is returned when a password does not match the stored hash.
var ErrInvalidPassword = errors.New("invalid password")

// Params defines the argon2id parameters used for hashing.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams are OWASP-recommended settings for argon2id.
var DefaultParams = Params{
	Memory:      64 * 1024, // 64 MiB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// PasswordHasher derives and checks argon2id password hashes.
type PasswordHasher struct {
	params Params
}

// NewPasswordHasher creates a password hasher with sane defaults.
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{params: DefaultParams}
}

// Hash derives an argon2id encoded hash from password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength)

	enc := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.params.Memory, h.params.Iterations, h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return enc, nil
}

// Verify checks whether password matches the stored argon2id hash.
// Malformed hashes are treated as non-matching without leaking clues.
func (h *PasswordHasher) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, nil
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("parse version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("parse parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	if subtle.ConstantTimeCompare(key, expected) != 1 {
		return false, nil
	}
	return true, nil
}
