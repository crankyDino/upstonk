package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerAddress string
	Environment   string
	JSEAPI        JSEAPIConfig
	Cache         CacheConfig
	Logging       LoggingConfig
}

type JSEAPIConfig struct {
	BaseURL string
	APIKey  string
	Timeout int
}

type CacheConfig struct {
	Enabled    bool
	TTLSeconds int
	MaxSize    int
}

type LoggingConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		JSEAPI: JSEAPIConfig{
			BaseURL: getEnv("JSE_API_BASE_URL", "https://api.jse.co.za/v1"),
			APIKey:  getEnv("JSE_API_KEY", ""),
			Timeout: 10,
		},
		Cache: CacheConfig{
			Enabled:    getEnv("CACHE_ENABLED", "true") == "true",
			TTLSeconds: 3600,
			MaxSize:    1000,
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
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
