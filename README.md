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
| Frontend | Next.js (React, SSR) PWA — `frontend/` *(stub in Item 004; full scaffold in Item 005)* |
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
frontend/                    # PWA frontend
  server.js                  # placeholder stub (replaced by Next.js in Item 005)
  Dockerfile                 # Node 20 Alpine image (updated in Item 005)
docker-compose.yml           # local dev: postgres + API + frontend
docs/aide/                   # planning: vision, roadmap, progress, items, queue
```

---

## Local Development

### Services & Ports

| Service | URL | Notes |
|---------|-----|-------|
| Go API | http://localhost:8080 | `/health` · `/ready` |
| PostgreSQL | localhost:5432 | user `barber`, db `barbershop` |
| Frontend | http://localhost:3000 | pt-BR stub (Next.js in Item 005) |

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) (for database and/or full stack)
- Go 1.25+ (for `make dev-local` and tests)
- Node 20+ (for `make frontend-local` only)
- `make` (`sudo apt-get install make` on Debian/Ubuntu)

### First-time setup

```bash
cp backend/.env.example backend/.env
# Edit backend/.env if you need non-default credentials
```

### Option A — Full stack in Docker (`make dev`)

Builds all images and starts all three services (Postgres, API, frontend) in
detached containers. No local Go or Node toolchain needed to run.

```bash
cd backend
make dev          # docker compose up --build -d (all three services)

# Verify (health checks may take ~20s on first start)
curl localhost:8080/health   # → {"status":"ok"}
curl localhost:8080/ready    # → {"status":"ready"}
curl localhost:3000/health   # → {"status":"ok"}

# Open in browser: http://localhost:3000

make dev-logs         # follow API logs
make frontend-logs    # follow frontend logs
make dev-down         # stop + remove containers (data volume preserved)
```

### Option B — Postgres in Docker, services locally (`make dev-local` + `make frontend-local`)

Starts only the database in Docker and runs the API and frontend directly.
**Fastest for development** — code changes take effect immediately without
rebuilding Docker images.

```bash
cd backend
make dev-local    # starts postgres in Docker + runs API via go run

# In a second terminal:
cd backend
make frontend-local   # runs frontend stub via node (requires Node 20+)

# In a third terminal — verify:
curl localhost:8080/health   # → {"status":"ok"}
curl localhost:8080/ready    # → {"status":"ready"}
curl localhost:3000          # → pt-BR placeholder HTML

# Ctrl-C stops the API/frontend; postgres keeps running
make db-stop      # stop postgres when done
```

### Troubleshooting

| Symptom | Fix |
|---------|-----|
| Port already in use (3000/5432/8080) | Stop conflicting process or change the port in `.env` |
| `frontend` container unhealthy | `make dev-down && make dev` — a first-build timing issue; rarely recurs |
| Stub page shows wrong API URL | `NEXT_PUBLIC_API_URL` is set to `http://api:8080` inside Docker (container DNS); browser calls go to `localhost:8080` directly |
| Image not updated after editing `server.js` | `make dev` always passes `--build`; if still stale run `docker compose build frontend` then `make dev` |
| `make frontend-local` fails | Ensure Node 20+ is installed: `node --version` |

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

**Backend** (`backend/.env` / `backend/.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | *(required)* | Full DSN: `postgres://user:pass@host:5432/db?sslmode=disable` |
| `PORT` | `8080` | API listen port |
| `LOG_LEVEL` | `info` | One of `debug`, `info`, `warn`, `error` |
| `APP_ENV` | `development` | Set to `test` to bypass `DATABASE_URL` requirement |
| `DB_MAX_CONNS` | `10` | pgxpool max connections (1–100) |
| `DB_CONN_TIMEOUT` | `5s` | Pool connection timeout |

**Frontend** (set in `docker-compose.yml` or shell for `make frontend-local`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Frontend listen port |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | API base URL (browser-exposed; `NEXT_PUBLIC_` prefix for Next.js convention) |

Copy `backend/.env.example` to `backend/.env` and adjust for your setup.
`DATABASE_URL` is read automatically by `make dev-local` and `make migrate-*`.
