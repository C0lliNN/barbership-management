package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("APP_ENV", "")

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
	if cfg.Env != "development" {
		t.Errorf("expected default env development, got %q", cfg.Env)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("APP_ENV", "production")

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
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		logLevel string
	}{
		{"non-numeric port", "abc", "info"},
		{"out-of-range port", "70000", "info"},
		{"bad log level", "8080", "trace"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PORT", tt.port)
			t.Setenv("LOG_LEVEL", tt.logLevel)
			if _, err := Load(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
