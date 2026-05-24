# Item 002: Go API Skeleton (Router, Config, Logging, Health)

**Stage:** 0 — Foundation & Scaffolding
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-24

---

## Description

Turn the buildable skeleton from Item 001 into a **running HTTP API service**. This
item wires up the cross-cutting plumbing every endpoint will rely on:
configuration loading (env-based), structured logging, an HTTP router with
middleware, graceful shutdown, and the operational endpoints `GET /health` and
`GET /ready`.

This is still infrastructure — **no domain/business logic and no database**. The DB
connection and a DB-aware `/ready` arrive in Item 003. Here, `/ready` returns a
simple "the process is up and serving" readiness signal (no dependency checks yet).

### Scope
- `internal/config`: load and validate configuration from environment variables
  (port, log level, environment name) with sane defaults.
- `internal/http`: router construction, base middleware (request logging, panic
  recovery, request ID), and the health/ready handlers.
- `cmd/api/main.go`: compose config → logger → server, start listening, and shut
  down gracefully on SIGINT/SIGTERM.
- A minimal test hitting `/health` (and ideally `/ready`).

---

## Acceptance Criteria

- [x] `internal/config` loads configuration from env with documented defaults and
      validates them (invalid port / log level → clear startup error).
- [x] Structured logging via **`go.uber.org/zap`** (JSON encoder) is initialized from
      config and used by middleware and startup/shutdown logs.
- [x] `internal/http` exposes a router with base middleware: request logging, panic
      recovery, and a request-ID per request.
- [x] `GET /health` returns **200** with a small JSON body (`{"status":"ok"}`).
- [x] `GET /ready` returns **200** with a JSON body indicating readiness
      (`{"status":"ready"}`; no dependency checks yet — placeholder for DB in Item 003).
- [x] `cmd/api` starts an `http.Server` and shuts down gracefully on SIGINT/SIGTERM
      (in-flight requests drained within `ShutdownTimeout`).
- [x] Tests exercise `GET /health` and `GET /ready` and assert 200 + expected body.
- [x] `go build ./...`, `go vet ./...`, and `go test ./...` all pass from `backend/`.
- [x] Service starts locally and `curl localhost:8080/health` returns 200.
- [x] No domain logic and no database code introduced.

---

## Implementation Steps

1. **Config (`internal/config`)**
   - Define a `Config` struct: `Port` (default `8080`), `LogLevel` (default `info`),
     `Env` (default `development`), `ShutdownTimeout` (default `10s`),
     `ReadTimeout`/`WriteTimeout` (sane defaults).
   - `Load()` reads from env (`PORT`, `LOG_LEVEL`, `APP_ENV`, etc.), applies defaults,
     validates (port numeric/in range; log level one of debug/info/warn/error), and
     returns a typed error on invalid input.
   - Keep it dependency-free (stdlib `os`, `strconv`, `time`). No config library.

2. **Logger**
   - Build a `*zap.Logger` (production JSON config) at the configured level. Map the
     `LOG_LEVEL` config value to a `zapcore.Level`. Provide a small constructor (e.g.
     a `logging` helper). Inject the logger; avoid global state where practical, and
     ensure `logger.Sync()` is deferred on shutdown.
   - Add `go.uber.org/zap` to the module and run `go mod tidy` (this introduces the
     first external dependency + `go.sum`).

3. **HTTP layer (`internal/http`)**
   - Build a router using the stdlib `net/http.ServeMux` (Go 1.22+ method+path
     patterns) — no third-party router needed for this scope.
   - Middleware: `RequestID` (generate/propagate an ID), `RequestLogger` (log method,
     path, status, duration), `Recover` (catch panics → 500 + log).
   - Handlers: `GET /health` and `GET /ready` returning JSON (set
     `Content-Type: application/json`). A tiny JSON-write helper is acceptable.
   - Expose a constructor like `NewRouter(logger, ...) http.Handler`.

4. **Server wiring (`cmd/api/main.go`)**
   - `config.Load()` → build logger → `http.NewRouter(...)` → `&http.Server{Addr,
     Handler, ReadTimeout, WriteTimeout}`.
   - Start in a goroutine; listen for SIGINT/SIGTERM; on signal, `server.Shutdown(ctx)`
     with `ShutdownTimeout`. Log startup (with port) and shutdown.

5. **Test**
   - In `internal/http`, a test using `httptest` (or `net/http/httptest.NewServer` /
     `ResponseRecorder`) that calls `/health` and asserts status 200 and body. Add a
     `/ready` assertion if convenient.

6. **Verify** — `go build/vet/test ./...`; run the server and `curl` both endpoints.

---

## Testing Strategy

- **Unit/handler test:** Use `httptest.ResponseRecorder` against the router from
  `NewRouter` to assert `/health` returns 200 and the expected JSON. This is the
  required automated test for this item.
- **Config test (recommended):** table-driven test for `Load()` covering defaults and
  at least one invalid value (bad port, bad log level) → error.
- **Manual smoke:** run `go run ./cmd/api`, `curl` `/health` and `/ready`, and verify
  graceful shutdown logs on Ctrl-C.
- No DB, so no integration test against external services yet (that begins in Item
  003).

---

## Dependencies

- **Upstream:** Item 001 (module + package layout) — ✅ complete.
- **Downstream (enables):** Item 003 (DB + migrations; adds DB check to `/ready`),
  Item 004 (docker-compose uses this server + `/health`), and all later API work.
- **Toolchain:** Go 1.25+ (already verified). New external dependency:
  `go.uber.org/zap` (logging) — adds `go.sum` to the module. Run `go mod tidy` after
  adding it. Router/config remain stdlib.

---

## Testing Prerequisites

### Required Services
None. This item runs the API process standalone with no external dependencies (no DB
until Item 003).

### Environment Configuration
- **Env vars (all optional, have defaults):**
  - `PORT` (default `8080`)
  - `LOG_LEVEL` (default `info`; one of debug|info|warn|error)
  - `APP_ENV` (default `development`)
- **Secrets:** None.
- **Config files:** None required.
- **Ports:** `8080` (or `PORT`) must be free locally.

### Manual Validation Checklist
- [ ] **Build succeeds:** `cd backend && go build ./...`
- [ ] **Vet passes:** `go vet ./...`
- [ ] **Tests pass:** `go test ./...`
- [ ] **Application runs:** `go run ./cmd/api` (logs a JSON startup line with the port)
- [ ] **Health check passes:** `curl -i localhost:8080/health` → `200` + `{"status":"ok"}`
- [ ] **Ready check passes:** `curl -i localhost:8080/ready` → `200` + readiness JSON
- [ ] **Logging verified:** requests produce structured JSON logs (method, path,
      status, duration, request id)
- [ ] **Graceful shutdown:** Ctrl-C drains and logs a clean shutdown

### Expected Outcomes
- `GET /health` → HTTP **200**, body `{"status":"ok"}` (or equivalent documented
  shape), `Content-Type: application/json`.
- `GET /ready` → HTTP **200** with a readiness JSON body (no dependency checks yet).
- Startup emits a structured JSON log line including the listen port; each request
  emits a structured log line; SIGINT/SIGTERM triggers a graceful shutdown log.
- `go test ./...` passes including the new `/health` handler test.

### Validation Results
- [x] Service started: N/A (no external services) — API process started: ✅ logs `server starting` on port 8080
- [x] Application started successfully: ✅
- [x] Database tables verified: N/A
- [x] Seed data verified: N/A
- [x] API endpoints verified: /health ✅ 200 `{"status":"ok"}`, /ready ✅ 200 `{"status":"ready"}`
- [x] Screenshots captured: N/A (no UI)
- [x] `go build ./...`: ✅   `go vet ./...`: ✅   `go test ./...`: ✅ (config + http packages pass)
- [x] Structured JSON logs verified (method/path/status/duration/request_id); `X-Request-Id` header echoed
- [x] Graceful shutdown on SIGTERM verified (process exits cleanly)

---

## Decisions & Trade-offs

Resolved during implementation:

- **Router = `github.com/gin-gonic/gin` v1.12.0** (post-completion change, requested
  by the user, superseding the original stdlib `net/http.ServeMux` decision).
  `NewRouter` now returns `*gin.Engine`, which implements `http.Handler`, so
  `cmd/api`'s `http.Server` and graceful-shutdown wiring are unchanged. Built with
  `gin.New()` (not `gin.Default()`) so Gin's own logger/recovery are **not** used —
  zap stays the logging path. Gin release mode is enabled when `APP_ENV=production`
  (`gin.SetMode`); tests force `gin.TestMode` via `TestMain`. Middleware are now
  `gin.HandlerFunc`s registered via `engine.Use(recoverer, requestID, requestLogger)`.
  Trade-off: pulls in a sizeable transitive dependency tree vs. stdlib, in exchange
  for Gin's routing/binding/validation ergonomics for the upcoming domain endpoints.
  Test note: Gin sets `Content-Type: application/json; charset=utf-8`, so the
  content-type assertion checks a prefix rather than an exact match. `cmd/api` keeps
  the `apihttp` import alias.
- **Logging = `go.uber.org/zap` v1.28.0** (JSON encoder) — chosen by the user over
  stdlib `slog`. First external dependency; `go.sum` now present. Built via
  `zap.NewProductionConfig()` with `LOG_LEVEL` mapped to `zapcore.Level`;
  `logger.Sync()` deferred in `main`. Tests use `zap.NewNop()`.
- **Logger location** — placed the constructor in a **new `internal/logging`
  package** (not in the Item 001 layout). Rationale: keeps logger construction out of
  both `config` (pure, dependency-free) and `http`, and gives a single injection
  point. Layout addition documented here.
- **Config = hand-rolled env loader** (stdlib `os`/`strconv`/`time`). Defaults: port
  8080, log level info, env development, 10s shutdown, 15s read/write timeouts.
  Added a `Config.Addr()` helper. Revisit a config lib when the surface grows.
- **`/ready` semantics** — returns 200 unconditionally with `{"status":"ready"}`;
  Item 003 will extend it to check DB connectivity (handler has a comment marking
  this).
- **JSON body shapes** — `/health` → `{"status":"ok"}`, `/ready` →
  `{"status":"ready"}`; both `Content-Type: application/json`.
- **Request ID** — generated with `crypto/rand` (16 bytes hex), honoring an inbound
  `X-Request-ID` header and echoing it on the response. No external UUID dependency.
- **Middleware order** — `recoverer → requestID → requestLogger → mux` (recover
  outermost so panics in any layer are caught; logger inside requestID so logs carry
  the ID).

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move the **Stage 0** deliverables this item satisfies from 📋 → ✅:
  - "Go API service skeleton (router, config, structured logging, graceful shutdown)"
  - "`GET /health` and `/ready` endpoints"
- Keep Stage 0 row at 🚧 (Items 003–006 remain).
- Only update rows corresponding to **Item 002**; do not tick deliverables owned by
  other items even if incidentally satisfied.
- Record final decisions (router, logger, config, JSON shapes) above.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 002` to implement
this work item.
