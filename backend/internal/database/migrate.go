package database

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	// Register the pgx/v5 database driver under the "pgx5" scheme.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

// RunMigrations applies all pending SQL migrations from the embedded FS and
// logs the outcome. It treats migrate.ErrNoChange as success.
//
// The caller should treat a non-nil error as fatal: running the service against
// an inconsistent schema is unsafe. The recommended pattern in main is:
//
//	if err := database.RunMigrations(...); err != nil {
//	    logger.Fatal("migrations failed", zap.Error(err))
//	}
func RunMigrations(databaseURL string, fs embed.FS, log *zap.Logger) error {
	src, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5URL(databaseURL))
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Warn("migration source close error", zap.Error(srcErr))
		}
		if dbErr != nil {
			log.Warn("migration db close error", zap.Error(dbErr))
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("migrations: already up-to-date")
			return nil
		}
		return fmt.Errorf("applying migrations: %w", err)
	}

	ver, _, _ := m.Version()
	log.Info("migrations: applied successfully", zap.Uint("schema_version", ver))
	return nil
}

// toPgx5URL converts a postgres:// or postgresql:// DSN to the pgx5:// scheme
// that golang-migrate's pgx/v5 database driver expects.
func toPgx5URL(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") {
		return "pgx5" + dsn[len("postgres"):]
	}
	if strings.HasPrefix(dsn, "postgresql://") {
		return "pgx5" + dsn[len("postgresql"):]
	}
	// Already a pgx5:// URL or unknown scheme — pass through unchanged.
	return dsn
}
