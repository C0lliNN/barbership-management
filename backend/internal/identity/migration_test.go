//go:build integration

package identity_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()

	if err := database.RunMigrations(dsn, database.Migrations, zap.NewNop()); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, table := range []string{"shop", `"user"`, "membership"} {
		var exists bool
		row := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
			strings.Trim(table, `"`))
		if err := row.Scan(&exists); err != nil || !exists {
			t.Errorf("table %s not found after migrate up", table)
		}
	}
}
