# Item 008: Shop Sign-up & Owner Account Creation (API)

**Stage:** 1 — Identity, Tenancy & Authentication
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-06-05

---

## Description

Implement `POST /v1/signup` — the entry point for new barbershops. A single request
creates a **shop** (tenant), its first **owner user**, and the **membership** linking
them, all inside one database transaction. The endpoint validates inputs, hashes the
password with bcrypt, returns the created shop + owner identity (no password hash in
the response), and maps domain errors to appropriate HTTP status codes.

This item also fills in the `panic("not implemented")` stubs in all three
`pg_*` identity repository implementations and introduces a `Querier` interface so
repositories work both standalone (pool) and inside transactions (tx).

**API contract:**

```
POST /v1/signup
Content-Type: application/json

{
  "shop": {
    "name":     "Barbearia do João",      // required, 2–100 chars
    "phone":    "+5511999999999",          // optional
    "address":  "Rua das Flores, 123",    // optional
    "city":     "São Paulo",              // optional
    "state":    "SP"                      // optional, exactly 2 chars if present
  },
  "owner": {
    "email":     "joao@barbearia.com.br", // required, valid email
    "password":  "SecretPass123",         // required, min 8 chars
    "full_name": "João Silva",            // required, 2–100 chars
    "phone":     "+5511999999999"         // optional
  }
}

201 Created
{
  "shop": {
    "id":         "b4e2…",
    "name":       "Barbearia do João",
    "slug":       "barbearia-do-joao",
    "phone":      "+5511999999999",
    "address":    "Rua das Flores, 123",
    "city":       "São Paulo",
    "state":      "SP",
    "created_at": "2026-06-05T12:00:00Z"
  },
  "owner": {
    "id":         "c7f1…",
    "email":      "joao@barbearia.com.br",
    "full_name":  "João Silva",
    "phone":      "+5511999999999",
    "created_at": "2026-06-05T12:00:00Z"
  }
}

409 Conflict   — email already registered, or slug collision unresolvable
422 Unprocessable Entity — validation failure (missing required field, bad email, etc.)
500 Internal Server Error — unexpected DB or system error
```

---

## Acceptance Criteria

- [ ] `POST /v1/signup` with valid input returns `201` with shop + owner JSON (no
  `password_hash` in response).
- [ ] Shop, user, and membership rows are created in the same transaction — if any
  insert fails, none are persisted.
- [ ] `slug` is derived from `shop.name` (Unicode-normalised, lowercased, diacritics
  stripped, non-alphanumeric → hyphens). Example: `"Barbearia do João"` →
  `"barbearia-do-joao"`.
- [ ] On slug collision, up to 5 suffix variants are tried (`-2` … `-6`) before
  returning `409`.
- [ ] Duplicate email → `409 {"error": "email already registered"}`.
- [ ] Missing required field → `422 {"error": "validation failed", "details": [...]}`.
- [ ] `state` present but not exactly 2 chars → `422`.
- [ ] `password` present but shorter than 8 chars → `422`.
- [ ] Password is hashed with bcrypt (cost 12); the raw password never touches the DB.
- [ ] All three `pg_*` repository methods are implemented (no more `panic`).
- [ ] `go build ./...`, `go vet ./...`, `APP_ENV=test go test -short ./...` all pass.
- [ ] Integration tests covering happy path, duplicate email, and validation failures
  pass against a live database (`make test` or `-tags integration`).

---

## New Library Decision (confirm before implementing)

### Password hashing: `golang.org/x/crypto/bcrypt`

`golang.org/x/crypto` is already an **indirect** dependency in `go.mod` (pulled in by
`golang-migrate` and `pgx`). Promoting it to a direct dependency for `bcrypt` requires
no new download and is the de-facto Go standard for password hashing.

**Proposed:** `golang.org/x/crypto/bcrypt`, cost = 12.

> **Confirm:** Is bcrypt/cost-12 acceptable, or do you want Argon2id
> (`golang.org/x/crypto/argon2`) for stronger future-proofing? bcrypt is simpler and
> well-supported; Argon2id is more resistant to GPU attacks. Either way, no external
> library beyond `golang.org/x/crypto` is needed.
>
> **Default if not redirected: bcrypt cost 12.**

### Slug generation: in-house helper using `golang.org/x/text`

`golang.org/x/text` is already an indirect dependency. A small `slugify(string) string`
function in `internal/identity/slug.go` using `unicode/norm` (NFD decomposition) and
`transform` handles Brazilian Portuguese diacritics (ã, ç, ê, etc.) without a third-party
package.

**Proposed:** in-house helper. No new dependencies.

---

## Implementation Steps

### 1. Add `Querier` interface to `internal/database`

Create `internal/database/querier.go`:

```go
package database

import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx, enabling
// repositories to work both standalone and inside transactions.
type Querier interface {
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
```

### 2. Update `pg_*` repository constructors to accept `database.Querier`

Refactor all three repositories (`pg_shop_repo.go`, `pg_user_repo.go`,
`pg_membership_repo.go`) to store a `database.Querier` instead of `*pgxpool.Pool`.
Constructors become:

```go
func NewShopRepository(q database.Querier) ShopRepository
func NewUserRepository(q database.Querier) UserRepository
func NewMembershipRepository(q database.Querier) MembershipRepository
```

`*pgxpool.Pool` already satisfies `database.Querier`, so existing wiring in
`cmd/api/main.go` continues to compile without change.

### 3. Implement repository methods

#### `pg_shop_repo.go`

```go
func (r *pgShopRepo) Create(ctx context.Context, shop *Shop) error
// INSERT INTO shop (id, name, slug, phone, address, city, state)
// VALUES ($1,$2,$3,$4,$5,$6,$7)
// RETURNING created_at, updated_at
// Scan RETURNING columns back into *shop.
// On pgconn unique-constraint violation (pgerrcode.UniqueViolation):
//   if constraint == "shop_slug_unique" → return ErrSlugTaken

func (r *pgShopRepo) GetByID(ctx context.Context, id [16]byte) (*Shop, error)
// SELECT * FROM shop WHERE id = $1

func (r *pgShopRepo) GetBySlug(ctx context.Context, slug string) (*Shop, error)
// SELECT * FROM shop WHERE slug = $1
```

#### `pg_user_repo.go`

```go
func (r *pgUserRepo) Create(ctx context.Context, user *User) error
// INSERT INTO "user" (id, email, password_hash, full_name, phone)
// VALUES ($1,$2,$3,$4,$5)
// RETURNING created_at, updated_at
// On unique violation on "user_email_unique" → return ErrEmailTaken

func (r *pgUserRepo) GetByID(ctx context.Context, id [16]byte) (*User, error)
func (r *pgUserRepo) GetByEmail(ctx context.Context, email string) (*User, error)
```

#### `pg_membership_repo.go`

```go
func (r *pgMembershipRepo) Create(ctx context.Context, m *Membership) error
// INSERT INTO membership (id, shop_id, user_id, role)
// VALUES ($1,$2,$3,$4)
// RETURNING created_at

func (r *pgMembershipRepo) GetByShopAndUser(ctx context.Context, shopID, userID [16]byte) (*Membership, error)
func (r *pgMembershipRepo) ListByShop(ctx context.Context, shopID [16]byte) ([]Membership, error)
func (r *pgMembershipRepo) Delete(ctx context.Context, shopID, userID [16]byte) error
```

**UUID handling:** pgx v5 scans PostgreSQL `UUID` columns into Go `[16]byte` natively
via `pgtype.UUID`. Use `pgtype.UUID{Bytes: id, Valid: true}` when passing UUIDs as
query parameters, and scan into `pgtype.UUID` then extract `.Bytes` on read.

**Sentinel errors** (define in `internal/identity/errors.go`):
```go
var (
    ErrEmailTaken   = errors.New("email already registered")
    ErrSlugTaken    = errors.New("shop slug already taken")
    ErrNotFound     = errors.New("not found")
)
```

Map `pgerrcode.UniqueViolation` in each repo to the appropriate sentinel.

### 4. Create `internal/identity/slug.go`

```go
// slugify converts a display name to a URL-safe slug.
// "Barbearia do João" → "barbearia-do-joao"
func slugify(s string) string
```

Implementation outline:
1. NFD-decompose with `golang.org/x/text/unicode/norm`
2. Strip combining diacritics (Unicode category Mn)
3. Lowercase
4. Replace any run of non-`[a-z0-9]` characters with a single `-`
5. Trim leading/trailing hyphens

### 5. Create `internal/identity/service.go`

```go
type Service struct {
    pool    *pgxpool.Pool
    log     *zap.Logger
}

func NewService(pool *pgxpool.Pool, log *zap.Logger) *Service

// SignUp creates a shop, its first owner user, and their membership atomically.
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (*SignUpResponse, error)
```

`SignUp` implementation:

1. Hash password: `bcrypt.GenerateFromPassword([]byte(req.Owner.Password), 12)`
2. Generate UUID for shop and user (use `pgtype.UUID` with `gen_random_uuid()` on DB
   side, or generate with `crypto/rand` — **let DB generate via DEFAULT**; scan
   `RETURNING id` back into the struct).
3. `tx, err := s.pool.Begin(ctx)` — start transaction.
4. `defer tx.Rollback(ctx)` — no-op after commit.
5. Build `NewShopRepository(tx)`, `NewUserRepository(tx)`, `NewMembershipRepository(tx)`.
6. Try up to 6 slug variants (`slug`, `slug-2` … `slug-6`):
   - Call `shopRepo.Create(ctx, shop)`.
   - If `ErrSlugTaken`, increment suffix and retry.
   - If 6 attempts exhausted, return `ErrSlugTaken`.
7. Call `userRepo.Create(ctx, user)`.
   - If `ErrEmailTaken`, roll back and return `ErrEmailTaken`.
8. Call `memberRepo.Create(ctx, &Membership{ShopID: shop.ID, UserID: user.ID, Role: RoleOwner})`.
9. `tx.Commit(ctx)`.
10. Return `&SignUpResponse{Shop: shop, Owner: user}`.

**Request/response types** (also in `service.go` or `service_types.go`):

```go
type SignUpRequest struct {
    Shop  ShopInput  `json:"shop"`
    Owner OwnerInput `json:"owner"`
}

type ShopInput struct {
    Name     string `json:"name"     binding:"required,min=2,max=100"`
    Phone    string `json:"phone"`
    Address  string `json:"address"`
    City     string `json:"city"`
    State    string `json:"state"    binding:"omitempty,len=2"`
}

type OwnerInput struct {
    Email    string `json:"email"     binding:"required,email"`
    Password string `json:"password"  binding:"required,min=8"`
    FullName string `json:"full_name" binding:"required,min=2,max=100"`
    Phone    string `json:"phone"`
}

type SignUpResponse struct {
    Shop  *Shop `json:"shop"`
    Owner *User `json:"owner"`
}
```

### 6. Create `internal/identity/routes.go` — route registration + handlers

Use the **domain-registers-its-own-routes** pattern to avoid importing the identity
package from `internal/http`:

```go
package identity

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// RegisterRoutes wires identity endpoints onto the provided router group.
// Call from cmd/api/main.go: identity.RegisterRoutes(router.Group("/v1"), svc, log)
func RegisterRoutes(rg *gin.RouterGroup, svc *Service, log *zap.Logger)
```

Handler `handleSignUp`:

```go
func handleSignUp(svc *Service, log *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req SignUpRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            // Format validator.ValidationErrors into {"error":"validation failed","details":[...]}
            c.JSON(http.StatusUnprocessableEntity, formatValidationError(err))
            return
        }
        resp, err := svc.SignUp(c.Request.Context(), req)
        if err != nil {
            switch {
            case errors.Is(err, ErrEmailTaken):
                c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
            case errors.Is(err, ErrSlugTaken):
                c.JSON(http.StatusConflict, gin.H{"error": "shop name too similar to an existing shop; try a different name"})
            default:
                log.Error("signup failed", zap.Error(err))
                c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
            }
            return
        }

        c.JSON(http.StatusCreated, toSignUpDTO(resp))
    }
}
```

**Response DTOs** (in `routes.go` or `dto.go`) — converts `[16]byte` UUIDs to strings
and omits `PasswordHash`:

```go
type shopDTO struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Slug      string `json:"slug"`
    Phone     string `json:"phone,omitempty"`
    Address   string `json:"address,omitempty"`
    City      string `json:"city,omitempty"`
    State     string `json:"state,omitempty"`
    CreatedAt string `json:"created_at"`
}

type ownerDTO struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    FullName  string `json:"full_name"`
    Phone     string `json:"phone,omitempty"`
    CreatedAt string `json:"created_at"`
}

type signUpDTO struct {
    Shop  shopDTO  `json:"shop"`
    Owner ownerDTO `json:"owner"`
}
```

UUID formatting helper (unexported, in `routes.go` or `dto.go`):
```go
func fmtUUID(id [16]byte) string {
    return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
        id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
}
```

`CreatedAt` formatted as RFC3339 UTC.

### 7. Update `cmd/api/main.go`

Wire the identity service and register routes:

```go
import "github.com/gcollin65/barbershop/internal/identity"

// After pool + migrations setup:
identitySvc := identity.NewService(pool, logger)

router := apihttp.NewRouter(logger, pool)
identity.RegisterRoutes(router.Group("/v1"), identitySvc, logger)
```

No changes to `NewRouter` signature needed — identity registers itself on the group.

### 8. Promote `golang.org/x/crypto` to direct dependency

```bash
cd backend
go get golang.org/x/crypto
go get golang.org/x/text
go mod tidy
```

Both are already in `go.sum`; this just moves them from `// indirect` to direct
in `go.mod`.

### 9. Tests

#### Handler tests (`internal/identity/routes_test.go`)

Use `httptest` + mock implementations of `ShopRepository`, `UserRepository`,
`MembershipRepository`. Test cases:

| Scenario | Input | Expected |
|----------|-------|----------|
| Happy path | Valid shop + owner | `201`, shop + owner in body, no password_hash |
| Missing shop name | `shop.name` omitted | `422`, details mention `name` |
| Invalid email | `owner.email = "not-email"` | `422` |
| Short password | `owner.password = "short"` | `422` |
| Duplicate email | service returns `ErrEmailTaken` | `409 {"error":"email already registered"}` |
| Slug exhausted | service returns `ErrSlugTaken` | `409` |

For handler tests, mock the service by extracting a `Signer` interface:
```go
type Signer interface {
    SignUp(ctx context.Context, req SignUpRequest) (*SignUpResponse, error)
}
```

Handler accepts `Signer` instead of `*Service` — makes the concrete type easy to mock.

#### Service unit tests (`internal/identity/service_test.go`)

Use mock repositories. Test:
- Slug is derived correctly from name.
- On slug collision, suffix is appended and retry succeeds.
- On user duplicate, `ErrEmailTaken` is returned.
- On success, all three repos are called in order.

#### Repository integration tests (`internal/identity/pg_signup_test.go`)

Build tag `integration`. Require `DATABASE_URL`. Test:
- `ShopRepository.Create` persists a row; `GetByID` retrieves it.
- `UserRepository.Create` persists a row; `GetByEmail` retrieves it.
- Duplicate email on `UserRepository.Create` returns `ErrEmailTaken`.
- Full `Service.SignUp` end-to-end: all three rows exist after commit.
- `Service.SignUp` with duplicate email leaves the DB unchanged (transaction rolled back).

#### Slug unit tests (`internal/identity/slug_test.go`)

```
"Barbearia do João" → "barbearia-do-joao"
"São Paulo"         → "sao-paulo"
"  Hello World  "   → "hello-world"
"A & B + C"         → "a-b-c"
""                  → ""
```

---

## Testing Strategy

| Layer | Tool | When |
|-------|------|------|
| Slug helper | `go test` (stdlib) | Always (no deps) |
| Handler (unit) | `httptest` + mock Signer | Always |
| Service (unit) | mock repos | Always |
| Repository (integration) | `go test -tags integration` | Requires live Postgres |
| End-to-end signup | integration test via `Service.SignUp` | Requires live Postgres |
| Manual curl | `curl -s -X POST localhost:8080/v1/signup` | Before marking complete |

---

## Dependencies

- **Upstream:** Item 007 ✅ (`shop`, `user`, `membership` tables exist; repo stubs exist).
- **Downstream (enables):**
  - Item 009 (Authentication & Login) — reads `user` rows by email to verify password;
    depends on `UserRepository.GetByEmail` and the bcrypt hash produced here.
  - Item 010 (Authorization & Frontend auth) — depends on the signup flow being complete.
- **New direct Go dependencies:**
  - `golang.org/x/crypto` (bcrypt) — already indirect; promote to direct.
  - `golang.org/x/text` (slug normalization) — already indirect; promote to direct.
- **No new external services.** PostgreSQL is the only runtime dependency.

---

## Testing Prerequisites

### Required Services

| Service | Version | Start Command | Port |
|---------|---------|---------------|------|
| PostgreSQL | 16 | `cd backend && make db-start` | 5432 |

Or for the full stack: `cd backend && make dev` (includes API + frontend, but tests
target the DB directly).

### Environment Configuration

| Variable | Required | Example |
|----------|----------|---------|
| `DATABASE_URL` | Yes (integration) | `postgres://barber:secret@localhost:5432/barbershop?sslmode=disable` |
| `PORT` | No | `8080` |

Copy `backend/.env.example` to `backend/.env`; `make db-start` reads it automatically.

### Manual Validation Checklist

- [ ] **Build:** `cd backend && go build ./...`
- [ ] **Vet:** `cd backend && go vet ./...`
- [ ] **Short tests:** `cd backend && APP_ENV=test go test -short ./...`
- [ ] **Start postgres:** `cd backend && make db-start`
- [ ] **Integration tests:** `cd backend && go test -tags integration ./internal/identity/...`
- [ ] **Start API:** `cd backend && make dev-local` (or `make dev`)
- [ ] **Happy path:**
  ```bash
  curl -s -X POST http://localhost:8080/v1/signup \
    -H 'Content-Type: application/json' \
    -d '{
      "shop":  {"name":"Barbearia do João","city":"São Paulo","state":"SP"},
      "owner": {"email":"joao@test.com","password":"Secret123","full_name":"João Silva"}
    }' | jq .
  ```
  Expected: `201` — JSON with `shop.id`, `shop.slug = "barbearia-do-joao"`,
  `owner.id`, no `password_hash`.
- [ ] **Duplicate email:**
  Re-run the same curl → `409 {"error":"email already registered"}`.
- [ ] **Validation failure:**
  ```bash
  curl -s -X POST http://localhost:8080/v1/signup \
    -H 'Content-Type: application/json' \
    -d '{"shop":{"name":"X"},"owner":{"email":"bad","password":"123"}}' | jq .
  ```
  Expected: `422` with `details` listing field errors.
- [ ] **DB verification:**
  ```sql
  SELECT s.name, s.slug, u.email, m.role
  FROM shop s
  JOIN membership m ON m.shop_id = s.id
  JOIN "user"  u ON u.id = m.user_id
  WHERE u.email = 'joao@test.com';
  ```
  Expected: one row with role `owner`.
- [ ] **Password not in DB:**
  ```sql
  SELECT password_hash FROM "user" WHERE email = 'joao@test.com';
  ```
  Expected: a bcrypt hash (`$2a$12$…`), never the plain text.

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `POST /v1/signup` (valid) | `201` — shop + owner JSON, `slug` derived from name |
| `POST /v1/signup` (dup email) | `409 {"error":"email already registered"}` |
| `POST /v1/signup` (bad input) | `422 {"error":"validation failed","details":[…]}` |
| DB: shop row | exists with correct slug |
| DB: user row | exists; `password_hash` starts with `$2a$12$` |
| DB: membership row | exists with `role = owner` |
| Integration tests | all pass against live Postgres |

### Validation Results

```markdown
## Validation Results
- [ ] Service started: PostgreSQL 16 (Docker) — requires Docker Desktop + WSL integration
- [x] Application started successfully — go build ./... passes
- [x] Unit tests pass: 14/14 (slug, service, handler)
- [ ] Integration tests: require live Postgres (run: go test -tags integration ./internal/identity/...)
- [ ] API endpoints verified: requires running server + Postgres
- [ ] Screenshots captured: N/A (no UI changes)
```

---

## Decisions & Trade-offs

**Confirmed before implementation (resolve during create-item):**

- **Password hashing = bcrypt, cost 12** — `golang.org/x/crypto/bcrypt` already an
  indirect dep; promote to direct. Cost 12 is OWASP-recommended minimum for 2024.
  Trade-off vs Argon2id: bcrypt is widely understood; Argon2id is more resistant to
  GPU attacks. Defaulting to bcrypt unless user redirects.

- **Slug helper = in-house (no new dep)** — uses `golang.org/x/text/unicode/norm`
  (already indirect) to NFD-decompose and strip diacritics. Handles Brazilian
  Portuguese correctly. Trade-off: more code to maintain vs a library, but the
  function is small and well-tested.

- **Route registration pattern = domain-registers-own-routes** — `identity.RegisterRoutes`
  is called from `main.go`, keeping `internal/http` free of identity imports. Each
  domain module wires its own handlers. This scales cleanly to Items 009, 010, and
  beyond without a monolithic `NewRouter` accumulating all dependencies.

- **`database.Querier` interface** — both `*pgxpool.Pool` and `pgx.Tx` satisfy it;
  repositories accept `Querier`. This enables transactional composition without a
  Unit-of-Work framework. Trade-off: a thin abstraction layer; worth it because the
  signup transaction is the first of many that will need this pattern.

- **Slug collision strategy = suffix `-2` … `-6` (max 6 attempts)** — simple and
  predictable. After 6 collisions the user gets a `409`; they can try a more specific
  shop name. Trade-off: rare edge case vs complexity of random suffix.

- **UUIDs in HTTP responses = formatted `[16]byte` strings** — a small `fmtUUID`
  helper formats `[16]byte` as a proper UUID string (`xxxxxxxx-xxxx-…`). No new
  UUID library needed. Domain models keep `[16]byte` for pgx compatibility.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move **"Shop sign-up flow (create shop + first Owner account)"** from 📋 → ✅.
- Stage 1 remains 🚧 (Items 009, 010 still pending).
- Only update rows corresponding to **Item 008**.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 008` to implement
this work item.
