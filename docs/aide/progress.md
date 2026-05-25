# Progress Tracker: Barbershop Management Platform (Brazil)

**Status:** Initialized
**Date:** 2026-05-24
**Sources:** `docs/aide/vision.md`, `docs/aide/roadmap.md`

## Status Legend

| Icon | Meaning |
|------|---------|
| 📋 | Planned |
| 🚧 | In Progress |
| ✅ | Complete |
| ⏸️ | Deferred |
| ❌ | Excluded |

## Stage Status Overview

| Stage | Name | Status |
|-------|------|--------|
| 0 | Foundation & Scaffolding | 🚧 In Progress |
| 1 | Identity, Tenancy & Authentication | 📋 Planned |
| 2 | Service Catalog & Staff Management | 📋 Planned |
| 3 | Availability Engine & One-off Booking | 📋 Planned |
| 4 | Barber Daily Schedule View | 📋 Planned |
| 5 | Payments: One-off (Mercado Pago) | 📋 Planned |
| 6 | Subscriptions & Recurring Billing | 📋 Planned |
| 7 | Notifications & Reminders | 📋 Planned |
| 8 | Owner Reporting Dashboard | 📋 Planned |
| 9 | PWA Polish, LGPD & Hardening | 📋 Planned |

---

## Stage 0 — Foundation & Scaffolding

**Status:** 🚧 In Progress

### Deliverables
- ✅ Go API service skeleton (router, config, structured logging, graceful shutdown)
- ✅ PostgreSQL connection + migration tooling
- ✅ `GET /health` and `/ready` endpoints
- 📋 Next.js PWA skeleton (SSR, manifest, service worker, pt-BR locale base)
- ✅ `docker-compose` local dev — all three services: postgres, API, frontend stub (`make dev`); `make dev-local` + `make frontend-local` for fast iteration
- 📋 CI pipeline: build, lint, test for Go and frontend
- ✅ Repository layout reflecting domain boundaries

### Acceptance Criteria
- [ ] `docker-compose up` brings up API, DB, and frontend
- [x] `GET /health` returns 200; migrations run cleanly up and down ✅
- [ ] Frontend renders pt-BR landing page and is installable (Lighthouse PWA pass)
- [ ] CI is green on a trivial test for each side

---

## Stage 1 — Identity, Tenancy & Authentication

**Status:** 📋 Planned

### Deliverables
- 📋 Data model: tenant/shop, user, role, membership; tenant ID scoping on all tables
- 📋 Shop sign-up flow (create shop + first Owner account)
- 📋 Authentication (email/password; JWT or server session) and login
- 📋 Authorization middleware: role checks + tenant-scoping at repository layer
- 📋 Frontend: sign-up, login, role-aware authenticated shell

### Acceptance Criteria
- [ ] New user can create a shop and log in as Owner
- [ ] Shop A cannot read or mutate Shop B's data (isolation test)
- [ ] Role-protected endpoints reject unauthorized roles
- [ ] Auth works end-to-end through the frontend

---

## Stage 2 — Service Catalog & Staff Management

**Status:** 📋 Planned

### Deliverables
- 📋 Services CRUD: name, price (BRL), duration
- 📋 Shop working hours (per weekday open/close)
- 📋 Barber management: invite/add barbers, set working hours
- 📋 Per-barber service availability
- 📋 Frontend Owner admin screens (services, hours, staff)

### Acceptance Criteria
- [ ] Owner can create/edit/delete services with price and duration
- [ ] Owner can add a barber and set working hours + offered services
- [ ] Data persists and is correctly tenant-scoped
- [ ] Validation: no negative price/duration; open < close

---

## Stage 3 — Availability Engine & One-off Booking (no payment)

**Status:** 📋 Planned

### Deliverables
- 📋 Availability engine (shop hours + barber hours + service duration + existing bookings)
- 📋 Booking model + state machine (requested → confirmed → completed/no-show/cancelled)
- 📋 Customer booking flow (barber → service(s) → date → slot → confirm)
- 📋 Multi-service bookings: computed total duration and price
- 📋 Cancellation & reschedule with cutoff-window rule

### Acceptance Criteria
- [ ] Selecting services sums duration/price and constrains slot length
- [ ] Engine never offers overlapping/out-of-hours slots (edge-case tests: back-to-back, day boundaries, TZ)
- [ ] Customer can create, view, and cancel a booking via frontend
- [ ] Concurrent attempts for same slot cannot both succeed (no double-book)

---

## Stage 4 — Barber Daily Schedule View

**Status:** 📋 Planned

### Deliverables
- 📋 Barber "My Day" view (time-ordered: customer, services, time, status; mobile-first)
- 📋 Status actions: completed / no-show (optional: in-progress)
- 📋 Day navigation + empty-state handling
- 📋 Owner can view any barber's day

### Acceptance Criteria
- [ ] Barber sees only their own appointments for the day, in correct order
- [ ] Marking completed/no-show updates state and persists
- [ ] View renders cleanly and quickly on a mid-range mobile viewport

---

## Stage 5 — Payments: One-off (Mercado Pago, Pix + Card)

**Status:** 📋 Planned

### Deliverables
- 📋 Mercado Pago integration (sandbox): create payment for booking (Pix + card)
- 📋 Checkout flow (Pix QR/copy-paste; card tokenized client-side)
- 📋 Webhook endpoint: idempotent event processing; payment drives booking confirmation
- 📋 Payment records linked to bookings; refund handling per cancellation rules

### Acceptance Criteria
- [ ] One-off booking paid via Pix in sandbox moves booking to confirmed
- [ ] Card payment path works end-to-end in sandbox
- [ ] Webhooks idempotent: duplicate events do not double-charge/duplicate state
- [ ] Failed/expired Pix leaves booking unconfirmed (slot released per rule)

---

## Stage 6 — Subscriptions & Recurring Billing

**Status:** 📋 Planned

### Deliverables
- 📋 Subscription plan definitions (price, period, included services/quota)
- 📋 Customer self-subscribe + recurring billing via Mercado Pago (card; Pix if supported)
- 📋 Entitlement check at booking: quota consumption; over-quota → one-off fallback
- 📋 Subscription lifecycle (active, past-due, paused, cancelled) + failed-renewal handling
- 📋 Frontend: plan selection/subscribe, "cuts remaining" indicator, manage/cancel

### Acceptance Criteria
- [ ] Owner defines a plan; customer subscribes and is billed in sandbox
- [ ] Plan-covered booking consumes quota; quota resets per cycle
- [ ] Over-quota booking routes to one-off payment
- [ ] Simulated failed renewal → past-due; entitlement reflects it (no free over-quota)
- [ ] Recurring webhook events processed idempotently

---

## Stage 7 — Notifications & Reminders

**Status:** 📋 Planned

### Deliverables
- 📋 Booking confirmation notification on confirm
- 📋 Appointment reminder (scheduled job) ahead of appointment
- 📋 Channels: in-app/PWA + email (pt-BR templates)
- 📋 Notification preferences / opt-out
- 📋 Notifier abstraction extensible to future WhatsApp channel

### Acceptance Criteria
- [ ] Customer receives confirmation on booking and reminder before appointment
- [ ] Reminder scheduling reliable and idempotent (no duplicate sends)
- [ ] Notifier abstraction supports new channel without changing call sites

---

## Stage 8 — Owner Reporting Dashboard

**Status:** 📋 Planned

### Deliverables
- 📋 Dashboard: upcoming bookings, daily/weekly revenue, active subscribers, no-show rate
- 📋 Tenant-scoped, role-restricted to Owner/Manager
- 📋 Mobile-friendly summary cards + basic trends

### Acceptance Criteria
- [ ] Metrics match underlying data for a seeded scenario
- [ ] Dashboard restricted to Owner/Manager and tenant-scoped

---

## Stage 9 — PWA Polish, LGPD Compliance & Hardening

**Status:** 📋 Planned

### Deliverables
- 📋 pt-BR localization pass; BRL formatting; BR timezone correctness verified
- 📋 PWA polish: install prompts, offline caching for core read screens, loading/empty/error states
- 📋 LGPD: consent capture, data access/export, deletion flows; PII retention review
- 📋 Security hardening: authz review, rate limiting, secrets, dependency audit; minimize PCI scope
- 📋 Observability: structured logs, payment/booking audit trail, metrics & alerting

### Acceptance Criteria
- [ ] Lighthouse PWA + accessibility checks pass on core flows
- [ ] LGPD data export and deletion work for a customer account
- [ ] No card data persisted; payment audit trail reconstructs payment history
- [ ] Security checklist complete; no high-severity issues outstanding

---

## Out-of-Scope (Excluded from v1)

Tracked from the vision so they aren't lost; revisit post-v1.

- ❌ Native iOS/Android apps (PWA only for v1)
- ⏸️ WhatsApp booking/reminders integration (fast-follow; notifier interface prepped in Stage 7)
- ❌ Marketplace / cross-shop discovery
- ❌ Inventory / retail product sales (POS)
- ❌ Advanced analytics / BI, marketing, loyalty points
- ❌ Multi-currency / multi-country (Brazil only)
- ❌ Payroll / barber commission calculation
- ❌ Native offline-first booking (basic PWA caching only)

---

## Next Step

Review this progress file. Then, in a **new chat session**, run
`/speckit.aide.create-queue` to generate the first batch of prioritized work items.
