package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config holds the runtime configuration for the API service. Values are loaded
// from the environment with sensible defaults.
type Config struct {
	Port            int
	LogLevel        string
	Env             string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration

	// Database
	DatabaseURL   string
	DBMaxConns    int32
	DBConnTimeout time.Duration
}

var validLogLevels = map[string]struct{}{
	"debug": {},
	"info":  {},
	"warn":  {},
	"error": {},
}

// Load reads configuration from environment variables, applies defaults, and
// validates the result. It returns a typed error when a value is invalid.
//
// DATABASE_URL is required unless APP_ENV=test.
func Load() (Config, error) {
	cfg := Config{
		Port:            8080,
		LogLevel:        "info",
		Env:             "development",
		ShutdownTimeout: 10 * time.Second,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		DBMaxConns:      10,
		DBConnTimeout:   5 * time.Second,
	}

	if v := os.Getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT %q: must be an integer", v)
		}
		cfg.Port = port
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("invalid PORT %d: must be between 1 and 65535", cfg.Port)
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if _, ok := validLogLevels[cfg.LogLevel]; !ok {
		return Config{}, fmt.Errorf("invalid LOG_LEVEL %q: must be one of debug, info, warn, error", cfg.LogLevel)
	}

	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Env = v
	}

	// Database configuration
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" && cfg.Env != "test" {
		return Config{}, fmt.Errorf("DATABASE_URL is required (set APP_ENV=test to bypass)")
	}

	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 || n > 100 {
			return Config{}, fmt.Errorf("invalid DB_MAX_CONNS %q: must be an integer between 1 and 100", v)
		}
		cfg.DBMaxConns = int32(n)
	}

	if v := os.Getenv("DB_CONN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid DB_CONN_TIMEOUT %q: %w", v, err)
		}
		cfg.DBConnTimeout = d
	}

	return cfg, nil
}

// Addr returns the listen address derived from the configured port.
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}

// DBHostPort parses the DatabaseURL and returns "host:port" for safe logging.
// Returns an empty string if the URL is empty or unparseable.
func (c Config) DBHostPort() string {
	if c.DatabaseURL == "" {
		return ""
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return ""
	}
	return u.Host // "host:port" or just "host" when port is default
}
