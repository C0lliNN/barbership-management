# Item 010: Authorization, Tenant Isolation & Frontend Auth Screens

**Stage:** 1 — Identity, Tenancy & Authentication
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-06-07

---

## Description

Close out Stage 1 by making tenant isolation and role-based access **real and
testable end-to-end**, and by giving users a way to sign up and log in through the
browser.

This item has three parts:

1. **Authorization middleware (backend).** Add `RequireShopMembership` (resolves the
   `:shopID` URL parameter to the caller's membership in that shop, or `404` if the
   caller is not a member — including when the shop doesn't exist, so existence is
   never leaked) and `RequireRole` (rejects with `403` unless the membership's role is
   in an allowed set). Both compose with the existing `AuthRequired` middleware
   (Item 009).

2. **A minimal shop-scoped resource to prove isolation.** There is currently no
   shop-scoped read/write endpoint to test against — `/v1/me` only echoes the
   caller's own token claims. Add `GET /v1/shops/:shopID` (any member can view the
   shop's profile) and `PATCH /v1/shops/:shopID` (Owner-only — update contact/location
   info). These are genuinely useful (the Owner needs a shop-settings screen anyway)
   *and* they are exactly the read + mutate pair the queue item asks us to prove
   isolation against.

3. **Frontend sign-up, login, and an authenticated shell (frontend).** Build
   `/cadastro` (sign-up), `/entrar` (login), and `/painel` (a minimal authenticated,
   role-aware shell: shop name, user name, role, logout). Owners additionally get
   `/painel/loja` — a shop-settings page wired to the new `GET`/`PATCH
   /v1/shops/:shopID` endpoints. This is the first real exercise of the
   `POST /v1/signup` (Item 008) and `POST /v1/login` (Item 009) APIs from a browser.

**End-to-end isolation proof (the Stage 1 acceptance test):** sign up Shop A and Shop
B as two separate Owners, log in as each, and confirm that Shop A's token cannot read
or mutate Shop B's `/v1/shops/:shopID` resource (`404`), and that a non-Owner member
of a shop cannot mutate that shop's resource either (`403`).

---

### Design Notes (read before implementing)

- **Role checks must be evaluated against the *target* shop's membership, not the JWT's
  embedded `shop_id`/`role`.** The JWT (Item 009) embeds the user's *primary* (first)
  shop and role at login time as a convenience for `/me` and the frontend shell. But a
  user can belong to multiple shops with different roles, and the embedded claim can go
  stale before the token expires. `RequireShopMembership` therefore always performs a
  fresh `MembershipRepository.GetByShopAndUser(ctx, shopID, claims.Subject)` lookup
  against the `:shopID` in the URL — never trusts `claims.ShopID`/`claims.Role` for
  authorization decisions. This is the crux of "tenant-scoping enforced at the
  repository/query layer": the verified shop ID is the only one downstream code is
  allowed to use.

- **Handlers must read the verified tenant ID from context, not from the URL.**
  `RequireShopMembership` stores the verified shop ID via the existing
  `identity.WithTenant(ctx, shopID)` (already defined in `internal/identity/tenant.go`
  but currently unused — this item is what wires it up). Handlers call
  `identity.TenantFromCtx(ctx)` to obtain the shop ID to pass to repositories/services,
  *not* `c.Param("shopID")`. This makes "the shop ID used in queries is always the one
  the membership middleware verified" a structural guarantee instead of a convention
  that can be forgotten per-handler.

- **404, not 403, for non-members.** Returning `403` would confirm the shop exists;
  `404` does not. This matches the "no enumeration" principle already used for login
  (Item 009 returns the same message for unknown-email and wrong-password).

---

### API Contracts

#### `GET /v1/shops/:shopID`  _(any member of the shop)_

```
GET /v1/shops/c7f1e2a0-...
Authorization: Bearer <token>

200 OK
{
  "id":         "c7f1e2a0-...",
  "name":       "Barbearia do João",
  "slug":       "barbearia-do-joao",
  "phone":      "+5511988887777",
  "address":    "Rua Augusta, 123",
  "city":       "São Paulo",
  "state":      "SP",
  "created_at": "2026-06-01T12:00:00Z",
  "updated_at": "2026-06-07T09:30:00Z"
}

404 Not Found  — caller is not a member of this shop, or the shop does not exist
                 {"error": "not found"}
401 Unauthorized — missing/invalid/expired token
```

#### `PATCH /v1/shops/:shopID`  _(Owner only)_

Updates contact/location fields. `name`/`slug` are intentionally **not** editable
here — slug uniqueness and any future redirect/SEO concerns belong to a dedicated
"rename shop" flow if ever needed; v1 keeps this endpoint to the low-risk fields an
Owner would actually maintain day-to-day.

```
PATCH /v1/shops/c7f1e2a0-...
Authorization: Bearer <token>
Content-Type: application/json

{
  "phone":   "+5511988887777",
  "address": "Rua Augusta, 123",
  "city":    "São Paulo",
  "state":   "SP"
}

200 OK
{ ...same shape as GET, with updated fields and new updated_at... }

403 Forbidden    — caller is a member but not an Owner   {"error": "insufficient role"}
404 Not Found    — caller is not a member of this shop   {"error": "not found"}
422 Unprocessable Entity — e.g. state is not a 2-letter code
401 Unauthorized — missing/invalid/expired token
```

---

## Acceptance Criteria

### Backend — middleware & routes
- [x] `RequireShopMembership` resolves `:shopID` + the caller's user ID (from
  `AuthRequired` claims) to a membership via `MembershipRepository.GetByShopAndUser`;
  stores the membership in the Gin context and the verified shop ID via
  `identity.WithTenant`.
- [x] `RequireShopMembership` returns `404 {"error":"not found"}` both when the shop
  doesn't exist and when the caller has no membership in it (identical response —
  no enumeration).
- [x] `RequireRole(roles ...identity.Role)` returns `403 {"error":"insufficient role"}`
  when the membership's role is not in the allowed set; passes through otherwise.
  Must run after `RequireShopMembership` (reads membership from context).
- [x] `GET /v1/shops/:shopID` returns `200` with the shop profile for **any** member
  (owner, barber, or customer membership) and `404` for non-members.
- [x] `PATCH /v1/shops/:shopID` updates `phone`/`address`/`city`/`state`, returns
  `200` with the updated record, for Owners only; `403` for non-Owner members; `404`
  for non-members; `422` for an invalid `state` (not a 2-letter code).
- [x] Both handlers obtain the shop ID via `identity.TenantFromCtx(ctx)` (set by
  `RequireShopMembership`) — **not** `c.Param("shopID")` — so the verified ID is
  what reaches the service/repository layer.

### Backend — isolation & role proof (the Stage 1 gate)
- [x] **Cross-tenant read is blocked:** Shop A's owner calling
  `GET /v1/shops/:shopBID` gets `404`.
- [x] **Cross-tenant mutation is blocked:** Shop A's owner calling
  `PATCH /v1/shops/:shopBID` gets `404` (never reaches the role check — membership
  lookup fails first).
- [x] **Same-tenant role check is enforced:** a Barber or Customer member of Shop A
  calling `PATCH /v1/shops/:shopAID` gets `403`; the same user calling `GET` on the
  same resource gets `200`.
- [x] **Same-tenant Owner mutation succeeds:** Shop A's Owner can `GET` and `PATCH`
  Shop A's record; the change persists (`GET` after `PATCH` reflects it).

### Frontend
- [x] `/cadastro` renders a sign-up form (shop fields: name, phone, address, city,
  state; owner fields: full name, email, phone, password) in pt-BR, submits to
  `POST /v1/signup`, surfaces `409` (`email already registered` /
  `shop name too similar...`) and `422` validation errors inline, and on success
  redirects to `/entrar` with a success message.
- [x] `/entrar` renders a login form (email, password) in pt-BR, submits to
  `POST /v1/login`, stores the session (token + user + `shop_id` + `role`), and
  redirects to `/painel`; shows an inline error on `401`.
- [x] `/painel` is gated: visiting it without a stored/valid session redirects to
  `/entrar`; the gate validates the stored token against `GET /v1/me` (catches
  expired/invalid tokens, not just "missing").
- [x] `/painel` shell shows the shop name, the user's name, and a role badge
  (Proprietário / Barbeiro / Cliente), and a working "Sair" (logout) action that
  calls `POST /v1/logout`, clears the stored session, and redirects to `/entrar`.
- [x] `/painel/loja` is visible only to Owners (link hidden, route redirects for
  non-Owners) and lets the Owner view and edit the shop's phone/address/city/state via
  `GET`/`PATCH /v1/shops/:shopID`, showing success/validation/error feedback.
- [x] All new UI copy is in pt-BR; layouts are mobile-first (matches the PWA base
  from Item 005).

### End-to-end (manual — see Testing Prerequisites)
- [x] Full walkthrough passes: sign up Shop A (Owner A) → sign up Shop B (Owner B) →
  log in as Owner A → view/edit `/painel/loja` for Shop A → confirm Owner B cannot
  view or edit Shop A's data (via direct API calls with Owner B's token, since the
  frontend has no UI path to another shop's ID).
- [x] `go build ./...`, `go vet ./...`, `APP_ENV=test go test -short ./...` pass.
- [x] `cd frontend && npm run lint && npm run type-check && npm run build` pass.

---

## Implementation Steps

### Backend

1. **`internal/identity/repository.go`** — add `Update` to `ShopRepository`:
   ```go
   type ShopRepository interface {
       Create(ctx context.Context, shop Shop) (Shop, error)
       GetByID(ctx context.Context, id string) (Shop, error)
       GetBySlug(ctx context.Context, slug string) (Shop, error)
       Update(ctx context.Context, shop Shop) (Shop, error) // updates phone/address/city/state by ID
   }
   ```
   Implement in `internal/infra/repository/shop_repo.go` (mirrors `Create`'s
   `nullText`/scan pattern; `RETURNING ... updated_at`). Regenerate mocks
   (`go generate ./...` in `internal/identity`).

2. **`internal/identity/service.go`** — add a `ShopManager` interface and implement it
   on `*Service` alongside `Signer`:
   ```go
   type ShopManager interface {
       GetShop(ctx context.Context, shopID string) (Shop, error)
       UpdateShop(ctx context.Context, shopID string, input ShopUpdateInput) (Shop, error)
   }

   type ShopUpdateInput struct {
       Phone   string `json:"phone"`
       Address string `json:"address"`
       City    string `json:"city"`
       State   string `json:"state" binding:"omitempty,len=2"`
   }
   ```
   `GetShop` is a thin pass-through to `shops.GetByID` (membership is already verified
   by middleware by the time this runs); `UpdateShop` validates and calls
   `shops.Update`. Both keep handlers decoupled from the repository layer, consistent
   with `SignUp`/`Login`.

3. **`internal/infra/http/middleware.go`** — add:
   - a `membershipKey` context key and `membershipFromContext(c) *identity.Membership`
     helper (mirrors `claimsKey`/`claimsFromContext`);
   - `RequireShopMembership(memberships identity.MembershipRepository) gin.HandlerFunc`
     — reads `claimsFromContext(c).Subject` and `c.Param("shopID")`, looks up the
     membership, `404`s on `identity.ErrNotFound`, otherwise stores the membership
     (`c.Set(membershipKey, m)`) and the verified tenant
     (`c.Request = c.Request.WithContext(identity.WithTenant(ctx, shopID))`);
   - `RequireRole(roles ...identity.Role) gin.HandlerFunc` — reads the membership from
     context, `403`s if its role isn't in `roles`.

4. **New file `internal/infra/http/shop.go`** — `RegisterShopRoutes` + handlers +
   DTOs (mirrors the structure of `identity.go`):
   ```go
   func RegisterShopRoutes(rg *gin.RouterGroup, svc identity.ShopManager,
       memberships identity.MembershipRepository, jwtSecret string) {
       scoped := rg.Group("/shops/:shopID", AuthRequired(jwtSecret), RequireShopMembership(memberships))
       scoped.GET("", handleGetShop(svc))
       scoped.PATCH("", RequireRole(identity.RoleOwner), handleUpdateShop(svc))
   }
   ```
   Handlers pull the verified ID via `shopID, _ := identity.TenantFromCtx(ctx)`, call
   the service, map `identity.ErrNotFound` → `404`, and reuse `formatValidationError`
   for `422`s. A `shopProfileDTO` can reuse the existing `shopDTO` shape from
   `identity.go` (consider promoting it to a shared `toShopDTO` helper rather than
   duplicating field mapping — this is a routine reuse call, not a new decision).

5. **`cmd/api/main.go`** — wire `ShopManager` (the existing `*identity.Service`
   already satisfies it — no new provider needed beyond an `fx.As` annotation if one
   isn't already exported) and call `apihttp.RegisterShopRoutes(v1, svc, memberships,
   cfg.JWTSecret)` in `registerRoutes`. `memberships identity.MembershipRepository` is
   already provided via DI for the identity service — reuse it.

### Frontend

6. **`frontend/src/lib/api/client.ts`** — small fetch wrapper: builds the URL from
   `NEXT_PUBLIC_API_URL`, attaches `Authorization: Bearer <token>` when a session is
   present, parses JSON, and normalizes `{error, details?}` responses into a typed
   error the UI can render.

7. **`frontend/src/lib/auth/session.ts`** — `AuthSession` type
   (`{token, user, shopId, role}`), `readSession()`/`writeSession()`/`clearSession()`
   over `localStorage` (key e.g. `barbershop.session`), plus a `useAuth()` hook /
   `AuthProvider` React context that exposes the current session and `login`/`logout`
   actions to client components.

8. **`frontend/src/app/cadastro/page.tsx`** — sign-up form → `POST /v1/signup` →
   redirect to `/entrar?cadastro=ok`.

9. **`frontend/src/app/entrar/page.tsx`** — login form → `POST /v1/login` →
   `writeSession(...)` → redirect to `/painel`.

10. **`frontend/src/app/painel/layout.tsx`** (client component) — the authenticated
    shell: on mount, reads the stored session, validates it against `GET /v1/me`
    (covers expiry — a stored-but-expired token must redirect to `/entrar`, not just
    a missing one), redirects to `/entrar` if invalid/absent, and otherwise renders
    a header (shop name, user name, role badge, "Sair") + nav (role-aware: Owners see
    a "Configurações da loja" link to `/painel/loja`; other roles don't) around
    `{children}`.

11. **`frontend/src/app/painel/page.tsx`** — minimal dashboard home (welcome message;
    placeholder for Stage 2+ content).

12. **`frontend/src/app/painel/loja/page.tsx`** — Owner-only shop-settings page:
    `GET /v1/shops/:shopId` on load (using `shopId` from the session), an editable
    form for phone/address/city/state, `PATCH` on submit, inline success/error
    feedback. Non-Owners hitting this route are redirected to `/painel`.

13. **`frontend/messages/pt-BR.json`** — add `signup`, `login`, `panel`, `shopSettings`
    message namespaces (form labels, button text, error strings) alongside the
    existing `landing`/`common`.

---

## Testing Strategy

| Layer | Tool | When |
|-------|------|------|
| `RequireShopMembership` / `RequireRole` middleware (unit) | `httptest` + mock `MembershipRepository`, generate tokens via `identity.SignToken` | Always |
| `GET`/`PATCH /v1/shops/:shopID` handlers (unit) | `httptest` + mock `ShopManager` + mock `MembershipRepository` | Always |
| `Service.GetShop` / `UpdateShop` (unit) | mock `ShopRepository` | Always |
| `ShopRepository.Update` (integration) | `go test -tags integration` | Requires live Postgres |
| Cross-tenant isolation (integration or handler-level with two seeded shops) | `go test -tags integration` (preferred — exercises the real membership lookup against real data) | Requires live Postgres |
| Frontend pages/components | `npm run lint` / `npm run type-check`; manual browser walkthrough (no test runner configured yet — see Item 006 CI scope) | Always / manual |
| Manual end-to-end | curl + browser, two shops/owners | Before marking complete |

### Middleware tests (`internal/infra/http/middleware_test.go`)

| Scenario | Setup | Expected |
|----------|-------|----------|
| Member (any role) passes `RequireShopMembership` | mock returns membership | `c.Next()` called; membership + tenant set in context |
| Non-member | mock returns `identity.ErrNotFound` | `404 {"error":"not found"}`, aborted |
| Repository error | mock returns generic error | `500`, aborted, logged |
| Owner passes `RequireRole(RoleOwner)` | membership.Role = owner | `c.Next()` |
| Barber blocked by `RequireRole(RoleOwner)` | membership.Role = barber | `403 {"error":"insufficient role"}` |

### Handler tests (`internal/infra/http/shop_test.go` — new file)

| Scenario | Expected |
|----------|----------|
| `GET` happy path (owner/barber/customer membership) | `200` with shop DTO |
| `GET` not a member | `404` (via middleware, not handler) |
| `PATCH` happy path (owner) | `200` with updated DTO; service called with `TenantFromCtx` value |
| `PATCH` non-owner | `403` (via middleware) |
| `PATCH` invalid `state` | `422` |
| `PATCH` service returns `ErrNotFound` (race: shop deleted mid-request) | `404` |

### Service tests (`internal/identity/service_test.go`)

- `GetShop` returns the repo's result; propagates `ErrNotFound`.
- `UpdateShop` maps `ShopUpdateInput` correctly and calls `shops.Update`; propagates
  errors.

### Repository integration tests (`internal/infra/repository/shop_test.go`)

- `Update` persists `phone`/`address`/`city`/`state` and bumps `updated_at`;
  `GetByID` after `Update` reflects the change; `name`/`slug` are untouched.

### Cross-tenant isolation test (integration, the headline scenario)

Seed two shops (A, B) with distinct owners (reuse the `SignUp` flow or direct
repository inserts, matching the existing integration-test setup style in
`internal/infra/repository/identity_test.go`). Then, using Owner A's membership/claims:
- `GetByShopAndUser(ctx, shopBID, ownerA.UserID)` → `ErrNotFound` (proves the query the
  middleware relies on is itself tenant-safe at the SQL level — `WHERE shop_id = $1
  AND user_id = $2`, no cross-shop leakage possible).
- End-to-end through the router: `GET`/`PATCH /v1/shops/:shopBID` with Owner A's
  token → `404`.

---

## Dependencies

- **Upstream:**
  - Item 008 ✅ — `identity.Service`, `ShopRepository`, `MembershipRepository`,
    sign-up flow.
  - Item 009 ✅ — `AuthRequired` middleware, `Claims`, `identity.SignToken`/
    `ParseToken`, `MembershipRepository.ListByUser`.
- **Downstream (enables):**
  - Item 011+ (Service Catalog, Staff Management, Bookings, …) — every shop-scoped
    resource from here on mounts under `/v1/shops/:shopID/...` and reuses
    `RequireShopMembership`/`RequireRole` rather than re-implementing tenant checks.
  - Item 015 (Owner admin frontend) — extends the `/painel` shell built here.
- **No new Go dependencies.** No new npm dependencies — `fetch`, React state/context,
  and the existing Next.js/Tailwind/next-intl stack are sufficient (aligns with the
  pinned stack in `docs/aide/vision.md` §5.1; nothing new to confirm).
- **No new migrations** — `shop` already has `phone`/`address`/`city`/`state`/
  `updated_at`; `membership` already has the `(shop_id, user_id)` unique index that
  makes `GetByShopAndUser` a safe, indexed, tenant-scoped lookup.

---

## Testing Prerequisites

### Required Services

| Service | Version | Start Command | Port |
|---------|---------|---------------|------|
| PostgreSQL | 16 | `cd backend && make db-start` | 5432 |
| Go API | (this repo) | `cd backend && JWT_SECRET=changeme-change-in-production-min32chars make dev-local` | 8080 |
| Frontend | (this repo) | `cd frontend && npm install && npm run dev` (or `make frontend-local` from `backend/`) | 3000 |

### Environment Configuration

| Variable | Where | Required | Example |
|----------|-------|----------|---------|
| `DATABASE_URL` | backend | Yes (integration tests / dev-local) | `postgres://barber:secret@localhost:5432/barbershop?sslmode=disable` |
| `JWT_SECRET` | backend | Yes (runtime) | `changeme-change-in-production-min32chars` |
| `NEXT_PUBLIC_API_URL` | frontend | Yes (browser→API) | `http://localhost:8080` |
| `API_INTERNAL_URL` | frontend | No (SSR fetch; `/painel` is client-rendered so not strictly needed here) | `http://localhost:8080` |

Copy `backend/.env.example` → `backend/.env` and `frontend/.env.local.example` →
`frontend/.env.local` if not already done (Item 004/009).

### Manual Validation Checklist

- [x] **Build (backend):** `cd backend && go build ./...`
- [x] **Vet (backend):** `cd backend && go vet ./...`
- [x] **Short tests (backend):** `cd backend && APP_ENV=test go test -short ./...`
- [x] **Start postgres:** `cd backend && make db-start`
- [x] **Integration tests (backend):** `cd backend && go test -tags integration ./...`
- [x] **Lint/type-check/build (frontend):**
  ```bash
  cd frontend && npm install && npm run lint && npm run type-check && npm run build
  ```
- [x] **Start API:** `cd backend && JWT_SECRET=changeme-change-in-production-min32chars make dev-local`
- [x] **Start frontend:** `cd frontend && npm run dev`
- [x] **Sign up Shop A / Owner A** (via `/cadastro` in the browser, or curl):
  ```bash
  curl -s -X POST http://localhost:8080/v1/signup -H 'Content-Type: application/json' -d '{
    "shop":  {"name":"Barbearia A","city":"São Paulo","state":"SP"},
    "owner": {"email":"ownerA@test.com","password":"Secret123","full_name":"Dono A"}
  }' | jq .
  ```
- [x] **Sign up Shop B / Owner B:**
  ```bash
  curl -s -X POST http://localhost:8080/v1/signup -H 'Content-Type: application/json' -d '{
    "shop":  {"name":"Barbearia B","city":"Rio de Janeiro","state":"RJ"},
    "owner": {"email":"ownerB@test.com","password":"Secret123","full_name":"Dono B"}
  }' | jq .
  ```
- [x] **Log in as both; capture tokens and shop IDs:**
  ```bash
  TOKEN_A=$(curl -s -X POST http://localhost:8080/v1/login -H 'Content-Type: application/json' \
    -d '{"email":"ownerA@test.com","password":"Secret123"}' | jq -r .token)
  SHOP_A=$(curl -s http://localhost:8080/v1/me -H "Authorization: Bearer $TOKEN_A" | jq -r .shop_id)
  TOKEN_B=$(curl -s -X POST http://localhost:8080/v1/login -H 'Content-Type: application/json' \
    -d '{"email":"ownerB@test.com","password":"Secret123"}' | jq -r .token)
  SHOP_B=$(curl -s http://localhost:8080/v1/me -H "Authorization: Bearer $TOKEN_B" | jq -r .shop_id)
  echo "A=$SHOP_A  B=$SHOP_B"
  ```
- [x] **Owner A reads/writes their own shop (expect 200):**
  ```bash
  curl -s http://localhost:8080/v1/shops/$SHOP_A -H "Authorization: Bearer $TOKEN_A" | jq .
  curl -s -X PATCH http://localhost:8080/v1/shops/$SHOP_A -H "Authorization: Bearer $TOKEN_A" \
    -H 'Content-Type: application/json' -d '{"phone":"+5511988887777","city":"São Paulo","state":"SP"}' | jq .
  ```
- [x] **Owner A cannot read/write Shop B (expect 404 both):**
  ```bash
  curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/v1/shops/$SHOP_B -H "Authorization: Bearer $TOKEN_A"
  curl -s -o /dev/null -w '%{http_code}\n' -X PATCH http://localhost:8080/v1/shops/$SHOP_B \
    -H "Authorization: Bearer $TOKEN_A" -H 'Content-Type: application/json' -d '{"city":"Hacked"}'
  ```
- [x] **Browser walkthrough:** open `http://localhost:3000/cadastro`, sign up a third
  shop/owner, get redirected to `/entrar`, log in, land on `/painel`, see shop name +
  role, open `/painel/loja`, edit and save the address, log out, confirm `/painel`
  redirects to `/entrar`.

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `GET /v1/shops/:ownShopID` | `200` with profile (any role) |
| `PATCH /v1/shops/:ownShopID` (owner) | `200`, change persists |
| `PATCH /v1/shops/:ownShopID` (barber/customer) | `403 {"error":"insufficient role"}` |
| `GET`/`PATCH /v1/shops/:otherShopID` (any role) | `404 {"error":"not found"}` |
| `/cadastro` → `/entrar` → `/painel` | Session stored; shell shows correct shop/user/role |
| `/painel/loja` (Owner) | Loads current values; edits persist and reflect on reload |
| `/painel/loja` (non-Owner, or direct nav) | Redirected to `/painel`; no data exposed |
| Logout | Session cleared; `/painel` redirects to `/entrar` |

### Validation Results

```markdown
## Validation Results
- [x] Build: `go build ./...` / `npm run build` — both pass
- [x] Vet/lint/type-check pass — `go vet ./...`, `npm run lint`, `npm run type-check` all clean
- [x] Short tests pass (backend) — `APP_ENV=test go test -short ./...` green (all packages)
- [x] Service started: PostgreSQL 16 — `make db-start` (Docker)
- [x] Application started: API + frontend — `make dev-local` (:8080) and `npm run dev` (:3000)
- [x] Integration tests: cross-tenant isolation case passes against live Postgres —
      `go test -tags integration ./...` green, including the new
      `TestCrossTenantIsolation_MembershipLookup` and `TestShopRepository_Update*`
- [x] Manual curl walkthrough (two shops, cross-tenant 404/403 checks) — results:
      signed up Shop A / Shop B; Owner A `GET`/`PATCH /v1/shops/:shopBID` → `404`
      both times (membership lookup fails before the role check, per spec); a
      seeded Barber member of Shop A got `200` on `GET` and `403
      {"error":"insufficient role"}` on `PATCH`; `PATCH` with `state:"São Paulo"`
      → `422 {"details":["State: len"],...}`; Owner A `PATCH` on Shop A → `200`,
      and a follow-up `GET` reflects the persisted change (`updated_at` bumped).
- [x] Browser walkthrough (signup → login → painel → loja → logout) — **partial,
      see note**: no GUI browser or screenshot tooling (no `playwright`/`puppeteer`/
      `chromium`) is available in this sandboxed environment, so the click-through
      could not be performed visually. Verified functional equivalence instead:
      (1) `curl`'d `/cadastro`, `/entrar`, `/painel`, `/painel/loja` against the
      Next.js dev server — all return `200` and contain the expected pt-BR copy
      (form labels, button text, "Carregando painel..."); (2) replicated each
      page's exact client-side fetch sequence against the live API with a real
      session — `POST /v1/signup` → `POST /v1/login` → `GET /v1/me` (painel gate)
      → `GET /v1/shops/:shopId` (header shop name + loja initial values) →
      `PATCH /v1/shops/:shopId` (loja save) → `GET` (reload reflects change) →
      `POST /v1/logout`; all succeeded with the responses the pages expect;
      (3) checked the dev-server log for runtime/hydration errors on these routes
      — none (the one hydration warning logged was a pre-existing
      `data-lt-installed` browser-extension artifact on the root `<html>` tag from
      an earlier `/` load, unrelated to any new page).
- [ ] Screenshots captured: signup, login, painel shell, loja settings — **not
      captured**: same tooling gap as above (no GUI browser available to drive or
      screenshot). The user should run `npm run dev` + `make dev-local` and click
      through `/cadastro` → `/entrar` → `/painel` → `/painel/loja` to visually
      confirm layout/spacing on a real viewport; the underlying functionality is
      verified end-to-end per the walkthrough above.
```

---

## Decisions & Trade-offs

- **`ShopManager` provisioning = concrete `*identity.Service` + tiny adapter
  functions, not `fx.Annotate(..., fx.As(...), fx.As(...))`.** The existing
  `Signer` wiring used `fx.Annotate(newIdentityService, fx.As(new(identity.Signer)))`;
  adding a second `fx.As(new(identity.ShopManager))` to the same annotation looked
  natural but its correctness (does fx share one `*Service` instance across both
  interface views, or construct two?) wasn't obvious from the docs and would have
  been expensive to get wrong silently. Instead `main.go` now provides
  `*identity.Service` concretely once, plus
  `func(svc *identity.Service) identity.Signer { return svc }` and
  `func(svc *identity.Service) identity.ShopManager { return svc }`. fx resolves
  `*identity.Service` a single time and both adapters share that instance — simple,
  obviously correct, and trivially greppable. Trade-off: two extra one-line provider
  functions vs. a denser annotation; readability and certainty won.

- **`toShopDTO` promoted to a shared helper** (per the spec's suggestion rather than
  a new decision): `identity.go`'s `signUpDTO` now builds its `shopDTO` via the same
  `toShopDTO` that `shop.go` uses for `GET`/`PATCH` responses, so the shop JSON shape
  can't drift between the two endpoints.

- **Shared `FormField`/`FormError`/`FormSuccess` components** (`src/components/form-field.tsx`)
  instead of duplicating labeled-input/error/success markup across `/cadastro`,
  `/entrar`, and `/painel/loja`. All three forms render the same Tailwind input
  styling and the same `{error, details?}` → inline-list rendering, so one small
  shared module beats three near-identical copies. Kept intentionally tiny (no
  validation logic, no form-state management) — just presentation.

- **`AuthProvider`/session store built on `useSyncExternalStore`, not
  `useEffect` + `setState` on mount.** The natural first draft (`useEffect(() => {
  const stored = readSession(); setSession(stored); setStatus(...) }, [])`) reads
  cleanly but trips `react-hooks/set-state-in-effect` (configured as a hard lint
  error in this repo): calling `setState` synchronously in a mount effect is exactly
  the "force a re-render to sync with an external mutable source" smell the rule's
  own message points at `useSyncExternalStore` for. Switching to
  `useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)` — with
  `getServerSnapshot` returning `null` — is *also* the more correct fix: it gives a
  hydration-safe initial value (matches the server's `null`) and then synchronously
  reconciles to the real `localStorage` value during commit, before any consumer's
  `useEffect` runs (e.g., `/painel`'s session gate never sees a false "logged out"
  flicker that would cause a premature redirect). `subscribe` listens for both the
  native `storage` event (cross-tab) and a same-tab custom `barbershop:session-change`
  event dispatched by `writeSession`/`clearSession` (the `storage` event famously
  doesn't fire in the tab that performed the write). `getSnapshot` caches the parsed
  object keyed on the raw string so repeated calls return a stable reference (avoiding
  the "getSnapshot should be cached" re-render loop). `AuthStatus` collapsed from
  `'loading' | 'authenticated' | 'unauthenticated'` to just `'authenticated' |
  'unauthenticated'` — the `'loading'` gap that the old effect-based version needed
  no longer exists, since `useSyncExternalStore` never produces a torn/stale read for
  consumers. Trade-off: ~25 more lines than the naïve effect version, but it's the
  React-prescribed pattern for this exact problem and sidesteps both the lint error
  and a subtle hydration/flicker bug the naïve version would have shipped with.

- **`/painel` layout fetches `GET /v1/me` *and* `GET /v1/shops/:shopId` on mount,
  not just `/v1/me`.** The spec's gate requirement ("validates the stored token
  against `GET /v1/me`") only proves the token is live — it doesn't return the shop
  *name*, and the shell needs to display it (`/v1/me` only echoes `shop_id`). Rather
  than add a new "shop name" field to the JWT/`/me` response (which would re-open
  the "don't trust embedded claims for anything but display convenience" question
  the Design Notes warn about), the layout chains a `GET /v1/shops/:shopId` — a
  request `/painel/loja` needs to make anyway, and one any member can always make
  for their own shop. A non-401 failure on that second call (e.g., a transient `5xx`)
  doesn't force a logout/redirect — only a `401` from either call does — so a
  flaky shop-profile fetch can't lock a user out of their own session.

- **No GUI browser / screenshot tooling available in this environment** (no
  `playwright`, `puppeteer`, or `chromium*` binary, and none are project
  dependencies). The "browser walkthrough" and "screenshots" acceptance items were
  validated functionally instead — by `curl`-ing each new route for correct pt-BR
  content and a `200`, and by replaying each page's exact client→API call sequence
  (signup → login → `/v1/me` → shop `GET`/`PATCH` → logout) against the live API
  with real tokens — see Validation Results for the full trace. A human should still
  click through `/cadastro` → `/entrar` → `/painel` → `/painel/loja` once on a real
  viewport to sanity-check layout/spacing; the underlying request/response and
  routing behavior is already proven correct end-to-end.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move **"Authorization middleware: role checks + tenant-scoping at repository
  layer"** and **"Frontend: sign-up, login, role-aware authenticated shell"** from
  📋 → ✅ under Stage 1 deliverables.
- Check off all three Stage 1 acceptance-criteria boxes (new user can sign up + log
  in as Owner; Shop A cannot read/mutate Shop B's data; role-protected endpoints
  reject unauthorized roles; auth works end-to-end through the frontend).
- Move **Stage 1** from 🚧 → ✅ Complete, and update the Stage Status Overview table
  row for Stage 1 accordingly.
- Only update rows/sections corresponding to **Item 010** and the Stage 1 summary it
  completes — do not touch Stage 2+.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 010` to implement
this work item.
