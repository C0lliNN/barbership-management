# Item 009: Authentication & Login (API)

**Stage:** 1 — Identity, Tenancy & Authentication
**Status:** 📋 Planned
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-06-06

---

## Description

Implement email/password authentication: a `POST /v1/login` endpoint that verifies
credentials and issues a **signed JWT**, a `POST /v1/logout` stub (client-side token
discard), and a `GET /v1/me` protected endpoint that exercises the auth middleware. The
`AuthRequired` Gin middleware validates the `Authorization: Bearer <token>` header on
every protected route and puts the parsed claims into the request context.

**Session strategy (confirmed):** JWT (stateless), HMAC-SHA256 via
`github.com/golang-jwt/jwt/v5`. No session table. Logout is client-side. Server-side
revocation can be added in a later item if needed.

---

### API Contracts

#### `POST /v1/login`

```
POST /v1/login
Content-Type: application/json

{
  "email":    "joao@barbearia.com.br",
  "password": "SecretPass123"
}

200 OK
{
  "token":   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9…",
  "user": {
    "id":         "b4e2…",
    "email":      "joao@barbearia.com.br",
    "full_name":  "João Silva",
    "phone":      "+5511999999999"
  },
  "shop_id": "c7f1…",   // primary shop (first membership); empty string if none
  "role":    "owner"    // role in primary shop; empty string if none
}

401 Unauthorized  — wrong email or password (identical message for both; no enumeration)
422 Unprocessable Entity — missing or blank email/password
```

#### `POST /v1/logout`

```
POST /v1/logout
Authorization: Bearer <token>   (optional — no server state to clear)

200 OK
{}
```

Client must discard the token. No server-side action is taken.

#### `GET /v1/me`  _(protected stub — validates auth middleware in tests)_

```
GET /v1/me
Authorization: Bearer <token>

200 OK
{
  "user_id": "b4e2…",
  "email":   "joao@barbearia.com.br",
  "shop_id": "c7f1…",
  "role":    "owner"
}

401 Unauthorized — missing, invalid, or expired token
```

---

### JWT Token Format

- **Algorithm:** HS256 (HMAC-SHA256)
- **Claims:**
  - `sub` — user ID (standard `jwt.RegisteredClaims.Subject`)
  - `email` — user email (custom claim)
  - `shop_id` — primary shop ID (custom claim; empty if user has no memberships)
  - `role` — role in primary shop (custom claim; empty if user has no memberships)
  - `iat` — issued-at (standard)
  - `exp` — expires-at (standard; default 24 h, configurable via `JWT_EXPIRY`)
- **Secret:** `JWT_SECRET` env var (required; min 32 characters; app refuses to start if absent)

---

## Acceptance Criteria

- [x] `POST /v1/login` with valid credentials returns `200` with a signed JWT, user
  info, and primary shop/role. _(TestLoginHandlerSuite/TestLoginHappyPath,
  TestLoginSuite/TestHappyPathWithMembership)_
- [x] `POST /v1/login` with wrong password returns `401 {"error":"invalid email or
  password"}` — identical message to unknown email (no enumeration).
  _(TestLoginHandlerSuite/TestLoginWrongPassword, TestLoginSuite/TestWrongPassword)_
- [x] `POST /v1/login` with unknown email returns `401` with the same generic message.
  _(TestLoginHandlerSuite/TestLoginUnknownEmail, TestLoginSuite/TestUnknownEmail)_
- [x] `POST /v1/login` with missing `email` or `password` returns `422`.
  _(TestLoginHandlerSuite/TestLoginMissingEmail, .../TestLoginMissingPassword)_
- [x] Returned JWT decodes to correct `sub`, `email`, `shop_id`, `role`, `exp`, `iat`.
  _(TestSignAndParseTokenRoundTrip)_
- [x] `GET /v1/me` with a valid token returns `200` with claims from the token.
  _(TestLoginHandlerSuite/TestMeWithValidToken, TestAuthRequiredPassesValidToken)_
- [x] `GET /v1/me` with no `Authorization` header returns `401`.
  _(TestLoginHandlerSuite/TestMeNoHeader, TestAuthRequiredAbortsNoHeader)_
- [x] `GET /v1/me` with a malformed token returns `401`.
  _(TestLoginHandlerSuite/TestMeMalformedToken, TestAuthRequiredAbortsMalformedToken)_
- [x] `GET /v1/me` with an expired token returns `401`.
  _(TestLoginHandlerSuite/TestMeExpiredToken, TestAuthRequiredAbortsExpiredToken)_
- [x] `POST /v1/logout` returns `200 {}` regardless of whether a token is present.
  _(TestLoginHandlerSuite/TestLogout, .../TestLogoutWithToken)_
- [x] `JWT_SECRET` absent → service refuses to start with a clear error message.
  _(TestLoadInvalid/missing_JWT_SECRET_in_non-test_env, .../JWT_SECRET_too_short...)_
- [x] `JWT_EXPIRY` is configurable (env var); defaults to `24h`.
  _(TestLoadDefaults, TestLoadOverrides)_
- [x] `go build ./...`, `go vet ./...`, `APP_ENV=test go test -short ./...` all pass.
- [ ] Integration tests (with `DATABASE_URL`) covering login happy path, wrong
  password, and `GET /v1/me` round-trip pass. **Written** (`TestService_Login_EndToEnd`,
  `TestMembershipRepository_ListByUser`) and confirmed to compile/skip cleanly without
  a database, but **not run against a live Postgres** — Docker is unavailable in this
  environment. See Validation Results for the follow-up command to run elsewhere.

---

## New Library Decision (confirmed before implementing)

### `github.com/golang-jwt/jwt/v5`

The actively maintained standard Go JWT library. No alternatives needed for v1.

```bash
cd backend
go get github.com/golang-jwt/jwt/v5
go mod tidy
```

Add this import only to `internal/identity/token.go` (signs tokens) and
`internal/infra/http/middleware.go` (validates tokens). No other files should import
it directly.

---

## Implementation Steps

### 1. Add `JWT_SECRET` and `JWT_EXPIRY` to `internal/config/config.go`

Add fields to `Config`:

```go
JWTSecret string
JWTExpiry time.Duration
```

In `Load()`:

```go
cfg.JWTExpiry = 24 * time.Hour // default

cfg.JWTSecret = os.Getenv("JWT_SECRET")
if cfg.JWTSecret == "" && cfg.Env != "test" {
    return Config{}, fmt.Errorf("JWT_SECRET is required")
}
if len(cfg.JWTSecret) < 32 && cfg.Env != "test" {
    return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
}

if v := os.Getenv("JWT_EXPIRY"); v != "" {
    d, err := time.ParseDuration(v)
    if err != nil {
        return Config{}, fmt.Errorf("invalid JWT_EXPIRY %q: %w", v, err)
    }
    cfg.JWTExpiry = d
}
```

Update `.env.example` (backend):
```
JWT_SECRET=changeme-change-in-production-min32chars
JWT_EXPIRY=24h
```

### 2. Add `ListByUser` to `MembershipRepository`

In `internal/identity/repository.go`, add to `MembershipRepository`:

```go
// ListByUser returns all memberships for a given user across all shops.
ListByUser(ctx context.Context, userID string) ([]Membership, error)
```

In `internal/infra/repository/membership_repo.go`, implement:

```go
func (r *MembershipRepository) ListByUser(ctx context.Context, userID string) ([]Membership, error) {
    q := database.QuerierFromCtx(ctx, r.pool)
    rows, err := q.Query(ctx,
        `SELECT id::text, shop_id::text, user_id::text, role, created_at
         FROM membership WHERE user_id = $1::uuid ORDER BY created_at ASC`, userID)
    if err != nil {
        return nil, fmt.Errorf("membership list by user: %w", err)
    }
    defer rows.Close()

    var out []Membership
    for rows.Next() {
        var m Membership
        var createdAt time.Time
        if err := rows.Scan(&m.ID, &m.ShopID, &m.UserID, &m.Role, &createdAt); err != nil {
            return nil, fmt.Errorf("membership list by user scan: %w", err)
        }
        m.CreatedAt = createdAt.Unix()
        out = append(out, m)
    }
    return out, rows.Err()
}
```

Regenerate the mock after updating the interface:
```bash
cd backend
go generate ./internal/identity/...
```

This updates `internal/identity/mocks/mock_membershiprepository.go` with a `ListByUser` method.

### 3. Add `ErrInvalidCredentials` to `internal/identity/errors.go`

```go
ErrInvalidCredentials = errors.New("invalid email or password")
```

### 4. Create `internal/identity/token.go`

```go
package identity

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload embedded in each access token.
type Claims struct {
    Email  string `json:"email"`
    ShopID string `json:"shop_id,omitempty"`
    Role   string `json:"role,omitempty"`
    jwt.RegisteredClaims
}

// signToken creates a signed JWT for the given user and primary shop membership.
func signToken(secret string, expiry time.Duration, user User, shopID string, role Role) (string, error) {
    now := time.Now()
    claims := Claims{
        Email:  user.Email,
        ShopID: shopID,
        Role:   string(role),
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   user.ID,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(secret))
    if err != nil {
        return "", fmt.Errorf("sign token: %w", err)
    }
    return signed, nil
}

// ParseToken validates a JWT string and returns its claims.
// Returns an error if the token is invalid, expired, or uses an unexpected algorithm.
func ParseToken(secret, tokenStr string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return []byte(secret), nil
    })
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }
    return claims, nil
}
```

### 5. Add `Login` to `Signer` interface and implement on `Service`

In `internal/identity/service.go`, extend `Signer`:

```go
type Signer interface {
    SignUp(ctx context.Context, req SignUpRequest) (SignUpResponse, error)
    Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
}
```

Add request/response types (in `service.go` or `service_types.go`):

```go
type LoginRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
    Token  string `json:"token"`
    User   User   `json:"user"`
    ShopID string `json:"shop_id"`
    Role   Role   `json:"role"`
}
```

Add `jwtSecret` and `jwtExpiry` fields to `Service`, plus options:

```go
type Service struct {
    shops       ShopRepository
    users       UserRepository
    memberships MembershipRepository
    bcryptCost  int
    jwtSecret   string
    jwtExpiry   time.Duration
}

func WithJWTSecret(secret string) Option {
    return func(s *Service) { s.jwtSecret = secret }
}

func WithJWTExpiry(d time.Duration) Option {
    return func(s *Service) { s.jwtExpiry = d }
}
```

Set defaults in `NewService`:
```go
jwtExpiry: 24 * time.Hour,
```

Implement `Login`:

```go
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
    user, err := s.users.GetByEmail(ctx, req.Email)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return LoginResponse{}, ErrInvalidCredentials
        }
        return LoginResponse{}, fmt.Errorf("login lookup: %w", err)
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        return LoginResponse{}, ErrInvalidCredentials
    }

    // Determine primary shop and role.
    var shopID string
    var role Role
    memberships, err := s.memberships.ListByUser(ctx, user.ID)
    if err != nil {
        return LoginResponse{}, fmt.Errorf("login memberships: %w", err)
    }
    if len(memberships) > 0 {
        shopID = memberships[0].ShopID
        role = memberships[0].Role
    }

    token, err := signToken(s.jwtSecret, s.jwtExpiry, user, shopID, role)
    if err != nil {
        return LoginResponse{}, fmt.Errorf("login sign: %w", err)
    }

    return LoginResponse{Token: token, User: user, ShopID: shopID, Role: role}, nil
}
```

### 6. Update `mock_signer.go`

After running `go generate ./internal/identity/...`, verify that
`internal/identity/mocks/mock_signer.go` contains a `Login` method stub. If mockery
doesn't pick it up automatically, add it manually following the existing pattern.

### 7. Add `AuthRequired` middleware to `internal/infra/http/middleware.go`

```go
// AuthRequired validates the JWT in the Authorization header and stores parsed
// claims in the Gin context under the key "claims".
func AuthRequired(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        header := c.GetHeader("Authorization")
        const prefix = "Bearer "
        if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed token"})
            return
        }
        tokenStr := header[len(prefix):]
        claims, err := identity.ParseToken(jwtSecret, tokenStr)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
            return
        }
        c.Set("claims", claims)
        c.Next()
    }
}
```

### 8. Add login, logout, and `/me` handlers to `internal/infra/http/identity.go`

Extend `RegisterIdentityRoutes`:

```go
func RegisterIdentityRoutes(rg *gin.RouterGroup, svc identity.Signer, jwtSecret string) {
    rg.POST("/signup", handleSignUp(svc))
    rg.POST("/login",  handleLogin(svc))
    rg.POST("/logout", handleLogout())

    protected := rg.Group("", AuthRequired(jwtSecret))
    protected.GET("/me", handleMe())
}
```

Handler implementations:

```go
func handleLogin(svc identity.Signer) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req identity.LoginRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusUnprocessableEntity, formatValidationError(err))
            return
        }
        resp, err := svc.Login(c.Request.Context(), req)
        if err != nil {
            if errors.Is(err, identity.ErrInvalidCredentials) {
                c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
                return
            }
            logger.FromContext(c.Request.Context()).Error("login failed", zap.Error(err))
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
            return
        }
        c.JSON(http.StatusOK, toLoginDTO(resp))
    }
}

func handleLogout() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{})
    }
}

func handleMe() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims, _ := c.Get("claims")
        cl := claims.(*identity.Claims)
        c.JSON(http.StatusOK, gin.H{
            "user_id": cl.Subject,
            "email":   cl.Email,
            "shop_id": cl.ShopID,
            "role":    cl.Role,
        })
    }
}
```

Add the `loginDTO` and `toLoginDTO` helper (near the existing DTOs):

```go
type loginDTO struct {
    Token  string   `json:"token"`
    User   ownerDTO `json:"user"`
    ShopID string   `json:"shop_id"`
    Role   string   `json:"role"`
}

func toLoginDTO(resp identity.LoginResponse) loginDTO {
    return loginDTO{
        Token: resp.Token,
        User: ownerDTO{
            ID:        resp.User.ID,
            Email:     resp.User.Email,
            FullName:  resp.User.FullName,
            Phone:     resp.User.Phone,
            CreatedAt: time.Unix(resp.User.CreatedAt, 0).UTC().Format(time.RFC3339),
        },
        ShopID: resp.ShopID,
        Role:   string(resp.Role),
    }
}
```

### 9. Update `RegisterIdentityRoutes` call signature in `cmd/api/main.go`

Update `registerRoutes` to pass the JWT secret:

```go
func registerRoutes(engine *gin.Engine, pool *pgxpool.Pool, svc identity.Signer, cfg config.Config) {
    v1 := engine.Group("/v1", apihttp.Transaction(pool))
    apihttp.RegisterIdentityRoutes(v1, svc, cfg.JWTSecret)
}
```

Update `newIdentityService` to wire JWT config from `Config`:

```go
func newIdentityService(
    shops identity.ShopRepository,
    users identity.UserRepository,
    members identity.MembershipRepository,
    cfg config.Config,
) *identity.Service {
    return identity.NewService(shops, users, members,
        identity.WithJWTSecret(cfg.JWTSecret),
        identity.WithJWTExpiry(cfg.JWTExpiry),
    )
}
```

### 10. Install dependency

```bash
cd backend
go get github.com/golang-jwt/jwt/v5
go mod tidy
```

---

## Testing Strategy

| Layer | Tool | When |
|-------|------|------|
| `ParseToken` / `signToken` unit | `go test` (stdlib) | Always |
| Handler — login (unit) | `httptest` + mock `Signer` | Always |
| Handler — `/me` (unit) | `httptest` + real middleware | Always |
| `AuthRequired` middleware (unit) | `httptest`, generate tokens in-test | Always |
| Service `Login` (unit) | mock repos | Always |
| Repository `ListByUser` (integration) | `go test -tags integration` | Requires live Postgres |
| End-to-end login → `/me` | integration test against full stack | Requires live Postgres |
| Manual curl | sign up → login → `/me` | Before marking complete |

### Handler tests (`internal/infra/http/identity_test.go`)

Extend existing file. Add test cases:

| Scenario | Input | Expected |
|----------|-------|----------|
| Login happy path | Valid email + password, mock returns `LoginResponse` | `200`, token in body |
| Wrong password | Mock returns `ErrInvalidCredentials` | `401 {"error":"invalid email or password"}` |
| Unknown email | Mock returns `ErrInvalidCredentials` | `401` same message |
| Missing email | `email` omitted | `422` |
| Missing password | `password` omitted | `422` |
| `/me` with valid token | Signed token in header | `200` with claims |
| `/me` no header | No `Authorization` | `401` |
| `/me` bad token | `Authorization: Bearer garbage` | `401` |
| `/me` expired token | Token with past `exp` | `401` |

### `AuthRequired` middleware tests (`internal/infra/http/middleware_test.go`)

Generate test tokens using `identity.ParseToken` / a small helper that calls
`identity.SignToken` with a known secret. Verify pass-through vs abort behavior.

### Service tests (`internal/identity/service_test.go`)

Add test cases:
- `Login` returns `LoginResponse` with non-empty `Token` on success.
- `Login` returns `ErrInvalidCredentials` when `GetByEmail` returns `ErrNotFound`.
- `Login` returns `ErrInvalidCredentials` when bcrypt compare fails.
- `Login` embeds correct `ShopID` and `Role` when membership exists.
- `Login` leaves `ShopID`/`Role` empty when no memberships.

### Token unit tests (`internal/identity/token_test.go`)

```
signToken → ParseToken round-trip: claims match input
ParseToken with wrong secret → error
ParseToken with expired token → error
ParseToken with tampered payload → error
ParseToken with wrong alg in header → error
```

### Repository integration tests (`internal/infra/repository/identity_test.go`)

Add `ListByUser` test: create a user + membership, call `ListByUser`, verify the
membership is returned with correct shop_id and role.

---

## Dependencies

- **Upstream:** Item 008 ✅ — `UserRepository.GetByEmail` and bcrypt hashing in place;
  `MembershipRepository.Create` + `ListByShop` already implemented.
- **Downstream (enables):**
  - Item 010 (Authorization & Frontend auth) — depends on `AuthRequired` middleware
    and the `Claims` type defined here.
- **New direct Go dependency:** `github.com/golang-jwt/jwt/v5`
- **No new migrations** — stateless JWT; no session table needed.

---

## Testing Prerequisites

### Required Services

| Service | Version | Start Command | Port |
|---------|---------|---------------|------|
| PostgreSQL | 16 | `cd backend && make db-start` | 5432 |

### Environment Configuration

| Variable | Required | Example |
|----------|----------|---------|
| `DATABASE_URL` | Yes (integration) | `postgres://barber:secret@localhost:5432/barbershop?sslmode=disable` |
| `JWT_SECRET` | Yes (runtime) | `changeme-change-in-production-min32chars` |
| `JWT_EXPIRY` | No | `24h` |

`APP_ENV=test` bypasses both `DATABASE_URL` and `JWT_SECRET` validation for unit tests.

### Manual Validation Checklist

- [ ] **Build:** `cd backend && go build ./...`
- [ ] **Vet:** `cd backend && go vet ./...`
- [ ] **Short tests:** `cd backend && APP_ENV=test go test -short ./...`
- [ ] **Start postgres:** `cd backend && make db-start`
- [ ] **Integration tests:** `cd backend && go test -tags integration ./...`
- [ ] **Start API:**
  ```bash
  cd backend
  JWT_SECRET=changeme-change-in-production-min32chars make dev-local
  ```
- [ ] **Sign up (if no account exists):**
  ```bash
  curl -s -X POST http://localhost:8080/v1/signup \
    -H 'Content-Type: application/json' \
    -d '{
      "shop":  {"name":"Barbearia do João","city":"São Paulo","state":"SP"},
      "owner": {"email":"joao@test.com","password":"Secret123","full_name":"João Silva"}
    }' | jq .
  ```
- [ ] **Login happy path:**
  ```bash
  TOKEN=$(curl -s -X POST http://localhost:8080/v1/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"joao@test.com","password":"Secret123"}' | jq -r .token)
  echo "Token: $TOKEN"
  ```
  Expected: `200` — JSON with `token` (non-empty JWT string), `user`, `shop_id`, `role = "owner"`.
- [ ] **Wrong password:**
  ```bash
  curl -s -X POST http://localhost:8080/v1/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"joao@test.com","password":"wrong"}' | jq .
  ```
  Expected: `401 {"error":"invalid email or password"}`.
- [ ] **Protected route (`/me`):**
  ```bash
  curl -s http://localhost:8080/v1/me \
    -H "Authorization: Bearer $TOKEN" | jq .
  ```
  Expected: `200` with `user_id`, `email`, `shop_id`, `role`.
- [ ] **`/me` with no token:**
  ```bash
  curl -s http://localhost:8080/v1/me | jq .
  ```
  Expected: `401 {"error":"missing or malformed token"}`.
- [ ] **Logout:**
  ```bash
  curl -s -X POST http://localhost:8080/v1/logout \
    -H "Authorization: Bearer $TOKEN" | jq .
  ```
  Expected: `200 {}`.

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `POST /v1/login` (valid) | `200` — JWT + user + shop_id + role |
| `POST /v1/login` (wrong password) | `401 {"error":"invalid email or password"}` |
| `POST /v1/login` (unknown email) | `401` same message |
| `POST /v1/login` (missing field) | `422 {"error":"validation failed","details":[…]}` |
| `GET /v1/me` (valid token) | `200` with correct claims |
| `GET /v1/me` (no token) | `401 {"error":"missing or malformed token"}` |
| `GET /v1/me` (expired token) | `401 {"error":"invalid or expired token"}` |
| `POST /v1/logout` | `200 {}` |
| JWT decode | `sub` = user UUID, `email`, `shop_id`, `role`, `exp` ≈ now + 24 h |

### Validation Results

```markdown
## Validation Results
- [x] Build: `go build ./...` passes
- [x] Vet: `go vet ./...` passes
- [x] Short tests: `APP_ENV=test go test -short ./...` passes — all identity,
      http, and config suites green, including the new TestLoginSuite,
      TestLoginHandlerSuite, TestSignAndParseTokenRoundTrip /
      TestParseToken* suite, and TestAuthRequired* middleware suite.
- [x] `JWT_SECRET` absent / too short in non-test env → config.Load() returns a
      clear error (covered by TestLoadInvalid/missing_JWT_SECRET_in_non-test_env
      and .../JWT_SECRET_too_short_in_non-test_env).
- [x] `JWT_EXPIRY` configurable via env, defaults to 24h (TestLoadDefaults,
      TestLoadOverrides).
- [ ] Service started: PostgreSQL 16 (Docker) — NOT RUN. Docker is unavailable
      in this WSL2 environment ("docker: could not be found"), and no local
      Postgres instance is installed, so `make db-start` / `make dev-local`
      cannot run here.
- [ ] Application started successfully — NOT RUN (depends on Postgres above).
- [ ] Integration tests: `go test -tags integration ./...` — compiles cleanly
      (verified `go vet -tags integration ./internal/infra/repository/...`) and
      all repository/service integration tests, including the new
      TestMembershipRepository_ListByUser(_Empty) and
      TestService_Login_EndToEnd, skip cleanly with "DATABASE_URL not set".
      NOT actually executed against a live database — needs to be run in an
      environment with Docker or a local Postgres 16 instance.
- [ ] Manual curl walkthrough (signup → login → /me → logout) — NOT RUN, same
      Docker/Postgres limitation. The equivalent request/response flows are
      covered by TestLoginHandlerSuite (mocked Signer) and
      TestService_Login_EndToEnd (real Postgres, when run with DATABASE_URL).
- [ ] Database tables verified: no new tables expected (stateless JWT) — not
      independently verified against a live DB for the reason above.
- [ ] Seed data verified: N/A
- [ ] Screenshots captured: N/A (no UI changes)

**Action needed before this item can be considered fully verified:** run
`make db-start && DATABASE_URL=... go test -tags integration ./...` and the
manual curl checklist above on a machine with Docker (or a local Postgres 16),
per the spec's Manual Validation Checklist.
```

---

## Decisions & Trade-offs

- **JWT (stateless) over server session** — confirmed by user. No `session` table
  needed. Token expiry enforces rotation. Logout is client-side for v1. Trade-off:
  cannot invalidate a stolen token without a blacklist. Acceptable for v1; add
  server-side revocation in a later item if required.

- **HS256 over RS256** — symmetric signing is simpler (no key-pair management) and
  sufficient while there is only one backend service. Switch to RS256 if a second
  service needs to verify tokens independently.

- **Primary shop = first membership by created_at** — a user can theoretically belong
  to multiple shops. For v1, embedding the first membership's shop_id and role in the
  JWT is sufficient. When multi-shop support is needed, the token can be scoped per
  request or a new "active shop" selection flow can be added.

- **`APP_ENV=test` bypasses JWT_SECRET validation** — consistent with how
  `DATABASE_URL` is bypassed in tests. Handler/service tests supply a short
  in-test secret like `"test-secret"` via `WithJWTSecret`.

- **`POST /v1/logout` returns 200 unconditionally** — no Authorization header
  required. This keeps the frontend logout path simple: always call logout, always
  discard the local token, regardless of server response.

- **`ListByUser` added to `MembershipRepository`** — the interface gains one method.
  The existing mock is regenerated via `go generate`. Downstream code (Item 010
  authorization middleware) will also benefit from this query.

- **`signToken` exported as `identity.SignToken`** — the spec's middleware-test
  guidance referenced `identity.SignToken`, but the implementation sketch defined
  it as unexported `signToken`. Exported it (used internally by `Service.Login`
  and externally by `internal/infra/http` test files in `package http`/`http_test`,
  which cannot reach an unexported function in another package) so tests can mint
  valid tokens for `AuthRequired` and `/me` round-trips without duplicating the
  signing logic.

- **Fixed pre-existing compile error in `internal/infra/repository/identity_test.go`**
  — the integration test file (added in Item 008, build-tagged `integration`)
  referenced `repository.NewTxManager(pool)` and a two-argument `identity.NewService`
  signature that don't exist in the current code (the constructor takes the three
  repositories directly). This was a latent bug: `go vet ./...` (no integration tag)
  never compiled the file, so it went unnoticed. Fixed by wiring the real
  `ShopRepository`/`UserRepository`/`MembershipRepository` and adding a `runInTx`
  helper that mirrors the production `Transaction` middleware (commit on success,
  rollback on error) so `TestService_SignUp_DuplicateEmail_Rollback` still exercises
  real rollback semantics. This was required to make the package compile at all —
  without it, the new `ListByUser`/`Login` integration tests couldn't be added.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move **"Authentication (email/password; JWT or server session) and login"** from
  📋 → ✅.
- Stage 1 remains 🚧 (Item 010 still pending).
- Only update rows corresponding to **Item 009**.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 009` to implement
this work item.
