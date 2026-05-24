# Roadmap: Barbershop Management Platform (Brazil)

**Status:** Draft v1
**Date:** 2026-05-24
**Source:** `docs/aide/vision.md`

This roadmap breaks the vision into incremental, demonstrable stages. Each stage
delivers a version that can be run locally, demoed, and tested against explicit
acceptance criteria. Stages are ordered so each builds on the previous one, moving
from foundation → core booking → payments → subscriptions → polish.

**Conventions**
- Backend: Go (REST/JSON API). DB: PostgreSQL. Frontend: Next.js (React, SSR) PWA.
- Payments: Mercado Pago (sandbox until Stage 5+).
- Every stage ends with: app runs locally, automated tests pass, manual demo script
  succeeds.

---

## Stage 0 — Foundation & Scaffolding

**Goal:** A runnable skeleton for both backend and frontend with the local dev loop,
database migrations, and CI in place. No business features yet.

**Deliverables**
- Go API service skeleton (router, config, structured logging, graceful shutdown).
- PostgreSQL connection + migration tooling (e.g., `golang-migrate` or `goose`).
- `GET /health` (and `/ready`) endpoints.
- Next.js PWA skeleton (SSR, installable manifest, service worker, pt-BR locale base).
- `docker-compose` for local dev (API + Postgres + frontend).
- CI pipeline: build, lint, test for both Go and frontend.
- Repository layout reflecting domain boundaries (Identity/Tenancy, Catalog,
  Scheduling, Booking, Subscription, Payment).

**Dependencies:** None.

**Acceptance criteria**
- `docker-compose up` brings up API, DB, and frontend.
- `GET /health` returns 200; migrations run cleanly up and down.
- Frontend renders a pt-BR landing page and is installable as a PWA (Lighthouse PWA
  checks pass).
- CI is green on a trivial test for each side.

---

## Stage 1 — Identity, Tenancy & Authentication

**Goal:** Multi-tenant foundation: shops, users, roles, auth, and enforced tenant
isolation. This is the backbone every later stage depends on.

**Deliverables**
- Data model: `tenant/shop`, `user`, `role` (Owner/Manager, Barber, Customer),
  membership linking users to shops with roles. Tenant ID scoping on all domain
  tables (shared-schema multi-tenancy).
- Shop sign-up flow (create shop + first Owner account).
- Authentication (email/password to start; JWT or server session) and login.
- Authorization middleware: role checks + tenant-scoping enforced at the
  repository/query layer.
- Frontend: sign-up, login, and a minimal authenticated shell (role-aware nav).

**Dependencies:** Stage 0.

**Acceptance criteria**
- A new user can create a shop and log in as Owner.
- A request scoped to Shop A cannot read or mutate Shop B's data (isolation test).
- Role-protected endpoints reject unauthorized roles (automated tests).
- Auth works end-to-end through the frontend.

---

## Stage 2 — Service Catalog & Staff Management

**Goal:** Owners configure their shop: services, hours, barbers, and per-barber
service availability — the data the booking engine will need.

**Deliverables**
- Services CRUD: name, price (BRL), duration. (e.g., Corte, Barba, Combo.)
- Shop working hours (per weekday open/close).
- Barber management: invite/add barbers, set individual working hours.
- Per-barber service availability (which barbers offer which services).
- Frontend admin screens (Owner): manage services, hours, and staff.

**Dependencies:** Stage 1.

**Acceptance criteria**
- Owner can create/edit/delete services with price and duration.
- Owner can add a barber and set that barber's working hours and offered services.
- Data persists and is correctly tenant-scoped.
- Validation: no negative price/duration; hours sane (open < close).

---

## Stage 3 — Availability Engine & One-off Booking (no payment)

**Goal:** Customers can book an appointment for selected services with a barber, with
correct slot availability and no double-booking. Payment is deferred to Stage 5.

**Deliverables**
- Availability engine: compute open slots from shop hours + barber working hours +
  service total duration + existing bookings.
- Booking model + state machine: requested → confirmed → completed / no-show /
  cancelled.
- Customer booking flow (frontend): pick barber (or "any") → service(s) → date →
  available slot → confirm.
- Multi-service bookings: total duration and price computed from selected services.
- Cancellation & reschedule with a cutoff-window rule.

**Dependencies:** Stage 2.

**Acceptance criteria**
- Selecting services correctly sums duration/price and constrains slot length.
- The engine never offers a slot that overlaps an existing booking or falls outside
  working hours (automated tests for edge cases: back-to-back, day boundaries, TZ).
- Customer can create, view, and cancel a booking through the frontend.
- Concurrent booking attempts for the same slot cannot both succeed (no double-book).

---

## Stage 4 — Barber Daily Schedule View

**Goal:** A barber opens the PWA on their phone and sees an accurate, ordered daily
schedule and can update appointment status.

**Deliverables**
- Barber "My Day" view: appointments in time order with customer, services, time,
  status (mobile-first).
- Status actions: mark completed / no-show; (optional) mark in-progress.
- Day navigation (today / next days) and empty-state handling.
- Owner can view any barber's day.

**Dependencies:** Stage 3.

**Acceptance criteria**
- Barber sees only their own appointments for the selected day, in correct order.
- Marking completed/no-show updates booking state and persists.
- View renders cleanly and quickly on a mid-range mobile viewport.

---

## Stage 5 — Payments: One-off (Mercado Pago, Pix + Card)

**Goal:** Casual customers pay per visit via Pix or card, with payment reconciled
back to the booking. Establishes the payment infrastructure.

**Deliverables**
- Mercado Pago integration (sandbox): create payment for a booking (Pix + card).
- Checkout flow in the frontend (Pix QR/copy-paste; card tokenized client-side —
  no card data on our servers).
- Webhook endpoint: idempotent processing of payment events; payment status drives
  booking confirmation.
- Payment records linked to bookings; refund handling consistent with cancellation
  rules.

**Dependencies:** Stage 3 (booking), Stage 1 (tenancy).

**Acceptance criteria**
- Customer completes a one-off booking and pays via Pix in sandbox; booking moves to
  confirmed on payment success.
- Card payment path works end-to-end in sandbox.
- Webhooks are idempotent: replayed/duplicate events do not double-charge or
  duplicate state (automated test).
- A failed/expired Pix leaves the booking unconfirmed (and slot released per rule).

---

## Stage 6 — Subscriptions & Recurring Billing

**Goal:** Shops sell monthly plans; customers self-subscribe, get billed recurrently,
and have plan-covered bookings honored against quota.

**Deliverables**
- Subscription plan definitions (Owner): price, billing period, included
  services/quota (e.g., "4 cuts/month").
- Customer self-subscribe + recurring billing via Mercado Pago (card recurrence;
  Pix recurrence if supported, else card fallback per vision).
- Entitlement check at booking time: subscriber booking consumes quota; over-quota
  falls back to one-off payment (Stage 5 path).
- Subscription lifecycle: active, past-due, paused, cancelled; handle failed
  renewals via webhooks.
- Frontend: plan selection/subscribe, "cuts remaining this cycle" indicator,
  manage/cancel subscription.

**Dependencies:** Stage 5 (payments), Stage 3 (booking).

**Acceptance criteria**
- Owner defines a plan; customer subscribes and is billed in sandbox.
- A plan-covered booking consumes quota; quota resets per billing cycle.
- An over-quota booking correctly routes to one-off payment.
- A simulated failed renewal moves the subscription to past-due and reflects in
  entitlement (no free over-quota bookings).
- Recurring webhook events processed idempotently.

---

## Stage 7 — Notifications & Reminders

**Goal:** Reduce no-shows with booking confirmations and appointment reminders.

**Deliverables**
- Booking confirmation notification on confirm.
- Appointment reminder (scheduled job) ahead of the appointment.
- Channels for v1: in-app/PWA notifications + email (pt-BR templates).
- Notification preferences / opt-out.
- (WhatsApp remains out of scope per vision; design the notifier interface so a
  WhatsApp channel can be added later without rework.)

**Dependencies:** Stage 3 (booking); benefits from Stage 5/6.

**Acceptance criteria**
- Customer receives a confirmation on booking and a reminder before the appointment.
- Reminder scheduling is reliable and idempotent (no duplicate sends).
- Notifier abstraction supports adding a channel without changing call sites.

---

## Stage 8 — Owner Reporting Dashboard

**Goal:** Owners get visibility into the health of their shop.

**Deliverables**
- Dashboard: upcoming bookings, daily/weekly revenue, active subscriber count,
  no-show rate.
- Tenant-scoped, role-restricted to Owner/Manager.
- Mobile-friendly summary cards + basic trends.

**Dependencies:** Stages 3, 5, 6 (data to report on).

**Acceptance criteria**
- Metrics match underlying data for a seeded scenario (revenue = sum of paid
  bookings/subscriptions in range; no-show rate computed correctly).
- Dashboard is restricted to Owner/Manager and tenant-scoped.

---

## Stage 9 — PWA Polish, LGPD Compliance & Hardening (Release Readiness)

**Goal:** Make v1 production-ready: localization, compliance, resilience, and
observability.

**Deliverables**
- pt-BR localization pass across all screens; BRL formatting; America/Sao_Paulo and
  BR timezone correctness verified.
- PWA polish: install prompts, offline affordances/caching for core read screens,
  loading/empty/error states.
- LGPD: consent capture, data access/export, and deletion flows; minimal PII
  retention review.
- Security hardening: authz review, rate limiting, secrets management, dependency
  audit; minimize PCI scope (confirm no card data stored).
- Observability: structured logs, payment/booking audit trail, basic metrics &
  alerting.

**Dependencies:** All prior stages.

**Acceptance criteria**
- Lighthouse PWA + accessibility checks pass on core flows.
- LGPD data export and deletion work for a customer account.
- No card data is persisted; payment audit trail reconstructs a booking's payment
  history.
- Security checklist completed; no high-severity issues outstanding.

---

## Stage Dependency Summary

```
0 → 1 → 2 → 3 → 4
                3 → 5 → 6
                3 → 7
        3,5,6 → 8
        all   → 9
```

---

## Next Step

Review this roadmap. Then, in a **new chat session**, run
`/speckit.aide.create-progress` to create the progress tracking file.
