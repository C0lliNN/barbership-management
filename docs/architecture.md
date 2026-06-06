# Backend Architecture

**Status:** Current  
**Last updated:** 2026-06-06

---

## Overview

The backend follows **Hexagonal Architecture** (Ports & Adapters). The central idea is that domain logic — entities, business rules, repository interfaces — has zero knowledge of infrastructure. Infrastructure (PostgreSQL, HTTP frameworks, etc.) depends on the domain, never the other way around.

```
┌─────────────────────────────────────────────────────────┐
│                        Domain                           │
│   entities · repository interfaces · service · errors  │
│      internal/identity/   internal/booking/   ...       │
└────────────────┬────────────────────────────────────────┘
                 │  depends on (implements interfaces)
┌────────────────▼────────────────────────────────────────┐
│                     Infrastructure                       │
│         internal/infra/repository/   internal/http/     │
│         internal/database/           cmd/api/           │
└─────────────────────────────────────────────────────────┘
```

---

## Package Layout

```
backend/
├── cmd/api/                    Entrypoint — wires domain + infra, starts server
├── internal/
│   ├── identity/               Domain: identity & tenancy
│   ├── booking/                Domain: bookings & scheduling  (planned)
│   ├── catalog/                Domain: services & pricing     (planned)
│   ├── subscription/           Domain: recurring plans        (planned)
│   ├── payment/                Domain: payment orchestration  (planned)
│   ├── scheduling/             Domain: availability engine    (planned)
│   ├── infra/
│   │   ├── http/               Gin router, middleware, HTTP handlers for all domains
│   │   └── repository/         PostgreSQL adapters for all domains
│   ├── database/               Pool, migrations, Querier interface
│   ├── config/                 Environment configuration
│   └── logger/                 Zap logger setup + context helpers
```

---

## The Two Layers in Detail

### Domain packages (`internal/<domain>/`)

A domain package owns:

| File | Contents |
|------|----------|
| `entity.go` | Entities and value types (`Shop`, `User`, `Membership`, `Role`) — IDs are `string` (UUID format), timestamps are `int64` (Unix seconds) |
| `repository.go` | Repository interfaces (`ShopRepository`, `UserRepository`, …) |
| `errors.go` | Sentinel errors (`ErrNotFound`, `ErrEmailTaken`, `ErrSlugTaken`) |
| `service.go` | Use-case logic, request/response types |
| `slug.go` | Pure domain logic helpers |

**What domain packages must NOT import:**
- `github.com/jackc/pgx/...`
- `github.com/jackc/pgerrcode`
- `github.com/jackc/pgx/v5/pgxpool`
- Any other infrastructure library

The only external runtime dependencies allowed in domain packages are:
- `go.uber.org/zap` (logging)
- `golang.org/x/crypto/bcrypt` (hashing)

### Logging

All application logging uses a per-request `*zap.Logger` stored in `context.Context`. The `Logger` middleware (registered on every route group) enriches the base logger with `request_id` and `tenant` fields before the handler executes.

```
request
  └─ Logger middleware
       ├─ reads request_id from gin context (set by requestID middleware)
       ├─ reads X-Tenant header
       └─ base.With(request_id, tenant) → stored via logger.WithLogger(ctx, log)

handler / service / repo
  └─ logger.FromContext(ctx).Info("...")   ← enriched logger, no extra fields needed
```

Every layer that wants to log **must** call `logger.FromContext(ctx)` — never hold a `*zap.Logger` as a struct field or receive one as a parameter. If no logger is in context (e.g., background jobs, tests), `FromContext` returns `zap.NewNop()`.

```go
// internal/logger/logger.go
func WithLogger(ctx context.Context, log *zap.Logger) context.Context
func FromContext(ctx context.Context) *zap.Logger
```

The sole exception is `recoverer`, which holds the base logger as a closed-over fallback for the unlikely case where the Logger middleware panics before enriching the context.

### Error wrapping

Errors crossing a layer boundary must always be wrapped with `fmt.Errorf("context: %w", err)`. This preserves the full error chain for logging while keeping `errors.Is` / `errors.As` working.

| Origin | Caller | Rule |
|--------|--------|------|
| pgx / stdlib | Repository method | Wrap: `fmt.Errorf("shop create: %w", err)` |
| Repository interface | Service method | Wrap: `fmt.Errorf("create shop: %w", err)` |
| Service interface | HTTP handler | Handler writes HTTP response — no re-wrapping needed |

**Exception:** sentinel error translations (e.g., a `UniqueViolation` mapped to `identity.ErrSlugTaken`) are not wrapped — they intentionally replace the lower-level error with a domain-level sentinel. The service then re-wraps only non-sentinel errors before propagating them.

### Pointer rules

Pointers are avoided in method parameters and return types to prevent nil-pointer bugs and reduce verbosity. The sole exception is pointer receivers on structs that implement interfaces (required by Go to satisfy the interface with mutating methods).

| Site | Rule | Example |
|------|------|---------|
| Repository interface methods | Value params, value returns | `Create(ctx, Shop) (Shop, error)` |
| Service method returns | Value returns | `SignUp(...) (SignUpResponse, error)` |
| Implementing struct receivers | Pointer receiver allowed | `func (r *ShopRepository) Create(...)` |
| Constructor return types | Pointer allowed | `func NewShopRepository(...) *ShopRepository` |

Because `Create` can no longer mutate a caller-owned pointer, it receives a partially-filled value, scans the DB-generated fields into its local copy, and returns the completed value:

```go
// infra/repository/shop_repo.go

func (r *ShopRepository) Create(ctx context.Context, shop identity.Shop) (identity.Shop, error) {
    var createdAt, updatedAt time.Time
    err := r.q.QueryRow(ctx,
        `INSERT INTO shop (...) RETURNING id::text, created_at, updated_at`, ...,
    ).Scan(&shop.ID, &createdAt, &updatedAt)
    if err != nil {
        return identity.Shop{}, err
    }
    shop.CreatedAt = createdAt.Unix()
    shop.UpdatedAt = updatedAt.Unix()
    return shop, nil
}
```

### Entity field types

Domain entities use infrastructure-agnostic types:

| Field | Go type | Notes |
|-------|---------|-------|
| IDs (`ID`, `ShopID`, `UserID`) | `string` | Standard UUID format `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| Timestamps (`CreatedAt`, `UpdatedAt`) | `int64` | Unix seconds (UTC) |

The infrastructure layer bridges to PostgreSQL types via SQL casts:

| Direction | Technique | Example |
|-----------|-----------|---------|
| UUID → string (output) | `id::text` in SQL | `RETURNING id::text` |
| string → UUID (input) | `$1::uuid` in SQL | `WHERE id = $1::uuid` |
| `timestamptz` → int64 | Scan into `time.Time`, call `.Unix()` | `createdAt.Unix()` |

The HTTP layer (DTOs in `routes.go`) converts `int64` timestamps to RFC 3339 strings for API consumers: `time.Unix(resp.Shop.CreatedAt, 0).UTC().Format(time.RFC3339)`.

### Infrastructure (`internal/infra/repository/`)

A single `repository` package holds every PostgreSQL adapter. Each domain entity gets its own struct that implements the corresponding repository interface:

| Struct | Implements |
|--------|-----------|
| `ShopRepository` | `identity.ShopRepository` |
| `UserRepository` | `identity.UserRepository` |
| `MembershipRepository` | `identity.MembershipRepository` |

Constructor functions accept `*pgxpool.Pool` as a fallback querier:

```go
func NewShopRepository(pool *pgxpool.Pool) *ShopRepository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository
func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository
```

---

## The Transaction Pattern

Transactions are owned by the HTTP middleware layer, not by services or repositories. This keeps the domain free of any connection or transaction management logic.

### Flow

```
HTTP request
  └─ Transaction middleware (internal/infra/http)
       ├─ pool.Begin(ctx)            ← open tx
       ├─ database.WithTx(ctx, tx)   ← store tx in context
       ├─ c.Next()                   ← handler runs
       │     └─ Service.SignUp(ctx, ...)
       │           └─ ShopRepository.Create(ctx, ...)
       │                 └─ database.QuerierFromCtx(ctx, pool)  ← returns tx
       │                       └─ tx.QueryRow(...)
       └─ status < 400 → tx.Commit()
          status ≥ 400 → tx.Rollback()
```

### Context plumbing (`internal/database/tx.go`)

```go
// Store the transaction in context (called by the middleware).
func WithTx(ctx context.Context, tx pgx.Tx) context.Context

// Retrieve the querier: returns tx if present, pool otherwise.
func QuerierFromCtx(ctx context.Context, pool *pgxpool.Pool) Querier
```

### Middleware (`internal/infra/http/middleware.go`)

```go
func Transaction(pool *pgxpool.Pool) gin.HandlerFunc {
    return func(c *gin.Context) {
        log := logger.FromContext(c.Request.Context())
        tx, err := pool.Begin(c.Request.Context())
        // ... error handling ...
        ctx := database.WithTx(c.Request.Context(), tx)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
        if c.Writer.Status() >= 400 { _ = tx.Rollback(ctx); return }
        if err := tx.Commit(ctx); err != nil { _ = tx.Rollback(ctx); log.Error(...) }
    }
}
```

> **Trade-off:** Gin writes the response into its internal buffer during handler execution. If `Commit` fails after a successful handler (rare), the client has already received a 2xx status — the commit error is logged but cannot be surfaced to the caller. This is the accepted trade-off of the transaction-per-request pattern without response buffering.

### Repository usage

Each repository method calls `database.QuerierFromCtx` to pick up the active transaction automatically:

```go
func (r *ShopRepository) Create(ctx context.Context, shop identity.Shop) (identity.Shop, error) {
    q := database.QuerierFromCtx(ctx, r.pool) // tx if middleware ran, pool otherwise
    err := q.QueryRow(ctx, `INSERT INTO shop (...) RETURNING id::text, ...`, ...).Scan(...)
    ...
}
```

Repositories work the same way in both transactional and non-transactional contexts — the caller never needs to know which is active.

---

## Wiring (cmd/api/main.go)

`main.go` uses **Uber FX** for dependency injection. It is the only place that imports both a domain package and its infrastructure adapter. All constructors are registered with `fx.Provide`; startup side-effects (migrations, route registration, server lifecycle) are registered with `fx.Invoke`.

```go
fx.New(
    fx.WithLogger(...),        // route FX logs through zap
    fx.Provide(
        config.Load,
        newLogger,             // wraps logger.New; adds OnStop log.Sync()
        newPool,               // wraps database.New; adds OnStop pool.Close()
    ),
    fx.Provide(
        fx.Annotate(repository.NewShopRepository, fx.As(new(identity.ShopRepository))),
        fx.Annotate(repository.NewUserRepository, fx.As(new(identity.UserRepository))),
        fx.Annotate(repository.NewMembershipRepository, fx.As(new(identity.MembershipRepository))),
    ),
    fx.Provide(
        fx.Annotate(newIdentityService, fx.As(new(identity.Signer))),
        newRouter,
    ),
    fx.Invoke(
        runMigrations,         // runs synchronously before server starts
        registerRoutes,        // wires v1 group + Transaction middleware
        runServer,             // appends OnStart/OnStop lifecycle hooks
    ),
).Run()
```

`fx.As` binds concrete repository structs to their domain interfaces, keeping the service layer free of infrastructure import paths. FX handles signal catching and graceful shutdown automatically via `app.Run()`.

---

## Adding a New Domain

When implementing a new domain (e.g., `booking`):

1. **Create `internal/booking/`** with `entity.go`, `repository.go`, `errors.go`, `service.go`.
2. **Define interfaces** in `repository.go` — no pgx imports.
3. **Add repository structs** to `internal/infra/repository/` — one file per entity (e.g., `appointment_repo.go`).
4. **Add HTTP handlers** to `internal/infra/http/booking.go` — `RegisterBookingRoutes(rg, svc)`, DTOs, and `toXxxDTO` helpers.
5. **Wire in `main.go`** — construct repos + service, call `apihttp.RegisterBookingRoutes(v1, svc)`.

The domain package stays free of any `pgx`, `gin`, or other infrastructure imports throughout.

---

## Testing Strategy

### Libraries

| Library | Role |
|---------|------|
| `github.com/stretchr/testify/assert` | Non-fatal assertions — test continues after failure |
| `github.com/stretchr/testify/require` | Fatal assertions — test stops immediately on failure |
| `github.com/stretchr/testify/suite` | Test suites — used when tests share mock setup |
| `github.com/vektra/mockery/v2` | Mock generation from interfaces |

Use `require` for anything that makes the remainder of the test meaningless if it fails (setup calls, first assertions that later assertions depend on). Use `assert` for the rest.

### Test layers

| Layer | Package | Structure | Mock tool |
|-------|---------|-----------|-----------|
| Domain service logic | `internal/identity` (package `identity_test`) | `testify/suite` | mockery-generated repo mocks |
| HTTP handlers | `internal/infra/http` (package `http_test`) | `testify/suite` | mockery-generated `MockSigner` |
| Pure domain helpers (`slug`, `tenant`) | `internal/identity` (package `identity` or `identity_test`) | plain `TestXxx` functions | none |
| Repository structs | `internal/infra/repository` (build tag `integration`) | plain `TestXxx` functions | none — live Postgres |
| Service end-to-end | `internal/infra/repository` (build tag `integration`) | plain `TestXxx` functions | none — live Postgres |

### Mock generation

Mocks live in `internal/<domain>/mocks/` and are generated by mockery from the domain's interfaces. The config is at `backend/.mockery.yaml`. The generate directive is in `internal/<domain>/generate.go`:

```
cd backend && go generate ./internal/identity/...
```

Never edit files in `mocks/` by hand — they start with `// Code generated by mockery`.

### Why suites for tests with mocks

A `testify/suite` struct holds the mocks as fields and resets them in `SetupTest()` before each test. This guarantees every test starts with a clean mock and avoids cross-test pollution without repeating construction boilerplate.

```go
type SignUpSuite struct {
    suite.Suite
    shops   *mocks.MockShopRepository
    users   *mocks.MockUserRepository
    members *mocks.MockMembershipRepository
    svc     *identity.Service
}

func (s *SignUpSuite) SetupTest() {
    s.shops   = mocks.NewMockShopRepository(s.T()) // auto-asserts expectations on cleanup
    s.users   = mocks.NewMockUserRepository(s.T())
    s.members = mocks.NewMockMembershipRepository(s.T())
    s.svc     = identity.NewService(s.shops, s.users, s.members, identity.WithBcryptCost(bcrypt.MinCost))
}
```

`NewMockXxx(s.T())` registers `AssertExpectations` as a cleanup function — any `.EXPECT()` call not satisfied causes the test to fail automatically.

### bcrypt cost in unit tests

`identity.NewService` defaults to bcrypt cost 12 (~300 ms/call). Pass `identity.WithBcryptCost(bcrypt.MinCost)` in unit test suites to keep them fast without modifying production behaviour.

### Commands

Run unit tests:
```
cd backend && APP_ENV=test go test -short ./...
```

Run integration tests (requires `DATABASE_URL`):
```
cd backend && go test -tags integration ./internal/infra/repository/...
```

The `database.Querier` interface (`internal/database/querier.go`) is satisfied by both `*pgxpool.Pool` and `pgx.Tx`, so repository structs work in both unit mocks and integration tests without modification.
