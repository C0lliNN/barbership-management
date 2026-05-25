//go:build integration

package database_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/config"
	"github.com/gcollin65/barbershop/internal/database"
)

func TestPoolIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	cfg := config.Config{
		DatabaseURL:   dsn,
		DBMaxConns:    5,
		DBConnTimeout: 5 * time.Second,
	}

	ctx := context.Background()
	log := zap.NewNop()

	// --- Open pool ---
	pool, err := database.New(ctx, &cfg, log)
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer database.Close(pool)

	// --- Ping ---
	if err := database.Ping(ctx, pool); err != nil {
		t.Fatalf("database.Ping: %v", err)
	}

	// --- Run migrations (up) ---
	if err := database.RunMigrations(dsn, database.Migrations, log); err != nil {
		t.Fatalf("database.RunMigrations: %v", err)
	}

	// --- Verify version = 1 via migrate instance ---
	migrateURL := toPgx5URL(dsn)
	src, err := iofs.New(database.Migrations, "migrations")
	if err != nil {
		t.Fatalf("iofs.New: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, migrateURL)
	if err != nil {
		t.Fatalf("creating migrator for verification: %v", err)
	}
	defer func() { m.Close() }() //nolint:errcheck

	ver, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("m.Version after Up: %v", err)
	}
	if dirty {
		t.Error("expected clean state after Up, got dirty=true")
	}
	if ver != 1 {
		t.Errorf("expected schema version 1 after Up, got %d", ver)
	}

	// --- Roll back (Steps -1) ---
	if err := m.Steps(-1); err != nil {
		t.Fatalf("m.Steps(-1) rollback: %v", err)
	}

	// After rolling back all migrations, Version returns ErrNilVersion.
	_, _, err = m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		t.Fatalf("unexpected error after rollback: %v", err)
	}

	// --- Idempotency: running Up again after rollback should re-apply ---
	if err := database.RunMigrations(dsn, database.Migrations, log); err != nil {
		t.Fatalf("database.RunMigrations (re-apply): %v", err)
	}
}

// toPgx5URL mirrors the unexported helper in the database package so the test
// can build a migrate URL directly.
func toPgx5URL(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") {
		return "pgx5" + dsn[len("postgres"):]
	}
	if strings.HasPrefix(dsn, "postgresql://") {
		return "pgx5" + dsn[len("postgresql"):]
	}
	return dsn
}
