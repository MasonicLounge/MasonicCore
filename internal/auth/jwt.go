package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// RefreshTokenBytes is the amount of randomness a refresh token carries.
	RefreshTokenBytes = 32
	// KeyContext is the context key carrying the authenticated user claims.
	keyContext = "auth.user"
)

// ErrNoClaims marks a request without authenticated user context.
var ErrNoClaims = errors.New("no authenticated user")

// Claims are the JWT payload fields for an authenticated user.
type Claims struct {
	Username string   `json:"username,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// UserInfo is the identity attached to request context.
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Roles    []string  `json:"roles"`
}

// Manager signs and verifies access tokens.
type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewManager creates a JWT Manager.
func NewManager(secret string, issuer string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, ttl: ttl}
}

// issue is the shared signing path for access tokens.
func (m *Manager) issue(userID uuid.UUID, username string, roles []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// NewAccessToken mints a short-lived access token for a user.
func (m *Manager) NewAccessToken(userID uuid.UUID, username string, roles []string) (string, error) {
	return m.issue(userID, username, roles, m.ttl)
}

// Parse verifies an access token and returns its claims.
func (m *Manager) Parse(raw string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// NewRefreshToken generates an opaque refresh token and returns it with its
// SHA-256 hex digest (the only form that should be persisted).
func NewRefreshToken() (token string, hash string, err error) {
	buf := make([]byte, RefreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

// HashRefreshToken computes the storage digest for a refresh token.
func HashRefreshToken(token string) (string, error) {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]), nil
}

// FromContext returns the authenticated user attached by middleware.
func FromContext(ctx context.Context) (*UserInfo, error) {
	claims, ok := ctx.Value(keyContext).(*UserInfo)
	if !ok || claims == nil {
		return nil, ErrNoClaims
	}
	return claims, nil
}

// WithUser stores the authenticated user in context.
func WithUser(ctx context.Context, info *UserInfo) context.Context {
	return context.WithValue(ctx, keyContext, info)
}
