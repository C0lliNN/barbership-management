# Work Queue 001

**Date:** 2026-05-24
**Sources:** `docs/aide/vision.md`, `docs/aide/roadmap.md`, `docs/aide/progress.md`
**Covers:** Stage 0 (Foundation & Scaffolding) → Stage 1 (Identity, Tenancy & Authentication)
**Batch size:** 10 items (~1 week)

This is the first queue. Items are ordered by dependency. Each item is testable
locally. Pick an item, start a new chat session, and run `/speckit.aide.create-item`
with its description to produce a detailed work-item spec.

---

### Item 001: Repository Structure & Go Module Layout
Initialize the Go module and establish the repository layout reflecting the vision's
domain boundaries (Identity/Tenancy, Catalog, Scheduling, Booking, Subscription,
Payment) plus shared infrastructure (config, db, http). Add a top-level layout for
the `backend/` (Go) and `frontend/` (Next.js) apps, a root README pointer, and base
`.gitignore`/`.editorconfig`. No business logic — just the skeleton directories,
package stubs, and a buildable `go build ./...`.

### Item 002: Go API Skeleton (Router, Config, Logging, Health)
Stand up the Go HTTP API service: configuration loading (env-based), structured
logging, an HTTP router, graceful shutdown, and `GET /health` + `GET /ready`
endpoints. The service must start locally and respond 200 on `/health`. Include a
minimal unit/integration test hitting the health endpoint.

### Item 003: PostgreSQL Connection & Migration Tooling
Add PostgreSQL connectivity (connection pool, configurable DSN) and a migration tool
(e.g., `golang-migrate` or `goose`) with an initial no-op/baseline migration. Provide
make targets or scripts to run migrations up/down. `/ready` should report DB
connectivity. Verify migrations apply and roll back cleanly against a local Postgres.

### Item 004: Docker Compose Local Dev Environment
Create a `docker-compose.yml` that brings up the Go API, PostgreSQL, and the frontend
together for local development, wired with env vars and depends-on/health checks.
`docker-compose up` should yield a running API (health 200), a reachable DB, and a
served frontend. Document the local dev loop in the README.

### Item 005: Next.js PWA Skeleton (SSR + pt-BR Base)
Scaffold the Next.js frontend with SSR, an installable PWA manifest, a service
worker, and a pt-BR locale foundation (i18n setup + BRL/timezone formatting helpers).
Render a simple pt-BR landing page that calls the API `/health`. Lighthouse PWA
checks should pass for installability.

### Item 006: CI Pipeline (Build, Lint, Test)
Add a CI workflow that, on push/PR, builds, lints, and tests both the Go backend and
the Next.js frontend. Include Go vet/staticcheck and frontend lint/type-check. CI must
go green on the trivial tests from Items 002 and 005. This completes Stage 0.

### Item 007: Multi-Tenant Data Model & Migrations
Define and migrate the core identity/tenancy schema: `shop` (tenant), `user`, `role`
(Owner/Manager, Barber, Customer), and `membership` linking users to shops with a
role. Enforce tenant ID scoping conventions on domain tables. Include repository-layer
helpers that require a tenant scope. Provide migration up/down and seed/fixtures for
tests.

### Item 008: Shop Sign-up & Owner Account Creation (API)
Implement the sign-up flow API: create a shop (tenant) together with its first Owner
user and membership in one transaction. Validate inputs (unique email, required shop
fields). Return the created shop + owner identity. Cover with tests including
duplicate-email and validation-failure cases.

### Item 009: Authentication & Login (API)
Implement email/password authentication: secure password hashing, a login endpoint
issuing a session/JWT, and token/session verification middleware. Include logout/token
invalidation as appropriate. Test login success, wrong-password, and unauthenticated
access to a protected stub endpoint.

### Item 010: Authorization, Tenant Isolation & Frontend Auth Screens
Add authorization middleware enforcing role checks and tenant-scoping at the
repository/query layer, and build the frontend sign-up, login, and a minimal
role-aware authenticated shell. Verify end-to-end: a user signs up a shop, logs in as
Owner, and a request scoped to Shop A cannot read/mutate Shop B's data (isolation
test). This completes Stage 1.

---

## Next Step

Select an item (start with Item 001) and start a **new chat session**. Run
`/speckit.aide.create-item` with the item description to create its detailed work-item
specification.
