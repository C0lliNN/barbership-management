package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("APP_ENV", "test") // bypass DATABASE_URL requirement
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_CONN_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level info, got %q", cfg.LogLevel)
	}
	if cfg.Env != "test" {
		t.Errorf("expected env test, got %q", cfg.Env)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("expected default DBMaxConns 10, got %d", cfg.DBMaxConns)
	}
	if cfg.DBConnTimeout != 5*time.Second {
		t.Errorf("expected default DBConnTimeout 5s, got %v", cfg.DBConnTimeout)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/barbershop?sslmode=disable")
	t.Setenv("DB_MAX_CONNS", "20")
	t.Setenv("DB_CONN_TIMEOUT", "10s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected log level debug, got %q", cfg.LogLevel)
	}
	if cfg.Addr() != ":9090" {
		t.Errorf("expected addr :9090, got %q", cfg.Addr())
	}
	if cfg.DatabaseURL != "postgres://user:pass@db:5432/barbershop?sslmode=disable" {
		t.Errorf("unexpected DatabaseURL: %q", cfg.DatabaseURL)
	}
	if cfg.DBMaxConns != 20 {
		t.Errorf("expected DBMaxConns 20, got %d", cfg.DBMaxConns)
	}
	if cfg.DBConnTimeout != 10*time.Second {
		t.Errorf("expected DBConnTimeout 10s, got %v", cfg.DBConnTimeout)
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name         string
		port         string
		logLevel     string
		appEnv       string
		databaseURL  string
		dbMaxConns   string
		dbConnTimeout string
	}{
		{
			name:     "non-numeric port",
			port:     "abc",
			logLevel: "info",
			appEnv:   "test",
		},
		{
			name:     "out-of-range port",
			port:     "70000",
			logLevel: "info",
			appEnv:   "test",
		},
		{
			name:     "bad log level",
			port:     "8080",
			logLevel: "trace",
			appEnv:   "test",
		},
		{
			name:        "missing DATABASE_URL in non-test env",
			port:        "8080",
			logLevel:    "info",
			appEnv:      "development",
			databaseURL: "",
		},
		{
			name:       "DB_MAX_CONNS zero",
			port:       "8080",
			logLevel:   "info",
			appEnv:     "test",
			dbMaxConns: "0",
		},
		{
			name:       "DB_MAX_CONNS over limit",
			port:       "8080",
			logLevel:   "info",
			appEnv:     "test",
			dbMaxConns: "101",
		},
		{
			name:          "invalid DB_CONN_TIMEOUT",
			port:          "8080",
			logLevel:      "info",
			appEnv:        "test",
			dbConnTimeout: "notaduration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)
			t.Setenv("LOG_LEVEL", tt.logLevel)
			t.Setenv("APP_ENV", tt.appEnv)
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("DB_MAX_CONNS", tt.dbMaxConns)
			t.Setenv("DB_CONN_TIMEOUT", tt.dbConnTimeout)
			if _, err := Load(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestDBHostPort(t *testing.T) {
	tests := []struct {
		dsn      string
		expected string
	}{
		{"postgres://user:secret@localhost:5432/db", "localhost:5432"},
		{"postgres://user:secret@db.example.com:5432/db?sslmode=require", "db.example.com:5432"},
		{"", ""},
	}
	for _, tt := range tests {
		cfg := Config{DatabaseURL: tt.dsn}
		got := cfg.DBHostPort()
		if got != tt.expected {
			t.Errorf("DBHostPort(%q) = %q, want %q", tt.dsn, got, tt.expected)
		}
	}
}
