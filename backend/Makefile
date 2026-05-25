# Barbershop Management Platform — backend Makefile
#
# Prerequisites:
#   Docker (for dev/db-* targets)
#   golang-migrate CLI (with pgx5 driver, for migrate-* targets):
#     go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
#
# Quick start:
#   make dev          → build images and start full backend in Docker (postgres + API)
#   make dev-local    → start postgres in Docker, run API locally via go run (fastest)

# ── Config ────────────────────────────────────────────────────────────────────
# docker compose project is defined by docker-compose.yml at the repo root.
COMPOSE := docker compose -f ../docker-compose.yml --project-directory ..

# DATABASE_URL for migrate-* targets and dev-local.
# Read from backend/.env if present; otherwise falls back to the dev default.
DATABASE_URL ?= $(shell grep -s '^DATABASE_URL=' .env | cut -d= -f2-)
DATABASE_URL ?= postgres://barber:secret@localhost:5432/barbershop?sslmode=disable

# golang-migrate requires pgx5:// scheme; convert automatically.
PGX5_URL := $(subst postgres://,pgx5://,$(subst postgresql://,pgx5://,$(DATABASE_URL)))

MIGRATIONS_DIR := internal/database/migrations

.PHONY: build vet test-short test \
        dev dev-down dev-logs dev-local \
        db-start db-stop \
        migrate-up migrate-down migrate-status

## ── Build & test ─────────────────────────────────────────────────────────────

build:
	go build ./...

vet:
	go vet ./...

# Run all tests except those requiring a live database.
test-short:
	APP_ENV=test go test -short ./...

# Run all tests including integration tests (requires DATABASE_URL + live postgres).
test:
	go test -tags integration ./...

## ── Local development — full stack in Docker ─────────────────────────────────

# Build images and start the full backend (postgres + API) in detached containers.
# Use this for a production-like local environment.
dev:
	$(COMPOSE) up --build -d
	@echo ""
	@echo "  Backend running:"
	@echo "    API  → http://localhost:8080/health"
	@echo "    DB   → localhost:5432 (user: barber, db: barbershop)"
	@echo ""
	@echo "  Logs:   make dev-logs"
	@echo "  Stop:   make dev-down"

# Stop all services and remove containers (the postgres data volume is preserved).
dev-down:
	$(COMPOSE) down

# Follow API container logs (Ctrl-C to stop).
dev-logs:
	$(COMPOSE) logs -f api

# Start only postgres in Docker and run the API directly via 'go run'.
# Faster iteration: no image rebuild needed on code changes — just re-run.
# Reads DATABASE_URL from backend/.env if present (copy from .env.example first).
dev-local: db-start
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

## ── Database only ────────────────────────────────────────────────────────────

# Start only the postgres service (used by dev-local, migrate-*, and integration tests).
db-start:
	$(COMPOSE) up -d postgres
	@echo "Waiting for postgres to be healthy..." && \
	    until $(COMPOSE) exec -T postgres pg_isready -U barber -q 2>/dev/null; \
	    do sleep 1; done && echo "postgres ready."

# Stop the postgres service (data volume is preserved).
db-stop:
	$(COMPOSE) stop postgres

## ── Migrations ───────────────────────────────────────────────────────────────

# Apply all pending migrations.
migrate-up:
	migrate -database "$(PGX5_URL)" -source "file://$(MIGRATIONS_DIR)" up

# Roll back the most recent migration.
migrate-down:
	migrate -database "$(PGX5_URL)" -source "file://$(MIGRATIONS_DIR)" down 1

# Print the current applied migration version.
migrate-status:
	migrate -database "$(PGX5_URL)" -source "file://$(MIGRATIONS_DIR)" version
