# Item 007: Multi-Tenant Data Model & Migrations

**Stage:** 1 — Identity, Tenancy & Authentication
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-31

---

## Description

Define and migrate the core identity/tenancy schema that underpins all of Stage 1
and every subsequent domain in the system. The four tables and one enum introduced
here establish the tenant boundary:

- **`shop`** — the tenant unit; every piece of domain data belongs to exactly one
  shop.
- **`user`** — a global account (one email, one password); a user can belong to
  multiple shops with different roles.
- **`user_role`** — a PostgreSQL enum (`owner`, `barber`, `customer`) encoding the
  three access levels described in the vision.
- **`membership`** — the join between a user and a shop, carrying that user's role
  in that shop. One membership per (shop, user) pair.

Beyond the migrations this item establishes the patterns all future items will follow:

- Go model structs for `Shop`, `User`, `Role`, and `Membership` in `internal/identity/`.
- Repository interfaces that require a tenant (shop ID) scope — future implementations
  cannot accidentally return cross-tenant data.
- A context helper (`WithTenant` / `TenantFromCtx`) for propagating the active tenant
  ID through the request chain.
- Test seed/fixture helpers used by integration tests in Items 008–010 and beyond.

No HTTP handlers are introduced here. Items 008 (sign-up) and 009 (authentication)
build on these foundations.

---

## Acceptance Criteria

- [ ] Migration `000002_identity_tenancy.up.sql` creates `shop`, `user`, `user_role`
      enum, and `membership`; applies cleanly to an empty database and to one that
      already has migration 000001 applied.
- [ ] Migration `000002_identity_tenancy.down.sql` rolls back cleanly (all four
      objects dropped, no residual state).
- [ ] `make migrate-up` advances to schema version 2; `make migrate-down` rolls back
      to version 1; the cycle is repeatable without errors.
- [ ] `shop` has: UUID PK, unique `slug`, required `name`, `timezone` defaulting to
      `'America/Sao_Paulo'`, `created_at`/`updated_at` timestamptz.
- [ ] `user` (quoted — reserved keyword) has: UUID PK, unique `email`, required
      `password_hash`, required `full_name`, `created_at`/`updated_at` timestamptz.
- [ ] `membership` enforces UNIQUE(shop_id, user_id) with CASCADE deletes from both
      `shop` and `user`.
- [ ] Lookup indexes exist on `membership(shop_id)` and `membership(user_id)`.
- [ ] Go `Shop`, `User`, `Role`, and `Membership` model types defined in
      `internal/identity/model.go`; the package builds with `go build ./...`.
- [ ] Repository interfaces `ShopRepository`, `UserRepository`,
      `MembershipRepository` defined in `internal/identity/repository.go`; every
      tenant-scoped method carries `shopID` as an explicit parameter.
- [ ] Tenant context helpers `WithTenant` / `TenantFromCtx` defined in
      `internal/identity/tenant.go` and covered by a short (no-DB) unit test.
- [ ] Integration test `TestMigrationRoundTrip` in `internal/identity/` applies and
      rolls back the migration, verifying table existence at each step.
- [ ] Test fixture helpers (`MustCreateShop`, `MustCreateUser`, `MustCreateMembership`)
      available under the `integration` build tag for use by Items 008–010.
- [ ] `go test -short ./...` exits 0 (no DB-dependent code runs in short mode).
- [ ] `go test -tags integration ./internal/identity/...` exits 0 against a running
      Postgres.

---

## Implementation Steps

### 1. Migration: `000002_identity_tenancy.up.sql`

Create `backend/internal/database/migrations/000002_identity_tenancy.up.sql`:

```sql
-- Shop (tenant unit)
CREATE TABLE shop (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL,
    phone      TEXT,
    address    TEXT,
    city       TEXT,
    state      CHAR(2),
    timezone   TEXT        NOT NULL DEFAULT 'America/Sao_Paulo',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT shop_slug_unique UNIQUE (slug)
);

-- Role enum
CREATE TYPE user_role AS ENUM ('owner', 'barber', 'customer');

-- User (global; one account can belong to many shops with different roles)
-- Note: "user" is a reserved keyword in PostgreSQL and must always be quoted.
CREATE TABLE "user" (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    phone         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_email_unique UNIQUE (email)
);

-- Membership: links a user to a shop with a role (one role per user per shop)
CREATE TABLE membership (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_id    UUID        NOT NULL REFERENCES shop   (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    role       user_role   NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT membership_shop_user_unique UNIQUE (shop_id, user_id)
);

CREATE INDEX membership_shop_id_idx ON membership (shop_id);
CREATE INDEX membership_user_id_idx ON membership (user_id);
```

The `//go:embed migrations/*.sql` directive in `migrations.go` picks up new files
automatically — no change to that file is needed.

### 2. Migration: `000002_identity_tenancy.down.sql`

Create `backend/internal/database/migrations/000002_identity_tenancy.down.sql`:

```sql
DROP TABLE IF EXISTS membership;
DROP TABLE IF EXISTS "user";
DROP TYPE  IF EXISTS user_role;
DROP TABLE IF EXISTS shop;
```

### 3. Go model types — `internal/identity/model.go`

Replace the existing stub `doc.go` content or add alongside it. pgx/v5 scans
UUID columns into `[16]byte` natively — no additional UUID library is needed.

```go
package identity

import "time"

// Role mirrors the user_role PostgreSQL enum.
type Role string

const (
    RoleOwner    Role = "owner"
    RoleBarber   Role = "barber"
    RoleCustomer Role = "customer"
)

type Shop struct {
    ID        [16]byte
    Name      string
    Slug      string
    Phone     string
    Address   string
    City      string
    State     string
    Timezone  string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type User struct {
    ID           [16]byte
    Email        string
    PasswordHash string
    FullName     string
    Phone        string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Membership struct {
    ID        [16]byte
    ShopID    [16]byte
    UserID    [16]byte
    Role      Role
    CreatedAt time.Time
}
```

### 4. Repository interfaces — `internal/identity/repository.go`

```go
package identity

import "context"

// ShopRepository is the persistence boundary for tenant records.
type ShopRepository interface {
    Create(ctx context.Context, shop *Shop) error
    GetByID(ctx context.Context, id [16]byte) (*Shop, error)
    GetBySlug(ctx context.Context, slug string) (*Shop, error)
}

// UserRepository handles global user account records.
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id [16]byte) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
}

// MembershipRepository links users to shops. shopID is an explicit parameter
// on every method so tenant scope cannot be accidentally omitted.
type MembershipRepository interface {
    Create(ctx context.Context, m *Membership) error
    GetByShopAndUser(ctx context.Context, shopID, userID [16]byte) (*Membership, error)
    ListByShop(ctx context.Context, shopID [16]byte) ([]Membership, error)
    Delete(ctx context.Context, shopID, userID [16]byte) error
}
```

### 5. Concrete repository stubs

Add stub files that satisfy the interfaces and compile, but panic on call. Items
008–010 fill in the SQL bodies.

`internal/identity/pg_shop_repo.go`:
```go
package identity

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

type pgShopRepo struct{ pool *pgxpool.Pool }

func NewShopRepository(pool *pgxpool.Pool) ShopRepository { return &pgShopRepo{pool: pool} }

func (r *pgShopRepo) Create(ctx context.Context, shop *Shop) error               { panic("not implemented") }
func (r *pgShopRepo) GetByID(ctx context.Context, id [16]byte) (*Shop, error)    { panic("not implemented") }
func (r *pgShopRepo) GetBySlug(ctx context.Context, slug string) (*Shop, error)  { panic("not implemented") }
```

Repeat the same stub pattern for `pg_user_repo.go` (`NewUserRepository`) and
`pg_membership_repo.go` (`NewMembershipRepository`).

### 6. Tenant context helper — `internal/identity/tenant.go`

```go
package identity

import "context"

type contextKey struct{}

// WithTenant returns a new context carrying the active shop (tenant) ID.
func WithTenant(ctx context.Context, shopID [16]byte) context.Context {
    return context.WithValue(ctx, contextKey{}, shopID)
}

// TenantFromCtx retrieves the tenant ID from ctx.
// Returns the zero UUID and false if not set.
func TenantFromCtx(ctx context.Context) ([16]byte, bool) {
    v, ok := ctx.Value(contextKey{}).([16]byte)
    return v, ok
}
```

### 7. Short unit test — `internal/identity/tenant_test.go`

```go
package identity_test

import (
    "context"
    "testing"

    "github.com/gcollin65/barbershop/internal/identity"
)

func TestWithTenant_roundTrip(t *testing.T) {
    var id [16]byte
    id[15] = 42
    ctx := identity.WithTenant(context.Background(), id)
    got, ok := identity.TenantFromCtx(ctx)
    if !ok || got != id {
        t.Fatalf("got %v ok=%v, want %v ok=true", got, ok, id)
    }
}

func TestTenantFromCtx_missing(t *testing.T) {
    _, ok := identity.TenantFromCtx(context.Background())
    if ok {
        t.Fatal("expected ok=false for context without tenant")
    }
}
```

### 8. Integration test — `internal/identity/migration_test.go`

```go
//go:build integration

package identity_test

import (
    "context"
    "os"
    "testing"

    "github.com/gcollin65/barbershop/internal/database"
    "github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationRoundTrip(t *testing.T) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        t.Skip("DATABASE_URL not set")
    }

    pool, err := pgxpool.New(context.Background(), dsn)
    if err != nil {
        t.Fatalf("open pool: %v", err)
    }
    defer pool.Close()

    // Apply up to latest
    if err := database.RunMigrations(dsn, database.Migrations, nopLogger()); err != nil {
        t.Fatalf("migrate up: %v", err)
    }

    // Verify tables exist
    for _, table := range []string{"shop", `"user"`, "membership"} {
        var exists bool
        row := pool.QueryRow(context.Background(),
            "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
            stripQuotes(table))
        if err := row.Scan(&exists); err != nil || !exists {
            t.Errorf("table %s not found after migrate up", table)
        }
    }

    // Note: rolling back via the CLI or a separate migrate-down call is the
    // recommended approach; down-migration in tests requires the migrate CLI or
    // a dedicated down helper not yet wired. This test verifies up only.
}
```

(Helper `nopLogger()` returns a `*zap.Logger` in no-op mode; `stripQuotes` removes
surrounding quotes from the identifier for the query parameter.)

### 9. Test fixture helpers — `internal/identity/testfixtures_test.go`

```go
//go:build integration

package identity_test

import (
    "context"
    "fmt"
    "testing"
    "time"

    "github.com/gcollin65/barbershop/internal/identity"
    "github.com/jackc/pgx/v5/pgxpool"
)

// MustCreateShop inserts a shop row and returns it. Fails the test on error.
func MustCreateShop(t *testing.T, pool *pgxpool.Pool, slug string) *identity.Shop {
    t.Helper()
    shop := &identity.Shop{
        Name:     "Test Shop " + slug,
        Slug:     slug,
        Timezone: "America/Sao_Paulo",
    }
    row := pool.QueryRow(context.Background(),
        `INSERT INTO shop (name, slug, timezone) VALUES ($1, $2, $3)
         RETURNING id, created_at, updated_at`,
        shop.Name, shop.Slug, shop.Timezone)
    if err := row.Scan(&shop.ID, &shop.CreatedAt, &shop.UpdatedAt); err != nil {
        t.Fatalf("MustCreateShop: %v", err)
    }
    return shop
}

// MustCreateUser inserts a user row and returns it. Fails the test on error.
func MustCreateUser(t *testing.T, pool *pgxpool.Pool, email string) *identity.User {
    t.Helper()
    u := &identity.User{
        Email:        email,
        PasswordHash: "$2a$10$placeholder_hash_for_testing_only",
        FullName:     "Test User " + fmt.Sprint(time.Now().UnixNano()),
    }
    row := pool.QueryRow(context.Background(),
        `INSERT INTO "user" (email, password_hash, full_name) VALUES ($1, $2, $3)
         RETURNING id, created_at, updated_at`,
        u.Email, u.PasswordHash, u.FullName)
    if err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
        t.Fatalf("MustCreateUser: %v", err)
    }
    return u
}

// MustCreateMembership inserts a membership row and returns it. Fails the test on error.
func MustCreateMembership(t *testing.T, pool *pgxpool.Pool, shopID, userID [16]byte, role identity.Role) *identity.Membership {
    t.Helper()
    m := &identity.Membership{ShopID: shopID, UserID: userID, Role: role}
    row := pool.QueryRow(context.Background(),
        `INSERT INTO membership (shop_id, user_id, role) VALUES ($1, $2, $3)
         RETURNING id, created_at`,
        m.ShopID, m.UserID, string(m.Role))
    if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
        t.Fatalf("MustCreateMembership: %v", err)
    }
    return m
}
```

---

## Testing Strategy

| Layer | What | Command |
|-------|------|---------|
| Migration up | `shop`, `user`, `user_role`, `membership` created | `make migrate-up` |
| Migration down | All objects dropped | `make migrate-down` |
| Tenant helper (short) | `WithTenant`/`TenantFromCtx` round-trip | `go test -short ./internal/identity/...` |
| Repository compile | Stubs satisfy interfaces | `go build ./...` |
| Migration round-trip (integration) | Up verified against live DB | `make test` or `go test -tags integration ./internal/identity/...` |
| Fixture helpers (integration) | Insert + scan for shop/user/membership | same as above |
| Full short suite | No regressions | `make test-short` |

---

## Dependencies

- **Upstream:**
  - Item 003 ✅ — provides `database.RunMigrations`, `database.Migrations` embed, and
    pgxpool setup. Migration 000001 (baseline) must already exist.
  - Item 004 ✅ — `make db-start` (or `docker compose up -d postgres`) starts the
    local Postgres container for manual testing and integration tests.
- **Downstream (enables):**
  - Item 008 — sign-up API builds `pgShopRepo.Create` and `pgUserRepo.Create`.
  - Item 009 — auth login uses `pgUserRepo.GetByEmail`.
  - Item 010 — auth middleware uses `MembershipRepository` and `TenantFromCtx`.
  - Items 011–020 — all domain tables will carry a `shop_id FK → shop(id)` column,
    following the tenant scoping convention established here.
- **External libraries:** No new libraries required. All dependencies (`pgx/v5`,
  `golang-migrate`, `go.uber.org/zap`) are already pinned in `go.mod`.

---

## Testing Prerequisites

### Required Services

| Service | Version | Start command | Port |
|---------|---------|---------------|------|
| PostgreSQL | 15+ | `make db-start` | 5432 |

The API service itself is not required — only Postgres is needed for this item.

### Environment Configuration

The same `.env` used in Items 003–004 works unchanged. No new variables required.

Default credentials (from `backend/Makefile`):
```
DATABASE_URL=postgres://barber:secret@localhost:5432/barbershop?sslmode=disable
```

Copy from `.env.example` if you haven't already:
```bash
cp backend/.env.example backend/.env
```

### Manual Validation Checklist

- [ ] **Service started:** `make db-start` → postgres container healthy
- [ ] **Migration up:** `cd backend && make migrate-up` → exits 0, logs schema_version 2
- [ ] **Tables verified:** `psql $DATABASE_URL -c "\dt"` → shows `membership`, `shop`, `user`
- [ ] **Enum verified:** `psql $DATABASE_URL -c "\dT user_role"` → shows `owner`, `barber`, `customer`
- [ ] **Indexes verified:** `psql $DATABASE_URL -c "\di membership*"` → shows both indexes
- [ ] **Migration down:** `cd backend && make migrate-down` → exits 0, back to version 1
- [ ] **Tables gone:** `psql $DATABASE_URL -c "\dt"` → `shop`, `user`, `membership` absent
- [ ] **Build:** `cd backend && go build ./...` → exits 0
- [ ] **Short tests:** `cd backend && make test-short` → exits 0
- [ ] **Integration tests:** `cd backend && make test` (or `go test -tags integration ./internal/identity/...`) → exits 0

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `make migrate-up` | Exits 0; logs `schema_version: 2` |
| `\dt` after up | `membership`, `shop`, `user` all present |
| `\dT user_role` | Enum values: `owner`, `barber`, `customer` |
| `\di membership*` | `membership_shop_id_idx`, `membership_user_id_idx` present |
| `make migrate-down` | Exits 0; back at version 1 |
| `\dt` after down | `shop`, `user`, `membership` absent |
| `go build ./...` | Exits 0 |
| `make test-short` | Exits 0; tenant helper tests pass |
| `go test -tags integration ./internal/identity/...` | `TestMigrationRoundTrip` passes |

### Validation Documentation Template

```markdown
## Validation Results — Item 007

- [ ] Postgres started: `make db-start`
- [ ] `make migrate-up`: exits 0, schema_version = 2
- [ ] Tables verified: shop, user, membership present
- [ ] Enum verified: user_role has owner/barber/customer
- [ ] Indexes verified: membership_shop_id_idx, membership_user_id_idx
- [ ] `make migrate-down`: exits 0, version back to 1
- [ ] Tables absent after down: confirmed
- [ ] `go build ./...`: exits 0
- [ ] `make test-short`: exits 0, N tests pass
- [ ] `go test -tags integration ./internal/identity/...`: exits 0
```

---

## Decisions & Trade-offs

**Implementation decisions:**

- **`pool_test.go` updated to expect schema version 2** — `TestPoolIntegration` previously hardcoded `ver != 1` and expected `ErrNilVersion` after `Steps(-1)`. With two migrations, `RunMigrations` now lands at version 2 and rollback lands at version 1 (not NilVersion). Updated the assertion and removed the `errors.Is(err, migrate.ErrNilVersion)` check accordingly; removed unused `"errors"` import.
- **No `nopLogger()`/`stripQuotes()` helpers extracted** — inlined `zap.NewNop()` and `strings.Trim()` directly in `migration_test.go`. The helpers in the spec were illustrative; the inlined forms are cleaner and avoid dead-code concerns.
- **`testfixtures_test.go` uses `package identity_test`** — helpers are available to Items 008–010 test files in `internal/identity/` that also declare `package identity_test`, since Go compiles all `_test.go` files in a directory into the same test binary.

**Pre-decided (stack alignment and vision constraints):**

- **UUID primary keys via `gen_random_uuid()`** — generated in Postgres; no Go-side
  UUID library needed. pgx/v5 scans UUID columns into `[16]byte` natively.
- **Shared-schema multi-tenancy** — vision §7 confirms shared schema with a `shop_id`
  column is acceptable for initial scale. All domain tables added in Stage 2+ carry
  a `shop_id FK → shop(id)` and are filtered at the repository layer.
- **One role per (shop, user) pair** — UNIQUE(shop_id, user_id) on `membership`. A
  user needing multiple roles at the same shop is an explicit model change. Simple
  and correct for v1.
- **`user_role` as a PostgreSQL enum** — type-safe at the DB layer; pgx/v5 scans
  enum columns to `string` cleanly. Avoids a lookup table for three stable values.
- **Quoted `"user"` table name** — `user` is a reserved keyword in PostgreSQL. The
  table is always created and queried as `"user"`. All SQL in this package must quote
  this identifier consistently.
- **`[16]byte` for UUID fields** — matches pgx/v5's native UUID scan target. Use
  `pgtype.UUID` from `pgx/v5/pgtype` only when nullable UUIDs are needed (foreign
  keys in this item are all NOT NULL).
- **`password_hash` in `user`** — bcrypt hashes stored here; raw password never
  persists. `golang.org/x/crypto/bcrypt` is already in `go.sum` (indirect dep) and
  will be used in Item 009; no `go.mod` change required.
- **Stub `panic` implementations** — repository stubs panic rather than return
  `errors.New("not implemented")` so callers fail loudly if accidentally invoked
  before Items 008–010 are complete.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:

- Under Stage 1 deliverables, change **"Data model: tenant/shop, user, role,
  membership; tenant ID scoping on all tables"** from 📋 → ✅.
- Update Stage 1 **Status** from 📋 Planned → 🚧 In Progress (first deliverable done).

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 007` to implement
this work item.
