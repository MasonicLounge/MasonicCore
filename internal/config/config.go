package config

import "os"

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	AppEnv      string
	HTTPAddr    string
	DatabaseURL string
}

// Load reads configuration from environment variables, applying defaults.
func Load() Config {
	return Config{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
