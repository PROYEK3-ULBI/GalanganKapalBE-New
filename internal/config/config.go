package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	AppEnv               string
	AppPort              string
	CorsAllowedOrigins   []string
	DatabaseURL          string
	DatabaseMigrationURL string
	JWTSecret            string
	JWTExpiryHours       int
}

// Load reads environment variables (optionally from .env file) and returns Config.
// Returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	// Try to load .env file - ignore error if file does not exist (e.g. in production)
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("APP_PORT", "8080"),
		CorsAllowedOrigins:   splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		DatabaseMigrationURL: os.Getenv("DATABASE_MIGRATION_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
	}

	expiry, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}
	cfg.JWTExpiryHours = expiry

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.DatabaseMigrationURL == "" {
		return fmt.Errorf("DATABASE_MIGRATION_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	return nil
}

// IsProduction returns true when running in production mode.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
