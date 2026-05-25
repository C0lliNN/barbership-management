# Barbershop Management Platform (Brazil)

A multi-tenant SaaS for managing barbershops in Brazil: online booking, monthly
subscription plans, one-off bookings, barber daily schedules, and Brazil-native
payments (Pix + cards via Mercado Pago). Delivered as a mobile-first Progressive
Web App.

Planning docs live in [`docs/aide/`](docs/aide/) (vision, roadmap, progress, queue,
and per-item specs).

---

## Stack

| Layer | Technology |
|-------|-----------|
| Backend API | Go 1.25 — REST/JSON (`backend/`) |
| HTTP framework | Gin (`github.com/gin-gonic/gin`) |
| Logging | zap (`go.uber.org/zap`, JSON encoder) |
| Database | PostgreSQL 16 |
| DB driver | pgx v5 (`github.com/jackc/pgx/v5`) |
| Migrations | golang-migrate v4 (`github.com/golang-migrate/migrate/v4`) |
| Frontend | Next.js (React, SSR) PWA — `frontend/` *(scaffolded in Item 005)* |
| Payments | Mercado Pago (Pix + cards + recurring) |

---

## Repository Layout

```
backend/
  cmd/api/                   # API entrypoint (main.go)
  internal/
    config/                  # env-based configuration
    database/                # Postgres pool, migrations, embed.FS
      migrations/            # SQL migration files (golang-migrate)
    http/                    # Gin router, middleware, handlers
    logging/                 # zap logger constructor
    identity/                # tenancy, users, roles, auth  (Stage 1)
    catalog/                 # services, staff, availability  (Stage 2)
    scheduling/              # availability engine            (Stage 3)
    booking/                 # bookings + state machine       (Stage 3)
    subscription/            # plans, quota, recurring        (Stage 6)
    payment/                 # Mercado Pago integration       (Stage 5)
  Dockerfile                 # multi-stage Go build
  Makefile                   # build, test, dev, migrate targets
  .env.example               # template — copy to .env for local dev
frontend/                    # Next.js PWA (Item 005)
docker-compose.yml           # local dev: postgres + API (frontend added in Item 004)
docs/aide/                   # planning: vision, roadmap, progress, items, queue
```

---

## Local Development

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (for database and/or full stack)
- Go 1.25+ (for `make dev-local` and tests)
- `make` (`sudo apt-get install make` on Debian/Ubuntu)

### First-time setup

```bash
cp backend/.env.example backend/.env
# Edit backend/.env if you need non-default credentials
```

### Option A — Full stack in Docker (`make dev`)

Builds the Go binary inside a container and starts both the API and Postgres.
No local Go toolchain needed to run.

```bash
cd backend
make dev          # docker compose up --build -d

# Verify
curl localhost:8080/health   # → {"status":"ok"}
curl localhost:8080/ready    # → {"status":"ready"}

make dev-logs     # follow API logs
make dev-down     # stop + remove containers (data volume preserved)
```

### Option B — Postgres in Docker, API locally (`make dev-local`)

Starts only the database in Docker and runs the API directly with `go run`.
**Fastest for development** — code changes take effect on the next `make dev-local`
without rebuilding a Docker image.

```bash
cd backend
make dev-local    # docker compose up -d postgres && go run ./cmd/api

# In another terminal:
curl localhost:8080/health   # → {"status":"ok"}
curl localhost:8080/ready    # → {"status":"ready"}

# Ctrl-C stops the API; postgres keeps running
make db-stop      # stop postgres when you're done
```

---

## Building & Testing

All commands run from `backend/`:

```bash
make build        # go build ./...
make vet          # go vet ./...
make test-short   # unit tests only (no DB required)
make test         # all tests including integration (requires DATABASE_URL)
```

---

## Database Migrations

Migrations live in `backend/internal/database/migrations/` as plain SQL files and
are **embedded in the binary** at build time — no separate migration step needed
in production. The API runs pending migrations automatically on startup.

For manual inspection or rollback during development:

```bash
# Prerequisites: golang-migrate CLI with pgx5 driver
go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

cd backend
make db-start        # ensure postgres is running
make migrate-status  # show current version
make migrate-up      # apply pending migrations
make migrate-down    # roll back one migration
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | *(required)* | Full DSN: `postgres://user:pass@host:5432/db?sslmode=disable` |
| `PORT` | `8080` | API listen port |
| `LOG_LEVEL` | `info` | One of `debug`, `info`, `warn`, `error` |
| `APP_ENV` | `development` | Set to `test` to bypass `DATABASE_URL` requirement |
| `DB_MAX_CONNS` | `10` | pgxpool max connections (1–100) |
| `DB_CONN_TIMEOUT` | `5s` | Pool connection timeout |

Copy `backend/.env.example` to `backend/.env` and adjust for your setup.
`DATABASE_URL` is read automatically by `make dev-local` and `make migrate-*`.
