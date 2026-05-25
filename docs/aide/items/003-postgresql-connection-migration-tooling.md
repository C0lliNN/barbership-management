# Item 003: PostgreSQL Connection & Migration Tooling

**Stage:** 0 — Foundation & Scaffolding
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-24

---

## Description

Wire in **PostgreSQL connectivity** via a pgxpool connection pool and a
**golang-migrate** migration runner with an initial baseline migration. This item
turns the stub `internal/database` package into real, testable infrastructure and
upgrades `/ready` from a dumb 200 to a genuine DB-connectivity probe.

Scope:
- `internal/database`: a `Pool` constructor wrapping `pgxpool.New`, a `Ping` helper,
  and a `Close` function — composable with the rest of the app's dependency graph.
- `internal/config`: extend `Config` with DB DSN fields (`DatabaseURL` or component
  parts) and validate them on startup.
- `backend/migrations/`: SQL migration files managed by golang-migrate; the initial
  file is a "baseline" migration (creates the `schema_migrations` tracking table via
  golang-migrate's own bookkeeping, plus a `-- no-op` SQL body so the migration tree
  is valid and versioned from the start).
- `Makefile` (or `scripts/`): `migrate-up`, `migrate-down`, `migrate-status` targets
  using the golang-migrate CLI.
- `cmd/api/main.go`: connect the pool on startup, run pending migrations, wire the
  pool's Ping into `/ready`, and close the pool on shutdown.
- Tests: a unit test for the new config fields; an integration test (requires a live
  Postgres) that opens a pool, pings, runs migrations up, asserts the baseline
  migration was applied, then runs down.

No domain tables yet — those arrive in Item 007 (identity/tenancy schema). This item
establishes the migration *infrastructure* and proves the toolchain end-to-end.

---

## Acceptance Criteria

- [x] `internal/database` exposes:
  - `New(ctx, cfg) (*pgxpool.Pool, error)` — builds a pool from config with a
    configurable `max_conns` and connect timeout; logs pool creation.
  - `Ping(ctx, pool) error` — sends a single `SELECT 1` to verify connectivity.
  - `Close(pool)` — gracefully closes the pool (called on shutdown).
- [x] `internal/config` adds `DatabaseURL` (full DSN string, e.g.
  `postgres://user:pass@host:5432/dbname?sslmode=disable`) and `DBMaxConns` (default
  `10`) to `Config`. `Load()` returns an error if `DATABASE_URL` is empty in
  non-test environments.
- [x] `backend/internal/database/migrations/` directory exists with:
  `000001_baseline.up.sql` and `000001_baseline.down.sql`. Files follow the
  `{version}_{title}.{direction}.sql` naming convention expected by golang-migrate.
- [x] `Makefile` provides:
  - `make migrate-up` — applies all pending migrations.
  - `make migrate-down` — rolls back the most recent migration.
  - `make migrate-status` — prints the current applied version.
  - Each target reads `DATABASE_URL` from the environment or `.env`.
- [x] `cmd/api/main.go` updated:
  - Opens a DB pool after config load; logs the DSN host:port (not the password).
  - Runs `database.RunMigrations()` programmatically on startup (uses embedded FS).
  - Passes the pool to `NewRouter` (implements `DBPinger`).
  - Calls `database.Close(pool)` in the graceful-shutdown sequence (via `defer`).
- [x] `GET /ready` returns **200** when DB ping succeeds; returns **503** with a JSON
  error body when the DB is unreachable (no change in `/health` semantics).
- [x] Integration test (`//go:build integration`; skips when `DATABASE_URL` unset):
  opens a pool, runs migrations up, verifies version=1, rolls back, re-applies.
- [x] `go build ./...`, `go vet ./...`, `APP_ENV=test go test -short ./...` all pass.
- [x] Manual validation with live Postgres complete (all 13 checks passed).

---

## Implementation Steps

### 1. Extend Config (`internal/config`)

Add fields to `Config`:
```go
DatabaseURL  string        // DATABASE_URL env var
DBMaxConns   int32         // DB_MAX_CONNS, default 10
DBConnTimeout time.Duration // DB_CONN_TIMEOUT, default 5s
```
- `Load()`: read `DATABASE_URL` from `os.Getenv`. Treat an empty value as an error
  (with a clear message) unless `APP_ENV=test`.
- Add `DB_MAX_CONNS` (default `10`, validate > 0 and ≤ 100) and
  `DB_CONN_TIMEOUT` (default `5s`).
- Add `Config.DBHostPort() string` helper that parses and returns `host:port` from
  the DSN for safe logging (no credentials).

### 2. Database package (`internal/database`)

Replace the stub `doc.go` with real implementation files:

```
internal/database/
  doc.go          # package comment (keep)
  pool.go         # New, Ping, Close
  pool_test.go    # integration test (build tag)
```

`pool.go`:
```go
// New creates a pgxpool.Pool from the config's DatabaseURL.
// It applies MaxConns and connect timeout, and calls Ping to
// validate connectivity before returning.
func New(ctx context.Context, cfg *config.Config, log *zap.Logger) (*pgxpool.Pool, error)

// Ping sends a SELECT 1 to verify the pool is alive.
func Ping(ctx context.Context, pool *pgxpool.Pool) error

// Close gracefully closes the pool.
func Close(pool *pgxpool.Pool)
```

Use `pgxpool.ParseConfig(dsn)` then set `MaxConns` from config before calling
`pgxpool.NewWithConfig`.

### 3. Migration files (`backend/migrations/`)

Create the directory and two files:

**`000001_baseline.up.sql`**
```sql
-- Baseline migration: establishes migration tracking.
-- No schema changes; domain tables begin in Item 007.
SELECT 1;
```

**`000001_baseline.down.sql`**
```sql
-- Rollback baseline: no-op.
SELECT 1;
```

Add `//go:embed migrations/*.sql` in a new file `internal/database/migrations.go` to
embed the SQL files into the binary:

```go
package database

import "embed"

//go:embed ../../migrations/*.sql
var Migrations embed.FS
```

> **Note on embed path:** `embed.FS` paths are relative to the Go source file.
> Adjust if the directory layout differs. Alternatively, place the
> `migrations/` directory inside `internal/database/migrations/` to keep the embed
> path simple (`//go:embed migrations/*.sql` from within `internal/database/`).
> Confirm the preferred layout during implementation and document the decision.

### 4. Migration runner wiring

In `internal/database/migrate.go`:
```go
// RunMigrations applies all pending migrations from the embedded FS.
// It returns the number of applied migrations (0 if already up-to-date).
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, fs embed.FS, log *zap.Logger) (int, error)
```

Use `golang-migrate/migrate/v4` with the `pgx/v5` source driver and the
`iofs` source driver to read from `embed.FS`:
- `migrate.NewWithInstance` with `iofs.New(fs, "migrations")` as source and
  `pgx5.WithInstance(conn, &pgx5.Config{})` as database driver.
- Call `m.Up()` — treat `migrate.ErrNoChange` as success (0 migrations applied).
- Log applied count if available via `m.Version()`.

> **Open decision (confirm before implementing):** `golang-migrate` with pgx/v5
> requires the `github.com/golang-migrate/migrate/v4/database/pgx/v5` driver
> package. Verify this is the correct import path for the pgx/v5 database driver
> (it may be `database/pgx5` or similar depending on the migrate version). Pin the
> exact import path in the Decisions log once verified.

### 5. Makefile targets

Add (or create) `backend/Makefile`:
```makefile
DATABASE_URL ?= $(shell cat .env 2>/dev/null | grep DATABASE_URL | cut -d= -f2-)

.PHONY: migrate-up migrate-down migrate-status

migrate-up:
	migrate -database "$(DATABASE_URL)" -source file://migrations up

migrate-down:
	migrate -database "$(DATABASE_URL)" -source file://migrations down 1

migrate-status:
	migrate -database "$(DATABASE_URL)" -source file://migrations version
```

Document in README that `migrate` CLI from `golang-migrate` must be installed
(`go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`).

### 6. Wiring into `cmd/api/main.go`

Update startup sequence:
```
config.Load()
→ logging.New(cfg)
→ database.New(ctx, cfg, log)        // open pool + ping
→ database.RunMigrations(...)        // apply pending
→ http.NewRouter(cfg, log, pool)     // pass pool to router
→ server.ListenAndServe()
```
On shutdown:
```
server.Shutdown(ctx)
→ database.Close(pool)
→ logger.Sync()
```

Log the DB host:port (not DSN) on successful connect. Log migration count on startup.

### 7. Update `/ready` handler

In `internal/http/handlers.go`, update the ready handler to accept a `pinger`
interface (or the pool directly):
```go
type DBPinger interface {
    Ping(ctx context.Context) error
}
```
- If `pinger` is nil or ping succeeds → 200 `{"status":"ready"}`.
- If ping fails → 503 `{"status":"not_ready","error":"database unreachable"}`.

Keep `/health` unchanged (always 200, process-level only).

### 8. Add dependencies and tidy

```bash
cd backend
go get github.com/jackc/pgx/v5
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/pgx/v5   # adjust path if needed
go get github.com/golang-migrate/migrate/v4/source/iofs
go mod tidy
```

### 9. Tests

- **Config unit test** (`internal/config/config_test.go`): add cases for
  `DATABASE_URL` missing (error), present (parsed), `DB_MAX_CONNS` default and
  invalid values.
- **Integration test** (`internal/database/pool_test.go`, build tag `//go:build integration`
  or `t.Skip` when `DATABASE_URL` is empty):
  - `New(...)` succeeds.
  - `Ping(...)` returns nil.
  - `RunMigrations(...)` applies baseline (0 or 1 migration applied, no error).
  - Query `SELECT version FROM schema_migrations` (or golang-migrate's table) to
    assert version = 1.
  - `migrate-down` equivalent: run `m.Steps(-1)` and assert version is 0.

### 10. Verify and document

- `go build ./...`, `go vet ./...`, `go test -short ./...` pass.
- Manually: start Postgres (Docker), set `DATABASE_URL`, run `go run ./cmd/api`,
  observe `/ready` 200 and migration log. Stop Postgres, observe `/ready` 503.
- `make migrate-up` / `make migrate-down` work.

---

## Testing Strategy

| Layer | Tool | When |
|-------|------|------|
| Config unit | `go test` (stdlib) | Always (no DB needed) |
| Handler unit | `httptest` + mock pinger | Always |
| DB integration | `go test -tags integration` or skip when `DATABASE_URL` unset | Requires local Postgres |
| Migration smoke | Part of integration test | Requires local Postgres |
| Manual | `curl /ready`, `make migrate-up/down` | Before marking complete |

**No end-to-end framework needed yet.** The DB-less tests remain fast; the integration
tests gate the "real Postgres" validation and are skipped in environments without a DB.

---

## Dependencies

- **Upstream:** Item 001 (repo layout ✅), Item 002 (API skeleton ✅).
- **Downstream (enables):**
  - Item 004 (docker-compose wires `DATABASE_URL` to this config).
  - Item 007 (first domain schema migrations build on this migration runner).
  - All items requiring DB access use this pool.
- **New Go dependencies:**
  - `github.com/jackc/pgx/v5` — DB driver + pgxpool.
  - `github.com/golang-migrate/migrate/v4` — migration runner.
  - `github.com/golang-migrate/migrate/v4/database/pgx/v5` — pgx database driver for migrate.
  - `github.com/golang-migrate/migrate/v4/source/iofs` — embed.FS source driver.
- **External tool (developer machine only):** `golang-migrate` CLI for `make migrate-*`
  targets (`go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`).

---

## Testing Prerequisites

### Required Services

| Service | Version | Start Command | Port |
|---------|---------|---------------|------|
| PostgreSQL | 15 or 16 | `docker run --rm -e POSTGRES_USER=barber -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=barbershop -p 5432:5432 postgres:16` | 5432 |

> For integration tests and manual validation only. Unit/config tests run without Postgres.

### Environment Configuration

**Environment variables:**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes (non-test) | — | Full DSN: `postgres://barber:secret@localhost:5432/barbershop?sslmode=disable` |
| `DB_MAX_CONNS` | No | `10` | Pool maximum connections |
| `DB_CONN_TIMEOUT` | No | `5s` | Pool connection timeout |
| `PORT` | No | `8080` | API listen port (unchanged) |
| `LOG_LEVEL` | No | `info` | Log level (unchanged) |
| `APP_ENV` | No | `development` | `test` disables `DATABASE_URL` required check |

**Secrets to set (example):**
```bash
export DATABASE_URL="postgres://barber:secret@localhost:5432/barbershop?sslmode=disable"
```

**Ports that must be available:**
- `5432` — PostgreSQL
- `8080` — API (unchanged)

### Manual Validation Checklist

- [ ] **Build succeeds:** `cd backend && go build ./...`
- [ ] **Vet passes:** `cd backend && go vet ./...`
- [ ] **Short tests pass (no DB):** `cd backend && go test -short ./...`
- [ ] **Postgres started:** `docker run --rm -e POSTGRES_USER=barber -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=barbershop -p 5432:5432 -d postgres:16`
- [ ] **DATABASE_URL set:** `export DATABASE_URL="postgres://barber:secret@localhost:5432/barbershop?sslmode=disable"`
- [ ] **Integration tests pass:** `cd backend && go test -tags integration ./internal/database/...`
  - Pool opens and pings successfully.
  - Baseline migration applied (version = 1).
  - Rollback succeeds (version = 0).
- [ ] **Application starts:** `cd backend && go run ./cmd/api`
  - Startup logs show: DB host:port connected, `N migration(s) applied`, server listening.
- [ ] **`/health` still works:** `curl -i localhost:8080/health` → `200 {"status":"ok"}`
- [ ] **`/ready` with DB up:** `curl -i localhost:8080/ready` → `200 {"status":"ready"}`
- [ ] **`/ready` with DB down:** Stop Postgres container; within DB connect timeout:
  `curl -i localhost:8080/ready` → `503 {"status":"not_ready","error":"database unreachable"}`
- [ ] **`make migrate-down`:** `cd backend && make migrate-down` rolls back the baseline; no errors.
- [ ] **`make migrate-up`:** `cd backend && make migrate-up` re-applies it; no errors.
- [ ] **`make migrate-status`:** prints current version.

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `go build ./...` | exits 0, no output |
| `go test -short ./...` | all pass, no DB required |
| `go test -tags integration ./internal/database/...` | pool opens, migrations apply and roll back |
| `GET /health` | `200 {"status":"ok"}` |
| `GET /ready` (DB up) | `200 {"status":"ready"}` |
| `GET /ready` (DB down) | `503 {"status":"not_ready","error":"database unreachable"}` |
| Startup log | JSON line with DB host:port, migration count |
| `make migrate-up` | exits 0, baseline migration at version 1 |
| `make migrate-down` | exits 0, rolled back to version 0 |

### Validation Results

```markdown
## Validation Results
- [x] Service started: PostgreSQL 16 (Docker) ✅
- [x] Application started successfully: startup log shows "database connected db=localhost:5432" and "migrations: applied successfully schema_version=1" ✅
- [x] Database tables verified: schema_migrations table created by golang-migrate (version=1 after up) ✅
- [x] Seed data verified: N/A (no seed data in this item)
- [x] API endpoints verified: /health → 200 {"status":"ok"} ✅ | /ready (DB up) → 200 {"status":"ready"} ✅ | /ready (DB down) → 503 {"error":"database unreachable","status":"not_ready"} ✅
- [x] Screenshots captured: N/A (no UI)
- [x] go build ./...: ✅  go vet ./...: ✅  APP_ENV=test go test -short ./...: ✅
- [x] Integration test (go test -tags integration ./internal/database/...): pool open ✅, ping ✅, migrations up (version=1) ✅, rollback (version=0/ErrNilVersion) ✅, re-apply ✅
- [x] migrate-up: "1/u baseline" exit 0 ✅ | migrate-down: "1/d baseline" exit 0 ✅ | migrate-status: "1" after up, "error: no migration" after down ✅
```

---

## Decisions & Trade-offs

**Resolved before implementation (confirmed with user during create-item):**

- **DB driver = `github.com/jackc/pgx/v5` (pgxpool, native API)** — confirmed by user.
  Using `pgxpool.New` / `pgxpool.ParseConfig` directly rather than the `database/sql`
  shim. Trade-off: commits to the pgx API surface for all DB code; no `sql.DB`
  portability. Benefit: best performance, full PostgreSQL feature access, idiomatic
  with golang-migrate's pgx/v5 database driver.

- **Migration tool = `github.com/golang-migrate/migrate/v4`** — confirmed by user.
  SQL-file migrations in `backend/migrations/`, embedded into the binary via
  `embed.FS`. Programmatic `m.Up()` on startup; CLI via `make migrate-*` for
  development convenience. Trade-off: two invocation paths (in-process + CLI) to keep
  in sync, but both read the same embedded/file SQL.

- **`schema_migrations` naming** — golang-migrate uses its own version-tracking table
  (`schema_migrations` by default for the postgres driver). No conflict with future
  app tables; do not create a second tracking table.

**Resolved during implementation:**

- **golang-migrate pgx/v5 driver import path = `github.com/golang-migrate/migrate/v4/database/pgx/v5`** —
  confirmed correct. The package registers itself as `"pgx5"` via `init()`, so DSNs
  must use the `pgx5://` scheme (not `postgres://`). The driver internally uses `database/sql` +
  `github.com/jackc/pgx/v5/stdlib`, not `pgxpool` directly. A `toPgx5URL` helper
  converts `postgres://` → `pgx5://` before passing to golang-migrate.
  **CLI note:** the `migrate` binary must be installed with `-tags pgx5`:
  `go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`.
  The default binary has no database drivers built in. Makefile's `PGX5_URL` variable
  does the `postgres://` → `pgx5://` scheme substitution automatically.

- **Migrations directory placement = `backend/internal/database/migrations/`** —
  co-located with the `database` package, enabling the simplest embed directive
  (`//go:embed migrations/*.sql` from `internal/database/migrations.go`). The `embed.FS`
  variable (`Migrations`) is exported so integration tests can reference it directly.
  Trade-off: `make migrate-*` targets point to `internal/database/migrations` instead
  of a top-level `migrations/` dir — documented in Makefile comments.

- **`RunMigrations` signature** — uses `(databaseURL string, fs embed.FS, log *zap.Logger) error`
  rather than `(ctx, pool, fs, log)`. Rationale: golang-migrate's pgx/v5 driver manages
  its own connection internally (via `database/sql`), so passing the `pgxpool.Pool` is
  not useful. The DSN is reused from config. Ctx not needed because golang-migrate
  doesn't accept one at the `migrate.NewWithSourceInstance` level.

- **Startup migration behavior = `logger.Fatal`** — confirmed crash on failure.
  Running the service against a dirty or partially-migrated schema is unsafe.

- **`DBPinger` interface location = `internal/http`** — the interface is defined
  alongside the handler that uses it, avoiding an import cycle. `*pgxpool.Pool`
  satisfies it via its native `Ping(ctx context.Context) error` method, so no
  adapter needed. `main.go` passes the pool directly to `NewRouter`.

- **`/ready` handler refactored to closure** — `handleReady(pinger DBPinger) gin.HandlerFunc`
  replaces the previous bare `handleReady` function. Nil pinger → always 200 (used
  by tests passing `nil`). Non-nil pinger → probe on every request.

- **Scope expansion: `Dockerfile` + `docker-compose.yml` + `Makefile` dev targets** —
  added post-spec at user request, beyond the original Item 003 scope.
  - `backend/Dockerfile`: multi-stage build (`golang:1.25-alpine` builder →
    `alpine:3.21` runtime); `CGO_ENABLED=0 -trimpath`; includes `ca-certificates`
    (Mercado Pago TLS) and `tzdata` (Brazil timezones); migrations embedded in binary.
  - `docker-compose.yml` (project root): `postgres:16` + `api` services; postgres
    health-checked before api starts; named volume `postgres_data` for persistence;
    compose project name `barbershop`.
  - `backend/.env.example`: template for local dev — copy to `backend/.env`.
  - `backend/Makefile` new targets: `make dev` (full stack in Docker),
    `make dev-down`, `make dev-logs`, `make dev-local` (postgres in Docker + API via
    `go run`), `make db-start`, `make db-stop`.
  - Note: the docker-compose frontend service is intentionally absent — it will be
    wired in Item 004 once the Next.js scaffold (Item 005) exists.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move the Stage 0 deliverable **"PostgreSQL connection + migration tooling"** from 📋 → ✅.
- Keep Stage 0 row at 🚧 (Items 004–006 remain).
- Update `docs/aide/vision.md` §5.1 if the final exact import paths or decisions
  differ from what was pinned here (they should already be updated from `create-item`,
  but verify).
- Only update rows corresponding to **Item 003**; do not tick deliverables owned by
  other items.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 003` to implement
this work item.
