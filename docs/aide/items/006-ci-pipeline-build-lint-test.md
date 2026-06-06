# Item 006: CI Pipeline (Build, Lint, Test)

**Stage:** 0 — Foundation & Scaffolding
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-31

---

## Description

Add a GitHub Actions CI workflow that, on every push and pull request, builds,
lints, and tests both the Go backend and the Next.js frontend. This is the final
deliverable of Stage 0 — it ensures every future commit is validated automatically.

The workflow runs two independent jobs:

- **`go`** — builds the module, vets with `go vet`, lints with `staticcheck`, and
  runs the short unit-test suite (no database required).
- **`frontend`** — installs dependencies, type-checks with TypeScript, lints with
  ESLint (`npm run lint`), and builds the Next.js app (`npm run build --webpack`).

After this item, `git push` to any branch triggers CI and the Stage 0 acceptance
criterion "CI is green on a trivial test for each side" is satisfied.

**What CI does NOT do in this item:**
- Run integration tests (those require a live PostgreSQL service — deferred to a
  later item or a dedicated integration workflow).
- Run Lighthouse / PWA audits (manual step only).
- Deploy anything.

---

## Acceptance Criteria

- [ ] `.github/workflows/ci.yml` exists and is syntactically valid (`actionlint` or
      a dry-run confirms it).
- [ ] On push to `main` and on any pull-request branch, both the `go` and `frontend`
      jobs execute.
- [ ] **Go job:** `go build ./...`, `go vet ./...`, `staticcheck ./...`, and
      `go test -short ./...` all exit 0.
- [ ] **Frontend job:** `npm ci`, `npm run type-check`, `npm run lint`, and
      `npm run build` all exit 0.
- [ ] The workflow uses the correct Go version (`1.25`) and Node version (`20`).
- [ ] The workflow completes in under 5 minutes on a cold runner for both jobs
      running in parallel.
- [ ] Caches are configured so repeat runs (warm cache) are meaningfully faster
      (`actions/cache` for Go module cache; `actions/cache` for `node_modules`).
- [ ] No secrets are needed — both jobs use only the public repository code.

---

## Implementation Steps

### 1. Create the `.github/workflows/` directory structure

```
.github/
  workflows/
    ci.yml
```

The repo root is `/home/collin/Projects/barbership-management`. The `.github`
directory does not exist yet — create it along with `workflows/`.

### 2. Write `ci.yml`

```yaml
name: CI

on:
  push:
    branches: ["**"]
  pull_request:
    branches: ["**"]

jobs:
  go:
    name: Go — build / vet / lint / test
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: backend

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"
          cache: true                  # caches $GOPATH/pkg/mod automatically

      - name: Install staticcheck
        run: go install honnef.co/go/tools/cmd/staticcheck@latest

      - name: Build
        run: go build ./...

      - name: Vet
        run: go vet ./...

      - name: Staticcheck
        run: staticcheck ./...

      - name: Test (short)
        run: go test -short ./...

  frontend:
    name: Frontend — type-check / lint / build
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: frontend

    steps:
      - uses: actions/checkout@v4

      - name: Set up Node
        uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"
          cache-dependency-path: frontend/package-lock.json

      - name: Install dependencies
        run: npm ci

      - name: Type-check
        run: npm run type-check

      - name: Lint
        run: npm run lint

      - name: Build
        run: npm run build
        env:
          # Suppress telemetry noise in CI logs
          NEXT_TELEMETRY_DISABLED: "1"
          # API URL must be set at build time for Next.js; use a placeholder since
          # no real API is available during CI. The build does not make live requests
          # (SSR fetches fail gracefully in the landing page component).
          NEXT_PUBLIC_API_URL: "http://localhost:8080"
```

**Key decisions encoded in this workflow:**

- `actions/setup-go@v5` with `cache: true` handles Go module caching automatically
  using the `go.sum` as the cache key.
- `actions/setup-node@v4` with `cache: "npm"` and `cache-dependency-path` caches
  `~/.npm` based on `package-lock.json`, making `npm ci` fast on subsequent runs.
- `staticcheck` is installed at the latest version compatible with Go 1.25 via
  `go install`. This is simpler than pinning a binary download and keeps the install
  consistent with local developer tooling.
- `NEXT_TELEMETRY_DISABLED: "1"` prevents Next.js from sending telemetry during CI,
  keeping logs clean.
- `NEXT_PUBLIC_API_URL` is set to a placeholder because Next.js bakes
  `NEXT_PUBLIC_*` values into the client bundle at build time; the build will fail
  without it. The landing page's SSR fetch (`getApiHealth`) catches errors gracefully
  and returns `'error'` — no real API call is made during the build phase.
- `working-directory` defaults are set per-job so every `run:` step executes from
  `backend/` or `frontend/` without manual `cd`.
- Both jobs run in parallel (`go` and `frontend` have no declared dependency on
  each other), so total CI time is the max of the two, not the sum.

### 3. Verify `npm run lint` uses the right command

The frontend's `package.json` defines:
```json
"lint": "eslint"
```
This runs ESLint using `eslint.config.mjs` (flat config). The CI step calls this
directly — no extra flags needed; ESLint will pick up the project root config.

### 4. Verify `npm run build` command

The frontend's `build` script is:
```json
"build": "next build --webpack"
```
The `--webpack` flag is required because `@ducanh2912/next-pwa` uses
`workbox-webpack-plugin` which is incompatible with Turbopack (the Next.js 16
default). CI will use this script as-is.

### 5. No integration-test job in this item

The integration tests in `backend/internal/database/pool_test.go` require a live
PostgreSQL connection and are tagged with `//go:build integration`. They are run
with `go test -tags integration ./...`. This item uses `go test -short ./...`
which skips any test calling `t.Skip()` for short mode and does not include
integration-tagged files. A dedicated integration CI job (with a PostgreSQL
service container) is a later concern.

---

## Testing Strategy

| Layer | How Verified |
|-------|-------------|
| Workflow syntax | Push to a branch and observe GitHub Actions UI; or run `actionlint` locally |
| Go build | `go build ./...` in `backend/` exits 0 |
| Go vet | `go vet ./...` in `backend/` exits 0 |
| staticcheck | `staticcheck ./...` in `backend/` exits 0 (install first: `go install honnef.co/go/tools/cmd/staticcheck@latest`) |
| Go unit tests | `go test -short ./...` in `backend/` exits 0 |
| Frontend type-check | `cd frontend && npm run type-check` exits 0 |
| Frontend lint | `cd frontend && npm run lint` exits 0 |
| Frontend build | `cd frontend && NEXT_PUBLIC_API_URL=http://localhost:8080 NEXT_TELEMETRY_DISABLED=1 npm run build` exits 0 |
| Full CI run | Push to a branch; verify both jobs go green in GitHub Actions |

---

## Dependencies

- **Upstream:**
  - Item 002 ✅ — provides the Go unit tests (`router_test.go`, `config_test.go`)
    that CI must pass.
  - Item 005 ✅ — provides the Next.js app with working `build`, `lint`, and
    `type-check` scripts.
- **Downstream (enables):**
  - All future items: CI validates every push from here on.
  - Stage 0 completion: this is the last planned deliverable of Stage 0.
- **External:** GitHub Actions (free tier for public repos; included for private
  repos up to usage limits). No new Go or Node libraries are introduced.

---

## Testing Prerequisites

### Required Services

None — both CI jobs run entirely without external services. The Go short tests use
only in-process mocks (see `router_test.go`); the frontend build does not make live
API calls.

For local pre-flight verification before pushing:

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.25+ | already installed (used in Items 001–004) |
| Node | 20+ | already installed (used in Item 005) |
| staticcheck | latest | `go install honnef.co/go/tools/cmd/staticcheck@latest` |

### Environment Configuration

**CI (GitHub Actions):** No secrets or environment variables required. The
`NEXT_PUBLIC_API_URL` placeholder is set inline in `ci.yml`.

**Local pre-flight:** No `.env` files required. Run the commands below from the
repo root with the `backend/` and `frontend/` working directories as noted.

### Manual Validation Checklist

**Local pre-flight (run before pushing to trigger CI):**

- [ ] `cd backend && go build ./...` — exits 0
- [ ] `cd backend && go vet ./...` — exits 0
- [ ] `cd backend && staticcheck ./...` — exits 0
- [ ] `cd backend && go test -short ./...` — exits 0, all tests pass
- [ ] `cd frontend && npm run type-check` — exits 0
- [ ] `cd frontend && npm run lint` — exits 0
- [ ] `cd frontend && NEXT_PUBLIC_API_URL=http://localhost:8080 NEXT_TELEMETRY_DISABLED=1 npm run build` — exits 0

**GitHub Actions verification:**

- [ ] Push a commit to a branch (or open a PR)
- [ ] Navigate to the repository's **Actions** tab
- [ ] Confirm the `CI` workflow triggered
- [ ] Confirm both `go` and `frontend` jobs are **green** (✅)
- [ ] Confirm both jobs ran in parallel (start times overlap in the timeline view)
- [ ] On a second push (warm cache), confirm the runs are faster

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `go build ./...` | exits 0, no output |
| `go vet ./...` | exits 0, no output |
| `staticcheck ./...` | exits 0, no findings |
| `go test -short ./...` | exits 0, all test functions in `router_test.go` and `config_test.go` pass |
| `npm run type-check` | exits 0, "Found 0 errors." (or no output) |
| `npm run lint` | exits 0, no ESLint violations |
| `npm run build` | exits 0, `.next/` directory created, standalone output in `.next/standalone/` |
| GitHub Actions CI run | Both jobs green, total wall time < 5 min |

### Validation Documentation Template

```markdown
## Validation Results — Item 006

**Local pre-flight:**
- [ ] `go build ./...`: exits 0
- [ ] `go vet ./...`: exits 0
- [ ] `staticcheck ./...`: exits 0 (or note any findings fixed)
- [ ] `go test -short ./...`: exits 0, N tests pass (list any skipped)
- [ ] `npm run type-check`: exits 0
- [ ] `npm run lint`: exits 0
- [ ] `npm run build`: exits 0

**GitHub Actions:**
- [ ] CI workflow triggered on push
- [ ] `go` job: [Pass / Fail — duration: ~Xs]
- [ ] `frontend` job: [Pass / Fail — duration: ~Xs]
- [ ] Both jobs ran in parallel: [yes / no]
- [ ] Second run (warm cache) faster: [yes / no — duration: ~Xs]
- [ ] Actions tab URL: [paste URL]
```

---

## Decisions & Trade-offs

**Implementation decisions (2026-05-31):**

- **`go-version: "1.25"` matches `go.mod`** — the local toolchain is Go 1.26, but
  `go.mod` declares `go 1.25.0`. The CI workflow targets the declared minimum so the
  build is validated against the required minimum version, not the dev machine version.
- **`staticcheck@latest` resolved to v0.7.0** — compatible with Go 1.25; installation
  via `go install` verified locally before CI creation, confirming the approach works.
- **Local pre-flight all passed before creating the workflow** — `go build`, `go vet`,
  `staticcheck`, `go test -short`, `npm run type-check`, `npm run lint`, and
  `npm run build` (with `NEXT_PUBLIC_API_URL` placeholder) all exited 0 locally.
  The workflow is committed with known-good local state.

**Pre-decided (queue spec and stack alignment):**

- **CI provider: GitHub Actions** — repository is on GitHub; Actions is the natural
  choice with no additional setup. Zero cost for this project's usage level.
- **`staticcheck` over `golangci-lint`** — the queue item specifically names
  `staticcheck`; `golangci-lint` is a meta-linter that bundles many tools including
  `staticcheck`. Using `staticcheck` directly is simpler and sufficient for Stage 0.
  `golangci-lint` can be added in a later hardening item if needed.
- **Short tests only (no integration job)** — integration tests require PostgreSQL.
  Adding a service container in CI is feasible (GitHub Actions supports it natively)
  but is deferred to keep this item focused on completing Stage 0. The integration
  tests will be excluded from this workflow by relying on the `-short` flag and
  build tags (`//go:build integration`).
- **`actions/setup-go@v5` with built-in cache** — simpler than a manual
  `actions/cache` step; the v5 action handles Go module and build caching
  automatically using `go.sum` as the cache key.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Change **"CI pipeline: build, lint, test for Go and frontend"** from 📋 → ✅
  under Stage 0 deliverables.
- Update Stage 0 **Status** from 🚧 to ✅ (all deliverables done).
- Update Stage 0 **Acceptance Criteria** — tick the "CI is green on a trivial test
  for each side" checkbox.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 006` to implement
this work item.
