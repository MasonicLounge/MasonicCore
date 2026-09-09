package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string

	JWTSecret  string        // HS256 signing secret
	JWTIssuer  string        // "iss" claim value
	AccessTTL  time.Duration // access token lifetime
	RefreshTTL time.Duration // refresh session lifetime

	RefreshCookieName   string
	RefreshCookieSecure bool
}

// Load reads configuration from environment variables, applying defaults.
func Load() Config {
	return Config{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", ""),

		JWTSecret:  getenv("JWT_SECRET", ""),
		JWTIssuer:  getenv("JWT_ISSUER", "masoniccore"),
		AccessTTL:  time.Duration(getInt("ACCESS_TTL_MINUTES", 15)) * time.Minute,
		RefreshTTL: time.Duration(getInt("REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,

		RefreshCookieName:   getenv("REFRESH_COOKIE_NAME", "masonic_refresh"),
		RefreshCookieSecure: getBool("REFRESH_COOKIE_SECURE", false),
	}
}

// Validate rejects configurations that would be insecure at runtime.
func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET is required and must be at least 32 characters")
	}
	return nil
}

// SecureCookies reports whether cookies should carry the Secure flag.
func (c Config) SecureCookies() bool {
	return c.RefreshCookieSecure || c.AppEnv == "production"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}

func getBool(key string, fallback bool) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
