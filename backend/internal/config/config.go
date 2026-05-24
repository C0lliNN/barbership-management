package config

import (
	"fmt"
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
}

var validLogLevels = map[string]struct{}{
	"debug": {},
	"info":  {},
	"warn":  {},
	"error": {},
}

// Load reads configuration from environment variables, applies defaults, and
// validates the result. It returns a typed error when a value is invalid.
func Load() (Config, error) {
	cfg := Config{
		Port:            8080,
		LogLevel:        "info",
		Env:             "development",
		ShutdownTimeout: 10 * time.Second,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
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

	return cfg, nil
}

// Addr returns the listen address derived from the configured port.
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}
