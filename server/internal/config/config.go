package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds every setting the server actually reads.
type Config struct {
	ServerAddress   string
	AlphaVantageKey string
}

func Load() (*Config, error) {
	// .env is optional: values can come from the real environment instead.
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file loaded (%v); using environment variables and defaults", err)
	}

	cfg := &Config{
		ServerAddress:   getEnv("SERVER_ADDRESS", ":8080"),
		AlphaVantageKey: getEnv("ALPHA_VANTAGE_API_KEY", "demo"),
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
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
