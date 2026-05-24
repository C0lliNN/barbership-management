# Barbershop Management Platform (Brazil)

A multi-tenant SaaS for managing barbershops in Brazil: online booking, monthly
subscription plans, one-off bookings, barber daily schedules, and Brazil-native
payments (Pix + cards via Mercado Pago). Delivered as a mobile-first Progressive
Web App.

Planning docs live in [`docs/aide/`](docs/aide/) (vision, roadmap, progress, queue,
and per-item specs).

## Stack

- **Backend:** Go (REST/JSON API) — `backend/`
- **Database:** PostgreSQL
- **Frontend:** Next.js (React, SSR) PWA — `frontend/` (scaffolded in Item 005)
- **Payments:** Mercado Pago (Pix + cards + recurring)

## Repository Layout

```
backend/
  cmd/api/            # API entrypoint
  internal/
    config/           # configuration loading (Item 002)
    database/         # Postgres connection + migrations (Item 003)
    http/             # router / transport (Item 002)
    identity/         # tenancy, users, roles, auth (Stage 1)
    catalog/          # services, staff, availability config (Stage 2)
    scheduling/       # availability engine (Stage 3)
    booking/          # bookings + state machine (Stage 3)
    subscription/     # plans, recurring billing, quota (Stage 6)
    payment/          # Mercado Pago integration (Stage 5)
frontend/             # Next.js PWA (Item 005)
docs/aide/            # planning artifacts (vision, roadmap, progress, items)
```

Backend packages live under `internal/` so they are private to the module by
default; each domain package maps to a boundary from the vision.

## Building the Backend

Requires Go 1.25+.

```bash
cd backend
go build ./...
go vet ./...
```

The HTTP server and database wiring arrive in Items 002–003; today `cmd/api` is a
buildable placeholder.
