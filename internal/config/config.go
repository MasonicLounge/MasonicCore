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

	S3Endpoint       string // MinIO/S3 endpoint, e.g. "localhost:9000"
	S3AccessKey      string
	S3SecretKey      string
	S3Bucket         string // default bucket for uploaded media
	S3Region         string
	S3UseSSL         bool
	S3ForcePathStyle bool

	MediaBaseURL string // public base prefix for stored media, e.g. "/media"
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

		S3Endpoint:       getenv("S3_ENDPOINT", ""),
		S3AccessKey:      getenv("S3_ACCESS_KEY", ""),
		S3SecretKey:      getenv("S3_SECRET_KEY", ""),
		S3Bucket:         getenv("S3_BUCKET", "masonic"),
		S3Region:         getenv("S3_REGION", "us-east-1"),
		S3UseSSL:         getBool("S3_USE_SSL", false),
		S3ForcePathStyle: getBool("S3_FORCE_PATH_STYLE", true),

		MediaBaseURL: getenv("MEDIA_BASE_URL", "/media"),
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
	if c.S3Endpoint == "" || c.S3AccessKey == "" || c.S3SecretKey == "" {
		return errors.New("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY are required")
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
