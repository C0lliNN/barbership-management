# Work Queue 002

**Date:** 2026-05-31
**Sources:** `docs/aide/vision.md`, `docs/aide/roadmap.md`, `docs/aide/progress.md`
**Covers:** Stage 2 (Service Catalog & Staff Management) → Stage 3 (Availability Engine & One-off Booking)
**Batch size:** 10 items (~1 week)

This queue picks up after Stage 1 completes (Items 007–010 in queue-001). Items 011–015
finish Stage 2; items 016–020 complete Stage 3. Each item is testable locally against
a running `docker-compose` stack. Dependencies flow top-to-bottom within the queue.

---

### Item 011: Services CRUD API
Implement the tenant-scoped services API: create, read, update, and delete services
(name, price in BRL cents, duration in minutes). Enforce tenant isolation at the
repository layer — a shop can only see and manage its own services. Validate inputs:
non-empty name, positive price and duration. Include migrations for the `service` table
and automated tests covering CRUD operations, validation failures, and cross-tenant
access rejection.

### Item 012: Shop Working Hours API
Implement per-weekday working-hours management for a shop: set and retrieve open/close
times per day (Mon–Sun), with support for days-off (closed). Validate that open time
is before close time and that all values are valid times. Include the `shop_hours` (or
`shop_schedule`) migration and tests covering happy paths, validation errors, and
tenant scoping.

### Item 013: Barber Management API
Implement barber (staff) management within a shop: an Owner can add a barber (link an
existing user or invite by email), assign that barber individual working hours
(per-weekday, overriding shop defaults or marking as unavailable), and remove a barber
from the shop. Expose list and detail endpoints. Include migrations, role-check (Owner
only for mutations), tenant isolation, and tests for invite, list, update hours, and
remove flows.

### Item 014: Per-Barber Service Availability API
Implement the many-to-many relationship between barbers and services: an Owner can
specify which services a barber offers, and retrieve a barber's offered services. This
data will constrain the availability engine in Item 016. Include migration for the
`barber_service` (or `staff_service`) join table, CRUD endpoints, and tests covering
assignment, removal, list, and cross-tenant rejection.

### Item 015: Service Catalog & Staff Management Frontend (Owner)
Build the Owner admin screens for the data created in Items 011–014: manage services
(create/edit/delete with price and duration), set shop working hours (weekly schedule
grid), manage barbers (add/list/remove), set a barber's working hours, and assign
services to a barber. All screens sit behind the authenticated Owner shell from Item
010, are mobile-first, and use pt-BR labels. This completes Stage 2.

### Item 016: Availability Engine (Backend)
Implement the slot-availability calculator: given a shop, barber (or "any"), a list of
services, and a target date, return the open time slots for that day. Slot computation
must respect shop working hours, the barber's individual working hours, the combined
service duration, and all existing bookings (no overlaps). All times are assumed to be
in America/Sao_Paulo — the system does not store or convert between timezones. Include
a comprehensive unit test suite covering back-to-back bookings, day-boundary edge cases,
multi-service durations, and the "any available barber" path.

### Item 017: Booking Data Model & State Machine (API)
Define and migrate the `booking` schema: booking ID, shop (tenant), barber, customer,
list of booked services, start/end times, status, cancellation metadata, and created/
updated timestamps. Implement the booking state machine: requested → confirmed →
completed / no-show / cancelled, with valid transition enforcement. Include repository
methods and tests that verify invalid transitions are rejected and state changes persist
correctly.

### Item 018: Customer Booking Creation API
Implement the booking creation endpoint: accept a barber (or "any"), selected
service(s), and a requested time slot; validate the slot is still available (using the
availability engine from Item 016); create the booking in `requested` state; and return
the booking record. Use a database-level lock or serializable transaction to prevent
double-booking under concurrent requests. Include tests for successful booking,
slot-already-taken conflict (concurrent), invalid slot (outside hours), unknown
service, and tenant isolation.

### Item 019: Booking Retrieval, Cancellation & Reschedule API
Implement: (a) list/view endpoints for a customer's bookings and a barber's bookings;
(b) cancellation with a configurable cutoff-window rule (e.g., no cancellation within
N hours of appointment) and slot release; (c) reschedule as cancel + rebook in a
single transaction. Include tests covering cancellation within/outside the cutoff
window, reschedule to an available vs. taken slot, and that cancelled slots become
available for new bookings.

### Item 020: Customer Booking Flow Frontend
Build the customer-facing booking UI: pick barber (or "any") → select service(s) (with
computed total price and duration) → pick a date → choose an available time slot →
confirm booking. Also provide: a "My Bookings" page listing upcoming and past bookings,
and a cancel/reschedule action (respecting the cutoff rule). Screens are mobile-first
with pt-BR copy and connect to the APIs from Items 016–019. This completes Stage 3.

---

## Next Step

Select an item (start with Item 011, or the next unstarted Stage-1 item from queue-001
if Stage 1 is not yet complete) and start a **new chat session**. Run
`/speckit.aide.create-item` with the item description to create its detailed work-item
specification.
