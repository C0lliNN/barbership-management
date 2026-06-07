# Barbershop Management Platform — root Makefile
#
# Thin convenience wrapper around backend/Makefile so the full local dev stack
# can be started from the repo root with one command. For finer-grained targets
# (migrations, tests, individual logs, …) use `make -C backend <target>` or see
# backend/Makefile directly.
#
# Quick start:
#   make dev        → build images and start full stack in Docker (postgres + API + frontend)
#   make dev-local  → start postgres in Docker, then run API + frontend natively, in parallel (fastest)
#   make dev-down   → stop the Docker stack

.PHONY: dev dev-down dev-logs frontend-logs dev-local db-start db-stop

## ── Full stack in Docker ──────────────────────────────────────────────────────

dev:
	$(MAKE) -C backend dev

dev-down:
	$(MAKE) -C backend dev-down

dev-logs:
	$(MAKE) -C backend dev-logs

frontend-logs:
	$(MAKE) -C backend frontend-logs

## ── Fastest local dev: postgres in Docker, API + frontend run natively ───────

# Starts postgres in Docker, then runs the Go API (go run) and the Next.js
# frontend (npm run dev) natively, in parallel, with combined log output.
# Ctrl-C stops both; postgres keeps running (use 'make db-stop' when done).
dev-local:
	@trap 'kill 0' EXIT INT TERM; \
	$(MAKE) -C backend dev-local & \
	$(MAKE) -C backend frontend-local & \
	wait

db-start:
	$(MAKE) -C backend db-start

db-stop:
	$(MAKE) -C backend db-stop
