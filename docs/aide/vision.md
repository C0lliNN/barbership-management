# Vision: Barbershop Management Platform (Brazil)

**Status:** Draft v1
**Date:** 2026-05-24
**Owner:** gcollin65@gmail.com

---

## 1. Project Overview

### 1.1 Problem

Running a barbershop in Brazil is painful for everyone involved:

- **Barbers / shop owners** juggle walk-ins, phone calls, WhatsApp messages, and a
  paper agenda to manage their day. They have no reliable view of who is coming,
  when, and for what service. Revenue is unpredictable and no-shows are common.
- **Customers** struggle to book a time, don't know wait times, and have no easy,
  trustworthy way to pay — especially recurring customers who would happily commit
  to a monthly plan if it were convenient.

There is strong demand in Brazil for a **recurring revenue model** (monthly
subscription for X cuts/month), but shops lack the tooling to sell and manage it.
Payments in Brazil are also distinctive: **Pix** (instant bank transfer) is the
dominant method, alongside credit/debit cards and recurring billing.

### 1.2 Solution

A **multi-tenant SaaS platform** that any barbershop in Brazil can sign up for to
manage bookings, services, staff schedules, and payments — including **monthly
subscription plans** and **one-off bookings** — with payments handled natively for
the Brazilian market (Pix + cards via Mercado Pago).

It is delivered as a **mobile-first Progressive Web App (PWA)**: customers and
barbers use it from a phone browser, installable to the home screen, with no
app-store friction.

### 1.3 Why This, Why Now

- Brazil's barbershop market is large, fragmented, and under-digitized.
- Pix adoption is near-universal, making instant and recurring digital payments
  finally frictionless.
- A subscription model smooths revenue for shops and locks in loyalty — a clear,
  monetizable wedge that generic booking tools don't address well for this market.

---

## 2. Goals & Objectives

| # | Objective | Measurable Outcome |
|---|-----------|--------------------|
| G1 | Let shops accept online bookings | A shop can go from sign-up to taking its first online booking in < 30 min |
| G2 | Enable recurring revenue | Shops can define subscription plans; customers can self-subscribe and pay monthly via Pix/card |
| G3 | Give barbers a clear daily view | A barber sees their full day (appointments, services, customer, status) on one mobile screen |
| G4 | Reduce no-shows | Automated reminders + clear booking confirmation; target measurable no-show reduction per shop |
| G5 | Frictionless Brazilian payments | Pix and card checkout for both one-off and recurring, with reconciliation back to the booking |
| G6 | Multi-tenant from day one | Multiple independent shops operate in isolation on shared infrastructure |

**Non-goals for v1** are listed in §8 (Out of Scope).

---

## 3. Target Users

### 3.1 Shop Owner / Manager (tenant admin)
Sets up the shop: services, prices, working hours, staff, and subscription plans.
Wants visibility into revenue, bookings, and subscriber count. Often not very
technical — onboarding must be simple and in Brazilian Portuguese.

### 3.2 Barber (staff)
Performs the service. Primary need: a clean, glanceable **daily schedule** on their
phone showing each appointment's time, customer, and selected services, with the
ability to mark appointments as done / no-show. May have an individual calendar
within a multi-barber shop.

### 3.3 Subscriber (recurring customer)
Pays a monthly plan and books appointments included in that plan. Wants effortless
re-booking, knowledge of how many cuts remain in their cycle, and a saved payment
method.

### 3.4 One-off Customer (walk-in / casual)
Books a single appointment, selects services, and pays per visit (Pix or card).
May or may not create a full account — booking friction must be minimal.

---

## 4. Core Features

### 4.1 Tenant / Shop Management
- Shop sign-up and onboarding (shop profile, address, hours, branding/logo).
- Multi-tenant isolation: each shop's data, staff, customers, and payments are
  segregated.
- Staff (barber) management: invite barbers, set individual working hours and
  availability.
- Role-based access: Owner/Manager vs Barber vs Customer.

### 4.2 Service Catalog
- Define services (e.g., *Corte de cabelo*, *Barba*, *Combo*, *Sobrancelha*) with
  price and duration.
- A booking can include **multiple services**; total duration and price are computed
  from the selected services.
- Per-barber service availability (some barbers may not offer all services).

### 4.3 Booking & Scheduling
- Customer-facing booking flow: pick shop → barber (or "any") → service(s) → date →
  available time slot.
- Availability engine: slots derived from shop hours, barber working hours, service
  duration, and existing bookings (no double-booking).
- Booking states: requested → confirmed → completed / no-show / cancelled.
- Cancellation and reschedule rules (e.g., cutoff window before appointment).
- **Barber daily schedule view** — the day's appointments in time order, mobile-first,
  with quick status updates.

### 4.4 Subscriptions (Recurring Plans)
- Shop defines subscription plans (e.g., "4 cuts/month — R$X", "unlimited beard
  trims", etc.) with billing period, price, and included services/quota.
- Customer self-subscribes and pays monthly (recurring billing).
- Plan entitlement is checked at booking time: a subscriber's booking consumes quota
  or is covered by the plan; over-quota bookings fall back to one-off payment.
- Subscription lifecycle: active, past-due, paused, cancelled; handle failed renewals.

### 4.5 Payments (Brazil-native)
- **Mercado Pago** integration as the payment provider.
- Supported methods: **Pix** (instant) and **credit/debit cards**.
- One-off payment for casual bookings.
- **Recurring/subscription billing** for plans (card or Pix-based recurrence per
  provider capabilities).
- Webhooks to reconcile payment status back to bookings/subscriptions.
- Refund / cancellation handling consistent with booking cancellation rules.

### 4.6 Notifications & Reminders
- Booking confirmation and appointment reminders (reduce no-shows).
- Channel for v1: in-app/PWA + email; **WhatsApp** reminders are a strong
  Brazil-fit candidate for a fast-follow (see §8).

### 4.7 Reporting (Owner)
- Basic dashboard: upcoming bookings, daily/weekly revenue, active subscribers,
  no-show rate.

---

## 5. Technical Architecture

### 5.1 Stack (confirmed)
- **Backend:** Go (Golang) 1.25+ — REST/JSON API.
  - **HTTP framework:** Gin (`github.com/gin-gonic/gin`).
  - **Logging:** zap (`go.uber.org/zap`), JSON encoder.
- **Database:** PostgreSQL.
- **Frontend:** Mobile-first **Progressive Web App** with **Server-Side Rendering (SSR)**.
  TypeScript + React with SSR (e.g., Next.js) consuming the Go API backend.
- **Payments:** Mercado Pago (Pix + cards + recurring).

> **Pinned libraries are authoritative.** Work items must align with the choices above
> rather than re-proposing alternatives. A genuinely new library need is a decision to
> confirm with the user **before** implementation — never a post-completion swap.
>
> **Pinned in Item 003:**
> - **DB driver:** `github.com/jackc/pgx/v5` — using `pgxpool` directly (native pgx API, not `database/sql` shim).
> - **Migration tool:** `github.com/golang-migrate/migrate/v4` — SQL-file migrations, embedded via `embed.FS`, CLI + programmatic API.

### 5.2 Architectural Principles
- **Multi-tenancy** baked into the data model from day one (tenant/shop ID scoping on
  every domain entity; enforce isolation at the query/repository layer). Decide
  shared-schema-with-tenant-column vs schema-per-tenant during planning — start
  shared-schema for simplicity unless a hard requirement dictates otherwise.
- **API-first:** the Go API is the single source of truth; the PWA and any future
  native app or WhatsApp integration are clients of it.
- **Clear domain boundaries:** Identity/Tenancy, Catalog, Scheduling/Availability,
  Booking, Subscription, Payment. Keep these as well-separated packages/modules.
- **Idempotent payment handling:** all payment webhooks processed idempotently;
  money state derived from provider events, not optimistic assumptions.
- **Localization:** Brazilian Portuguese (pt-BR) as the primary locale; currency in
  BRL (R$); America/Sao_Paulo and other BR timezones handled correctly.

### 5.3 Infrastructure & Deployment (initial assumptions)
- Containerized Go service + managed PostgreSQL.
- Single region (Brazil / South America) for latency and data residency comfort.
- Environment separation: **dev** and **prod** only. CI/CD with automated tests as a
  release gate (to be ratified in the project constitution).

---

## 6. Non-Functional Requirements

- **Performance:** Booking availability and schedule views should feel instant on
  mid-range Android phones over mobile networks (target sub-second perceived load for
  core screens; lean payloads).
- **Reliability:** Payment and booking operations must be transactional and
  consistent; no double-bookings, no lost payments. Webhook processing resilient to
  retries.
- **Security & Privacy:** Tenant data isolation enforced server-side; least-privilege
  roles; no card data stored on our servers (tokenize via Mercado Pago / PCI scope
  minimized). Comply with **LGPD** (Brazil's data-protection law) — consent, data
  access/deletion, minimal PII retention.
- **Scalability:** Architecture supports many shops and concurrent bookings on shared
  infra; stateless API for horizontal scaling; Postgres as the consistency backbone.
- **Accessibility & UX:** Mobile-first, installable PWA; usable one-handed; clear
  pt-BR copy; works on low-end devices and flaky connections (graceful loading/offline
  affordances where reasonable).
- **Observability:** Structured logging, payment/booking audit trail, and basic
  metrics for monitoring.

---

## 7. Constraints & Assumptions

**Constraints**
- Market is **Brazil**: pt-BR language, BRL currency, Pix-first payment expectations,
  LGPD compliance.
- Payment provider is **Mercado Pago**; capabilities and constraints of its Pix and
  recurring-billing APIs bound what we can offer.
- Backend is **Go + PostgreSQL** (fixed).
- Delivery is a **PWA** (no native app in scope for v1).

**Assumptions**
- Shops have basic smartphones and internet access.
- Customers are comfortable paying via Pix/card and using a phone web app.
- A shared-schema multi-tenant model is acceptable for the initial scale.
- Recurring billing via Mercado Pago is available and suitable for monthly plans;
  if Pix-recurrence has limitations, card-based recurrence is the fallback.

---

## 8. Out of Scope (v1)

Explicitly excluded to keep v1 focused and shippable:

- **Native iOS/Android apps** — PWA only for now (deferred; the API-first design keeps
  this open later).
- **WhatsApp booking/reminders integration** — high Brazil-fit, but a fast-follow,
  not v1 (deferred to avoid coupling launch to WhatsApp Business API onboarding).
- **Marketplace / cross-shop discovery** — each shop brings its own customers; we are
  not building a consumer discovery marketplace in v1.
- **Inventory / product sales (POS for retail products)** — services only.
- **Advanced analytics / BI, marketing campaigns, loyalty points** — beyond basic
  owner reporting.
- **Multi-currency / multi-country** — Brazil only.
- **Payroll / commission calculation for barbers** — out of scope for v1.
- **Native offline-first booking** — basic PWA caching only, not full offline sync.

---

## 9. Success Criteria

The project is successful for v1 if:

1. A new shop can self-onboard, configure services + a subscription plan, and take a
   real online booking and payment without engineering help.
2. A customer can complete a one-off booking and pay via **Pix or card** end-to-end,
   with the payment reconciled to the booking.
3. A customer can **subscribe to a monthly plan**, get billed recurrently, and have
   plan-covered bookings honored against quota.
4. A barber can open the PWA on their phone and see an accurate, ordered **daily
   schedule**, and update appointment status.
5. Tenant isolation holds: no shop can see or affect another shop's data.
6. Core booking and schedule screens load fast on a mid-range Android phone.
7. The system handles payment webhooks idempotently with no double-bookings and no
   lost/duplicated charges across a sustained test period.

---

## 10. Next Step

Review this vision. Then, in a **new chat session**, run
`/speckit.aide.create-roadmap` to generate a staged development roadmap from this
document.
