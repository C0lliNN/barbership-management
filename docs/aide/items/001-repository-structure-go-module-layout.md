# Item 001: Repository Structure & Go Module Layout

**Stage:** 0 — Foundation & Scaffolding
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-24

---

## Description

Establish the foundational repository structure for the Barbershop Management
Platform. This is the very first implementation item — it creates the skeleton that
every later item builds on. It includes initializing the Go module, laying out the
backend package structure to reflect the vision's domain boundaries, reserving a
`frontend/` location for the Next.js PWA (scaffolded later in Item 005), and adding
base repo hygiene files.

**No business logic, no HTTP server, no database.** Those arrive in Items 002–003.
The deliverable here is a clean, conventional, *buildable* skeleton: `go build ./...`
and `go vet ./...` succeed, packages exist as stubs, and the layout communicates the
intended architecture.

### Domain boundaries to represent (from vision §5.2)
Identity/Tenancy, Catalog, Scheduling/Availability, Booking, Subscription, Payment —
plus shared infrastructure (config, database, HTTP/transport).

---

## Proposed Layout

```
.
├── backend/
│   ├── go.mod                      # module: github.com/<owner>/barbershop (confirm path)
│   ├── go.sum
│   ├── cmd/
│   │   └── api/
│   │       └── main.go             # entrypoint stub (prints/exits; real server in Item 002)
│   └── internal/
│       ├── config/                 # config loading (stub)
│       ├── database/               # db connection (stub; implemented Item 003)
│       ├── http/                   # router/transport (stub; implemented Item 002)
│       ├── identity/               # tenancy, users, roles, auth (Stage 1)
│       ├── catalog/                # services, staff availability (Stage 2)
│       ├── scheduling/             # availability engine (Stage 3)
│       ├── booking/                # bookings + state machine (Stage 3)
│       ├── subscription/           # plans, recurring entitlement (Stage 6)
│       └── payment/                # Mercado Pago integration (Stage 5)
├── frontend/                       # placeholder dir + .gitkeep (Next.js in Item 005)
├── docs/                           # (existing) aide planning docs
├── .gitignore
├── .editorconfig
└── README.md
```

Each `internal/<domain>/` package starts with a single `doc.go` (package
declaration + one-line package comment) so it compiles and documents intent. No types
or functions beyond what is needed to compile.

> **Decision needed:** the Go module path (e.g.
> `github.com/<github-owner>/barbershop`). See Decisions & Trade-offs. Until the git
> remote/owner is known, a sensible placeholder will be used and noted.

---

## Acceptance Criteria

- [x] `backend/go.mod` exists with a Go version pinned (go 1.25.0) and a module path
      (`github.com/gcollin65/barbershop`).
- [x] The directory layout above exists; each `internal/<domain>/` package has a
      `doc.go` with a package comment.
- [x] `backend/cmd/api/main.go` exists as a minimal compilable entrypoint.
- [x] From `backend/`, `go build ./...` succeeds with no errors.
- [x] From `backend/`, `go vet ./...` succeeds with no errors.
- [x] `frontend/` exists (with `.gitkeep`) as a placeholder for Item 005.
- [x] Root `.gitignore` covers Go build artifacts, env files, and Node/Next.js
      artifacts; `.editorconfig` sets sane defaults.
- [x] `README.md` documents the layout and a short "how to build the backend" note.
- [x] No business logic, HTTP server, or DB code is introduced (those are Items
      002–003).

---

## Implementation Steps

1. **Confirm module path** — determine the Go module path from the intended git
   remote/owner. If unknown, use a documented placeholder (see Decisions).
2. **Init module** — create `backend/`, run `go mod init <module-path>`, pin Go
   version in `go.mod`.
3. **Create entrypoint** — `backend/cmd/api/main.go` with a minimal `main()` (e.g.,
   logs a startup line and exits 0, or prints a placeholder message). Keep it
   buildable and free of unused imports.
4. **Create infra package stubs** — `internal/config`, `internal/database`,
   `internal/http`, each with a `doc.go` package comment describing future
   responsibility.
5. **Create domain package stubs** — `internal/{identity,catalog,scheduling,booking,
   subscription,payment}`, each with a `doc.go`.
6. **Frontend placeholder** — create `frontend/.gitkeep`.
7. **Repo hygiene** — add root `.gitignore` (Go: `*.test`, `/bin`, build output;
   env: `.env`, `*.local`; Node: `node_modules/`, `.next/`, etc.) and
   `.editorconfig`.
8. **README** — document the layout, domain boundaries, and backend build command.
9. **Verify** — run `go build ./...` and `go vet ./...` from `backend/`.

---

## Testing Strategy

This item produces scaffolding, so "testing" is primarily compilation and tooling
verification rather than behavioral unit tests:

- **Build verification:** `go build ./...` from `backend/` must succeed.
- **Static analysis:** `go vet ./...` from `backend/` must pass.
- **Structure verification:** confirm all expected directories and `doc.go` files
  exist.
- No unit tests are required for empty stub packages. (The first real tests arrive in
  Item 002 with the health endpoint.)

---

## Dependencies

- **Upstream:** None. This is the first implementation item.
- **Downstream (enables):** Item 002 (API skeleton), Item 003 (DB/migrations),
  Item 005 (frontend), and all subsequent backend work depend on this layout.
- **Toolchain:** Go 1.22+ installed locally.

---

## Testing Prerequisites

### Required Services
None. This item introduces no runtime services (no DB, no HTTP server yet).

### Environment Configuration
- **Tooling required:** Go 1.22+ (`go version`). Optionally `gofmt`/`goimports`.
- **Env vars:** None.
- **Secrets:** None.
- **Ports:** None.

### Manual Validation Checklist
- [ ] **Toolchain present:** `go version` reports Go 1.22 or newer.
- [ ] **Build succeeds:** from `backend/`, `go build ./...` exits 0.
- [ ] **Vet passes:** from `backend/`, `go vet ./...` exits 0.
- [ ] **Layout present:** `internal/{config,database,http,identity,catalog,scheduling,booking,subscription,payment}` each contain `doc.go`.
- [ ] **Entrypoint present:** `backend/cmd/api/main.go` exists and compiles.
- [ ] **Frontend placeholder:** `frontend/.gitkeep` exists.
- [ ] **Hygiene files:** root `.gitignore` and `.editorconfig` exist; `README.md`
      documents layout + build.
- [ ] **No stray business logic:** packages contain only `doc.go` (and the entrypoint).

### Expected Outcomes
- A `backend/` Go module with **10 packages** (`cmd/api` + 9 `internal/*`) that all
  compile.
- `go build ./...` and `go vet ./...` both succeed with zero output/errors.
- A `frontend/` placeholder directory tracked via `.gitkeep`.
- Root-level `.gitignore`, `.editorconfig`, and `README.md` present.

### Validation Results
- [x] Service started: N/A (no runtime services)
- [x] Application started successfully: N/A (entrypoint compiles & runs — `go run ./cmd/api` prints placeholder line, exits 0)
- [x] Database tables verified: N/A
- [x] Seed data verified: N/A
- [x] API endpoints verified: N/A
- [x] Screenshots captured: N/A (no UI)
- [x] `go build ./...` succeeds: ✅ BUILD_OK
- [x] `go vet ./...` succeeds: ✅ VET_OK
- [x] 10 packages compile: `cmd/api` + 9 `internal/*` (verified via `go list ./...`)

---

## Decisions & Trade-offs

Resolved during implementation:

- **Go module path = `github.com/gcollin65/barbershop`** — no git remote was
  configured at implementation time, so the path was derived from the owner's email
  username (`gcollin65`) as a documented placeholder. **Trade-off / future task:** if
  the real GitHub owner/repo differs, this requires a module-path rename
  (`go mod edit -module ...` + import-path updates). Cheap now (no external imports
  reference it yet); record the true remote when known.
- **Go version = 1.25.0** — `go mod init` pinned `go 1.25.0` (local toolchain is
  1.26.0; spec floor was 1.22+). README states "Go 1.25+". Comfortably above the
  spec minimum.
- **Monorepo confirmed** — `backend/` and `frontend/` side by side in one repo,
  matching the planned docker-compose dev environment (Item 004).
- **`internal/` over `pkg/`** — domain packages live under `internal/` so they are
  private to the module by default; revisit only if external reuse is needed.
- **`doc.go`-per-package** — each stub package contains a single `doc.go` with a
  package comment naming its future responsibility and the item/stage that implements
  it. Keeps the tree compiling and self-documenting with zero premature logic.
- **Migration tool / router library deliberately not chosen** — deferred to Items
  002–003 to keep this item dependency-free (`go.mod` has no requirements; no
  `go.sum` needed yet).

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move **Stage 0 → "Repository layout reflecting domain boundaries"** deliverable from
  📋 → ✅.
- If this is the first Stage 0 work started, set the Stage 0 row in the overview table
  to 🚧 In Progress (📋 → 🚧 → ✅ as the stage progresses).
- Record any final decisions (especially the module path) in the Decisions &
  Trade-offs section above.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 001` to implement
this work item.
