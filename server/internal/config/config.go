package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every setting the server actually reads.
type Config struct {
	ServerAddress   string
	AlphaVantageKey string
	// DatabaseURL enables the Postgres store (Supabase session pooler,
	// port 5432). Empty means no store: all searches use live sources.
	DatabaseURL string
	// SupabaseURL (https://<ref>.supabase.co) turns on sign-in: every API
	// route except /api/v1/health needs a Supabase access token. Requires
	// DatabaseURL. Empty = API open (local use only).
	SupabaseURL string
	// AdminEmails are always admins (comma-separated ADMIN_EMAILS).
	AdminEmails []string
	// SupabasePublishableKey is the public (anon) key handed to the browser
	// via /config.js in the single-instance deployment.
	SupabasePublishableKey string
	// WebDir, when set, is the built GUI (gui/dist) served at / alongside
	// the API: the single-instance deployment.
	WebDir string
}

func Load() (*Config, error) {
	// .env is optional: values can come from the real environment instead.
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file loaded (%v); using environment variables and defaults", err)
	}

	cfg := &Config{
		ServerAddress:   getEnv("SERVER_ADDRESS", ":8080"),
		AlphaVantageKey: getEnv("ALPHA_VANTAGE_API_KEY", "demo"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		SupabaseURL:     getEnv("SUPABASE_URL", ""),
		AdminEmails:     splitList(getEnv("ADMIN_EMAILS", "")),

		SupabasePublishableKey: getEnv("SUPABASE_PUBLISHABLE_KEY", ""),
		WebDir:                 getEnv("WEB_DIR", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerAddress == "" {
		return fmt.Errorf("SERVER_ADDRESS is required")
	}
	if c.SupabaseURL != "" && c.DatabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL needs DATABASE_URL (membership is stored in the database)")
	}
	return nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
