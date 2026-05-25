# Item 004: Docker Compose Local Dev Environment (Full Stack)

**Stage:** 0 — Foundation & Scaffolding
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-24

---

## Description

Complete the local development environment so that a single `docker compose up`
brings up **all three services** — PostgreSQL, the Go API, and the frontend — with
proper health checks, dependency ordering, and env-var wiring.

Item 003's scope expansion already delivered the `docker-compose.yml` for
Postgres + API and the `backend/Makefile` dev targets. This item extends that
foundation by:

1. **Adding an `api` service health check** to `docker-compose.yml` so that
   `depends_on: condition: service_healthy` works for downstream services.
2. **Creating a minimal frontend stub** (`frontend/server.js` + `frontend/Dockerfile`)
   — a tiny Node.js HTTP server (stdlib only, zero npm deps) that serves a pt-BR
   placeholder page on port 3000. This satisfies "docker compose up yields a served
   frontend" immediately, and is replaced by the real Next.js app in Item 005.
3. **Adding the `frontend` service** to `docker-compose.yml` with `depends_on: api`
   (healthy), port mapping, and a healthcheck.
4. **Updating the `backend/Makefile`** with frontend-aware targets and updated help text.
5. **Polishing the `README.md`** with the complete three-service local dev loop,
   environment variable reference, and troubleshooting notes.

> **Dependency note:** Item 005 (Next.js PWA Skeleton) will replace
> `frontend/server.js` and update `frontend/Dockerfile`. That hand-off is documented
> in both items so neither is blocked.

---

## Acceptance Criteria

- [x] `docker compose up --build -d` (from the repo root, or `cd backend && make dev`)
  starts all three services: **postgres**, **api**, **frontend**. ✅
- [x] `GET http://localhost:8080/health` → `200 {"status":"ok"}`. ✅
- [x] `GET http://localhost:8080/ready` → `200 {"status":"ready"}` (DB connected). ✅
- [x] `GET http://localhost:3000` → `200` with a pt-BR placeholder HTML page. ✅
- [x] `docker compose ps` shows all three services as **healthy**. ✅
- [x] `api` service does not start until `postgres` is healthy. ✅
- [x] `frontend` service does not start until `api` is healthy. ✅
- [x] `make dev` (from `backend/`) starts the full stack and prints URLs for all
  three services. ✅
- [x] `make dev-local` starts Postgres in Docker and runs the API locally; frontend
  can be run separately with `make frontend-local`. ✅
- [x] `make dev-down` stops all three services cleanly. ✅
- [x] README documents the full three-service dev loop, environment variables, and
  a brief troubleshooting section. ✅
- [x] `go build ./...`, `go vet ./...`, `APP_ENV=test go test -short ./...` (backend)
  still pass. ✅

---

## Implementation Steps

### 1. Add `api` service healthcheck to `docker-compose.yml`

The existing `api` service has no healthcheck, so other services cannot use
`condition: service_healthy` to depend on it. Add:

```yaml
  api:
    ...
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:8080/health || exit 1"]
      interval: 5s
      timeout: 5s
      retries: 12
      start_period: 15s
```

> **Why `wget`?** The `alpine`-based Go runtime image already includes `wget`
> (via `busybox`). `curl` is not installed in the slim runtime image.
> Alternatively use `nc -z localhost 8080` if `wget` is not available — verify
> during implementation which is available in the final image.

### 2. Create the frontend stub

#### `frontend/server.js`

A zero-dependency Node.js HTTP server (uses only the built-in `http` module).
Serves a minimal pt-BR HTML page on port 3000 and exposes a `/health` endpoint.

```js
// frontend/server.js — placeholder until Item 005 (Next.js PWA)
const http = require('http');

const PORT = process.env.PORT || 3000;
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

const HTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Barbearia — Em breve</title>
  <style>
    body { font-family: sans-serif; display: flex; align-items: center;
           justify-content: center; min-height: 100vh; margin: 0;
           background: #f5f5f5; }
    .card { background: #fff; border-radius: 12px; padding: 2rem 3rem;
            text-align: center; box-shadow: 0 2px 12px rgba(0,0,0,.08); }
    h1 { color: #1a1a1a; } p { color: #666; }
  </style>
</head>
<body>
  <div class="card">
    <h1>💈 Barbearia</h1>
    <p>Plataforma de gerenciamento — em breve.</p>
    <p style="font-size:.85rem;color:#999">API: ${API_URL}</p>
  </div>
</body>
</html>`;

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok' }));
    return;
  }
  res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
  res.end(HTML);
});

server.listen(PORT, () => {
  console.log(JSON.stringify({ level: 'info', msg: 'frontend stub listening', port: PORT, api: API_URL }));
});
```

#### `frontend/Dockerfile`

```dockerfile
# frontend/Dockerfile — placeholder until Item 005 (Next.js PWA)
# Item 005 will replace server.js with a Next.js app and update this file.
FROM node:20-alpine

WORKDIR /app

# No npm dependencies — server.js uses only Node built-ins.
COPY server.js .

ENV PORT=3000
EXPOSE 3000

CMD ["node", "server.js"]
```

#### `frontend/.dockerignore`

```
node_modules/
.next/
.env*
*.md
```

### 3. Add `frontend` service to `docker-compose.yml`

```yaml
  # ── Frontend (Next.js PWA — stub until Item 005) ───────────────────────────
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      PORT: "3000"
      NEXT_PUBLIC_API_URL: http://api:8080
    depends_on:
      api:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://localhost:3000/health || exit 1"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 10s
    restart: unless-stopped
```

> **`NEXT_PUBLIC_API_URL`** is pre-named for the Next.js convention (`NEXT_PUBLIC_*`
> prefix for browser-exposed env vars). The stub reads it to display in the page;
> Item 005 will use it for `fetch()` calls.
>
> Inside the Docker network, the frontend container reaches the API at
> `http://api:8080` (service DNS). Browser clients use `http://localhost:8080`
> (or a reverse proxy — out of scope for v1 dev).

### 4. Update `backend/Makefile`

Update the `dev` target to print the frontend URL too:

```makefile
dev:
	$(COMPOSE) up --build -d
	@echo ""
	@echo "  Stack running:"
	@echo "    API      → http://localhost:8080/health"
	@echo "    DB       → localhost:5432 (user: barber, db: barbershop)"
	@echo "    Frontend → http://localhost:3000"
	@echo ""
	@echo "  Logs:   make dev-logs | make frontend-logs"
	@echo "  Stop:   make dev-down"
```

Add a `frontend-logs` target:

```makefile
# Follow frontend container logs (Ctrl-C to stop).
frontend-logs:
	$(COMPOSE) logs -f frontend
```

Update the `.PHONY` list and the quick-start comment block to include
`frontend-logs`.

For `dev-local` (API runs via `go run`, not in Docker), add a note/target for
running the frontend stub outside Docker:

```makefile
# Run the frontend stub locally (no Docker). Requires Node 20+.
# Set NEXT_PUBLIC_API_URL to point at the local API.
frontend-local:
	NEXT_PUBLIC_API_URL=http://localhost:8080 node ../frontend/server.js
```

### 5. Update `README.md`

Rewrite the **Local Development** section to cover all three services. Key additions:

- **Three-service diagram** (ports at a glance)
- **Option A** (`make dev`): full stack in Docker — emphasise frontend is now included
- **Option B** (`make dev-local` + `make frontend-local`): API locally + frontend stub
- **Environment variables** table — add `NEXT_PUBLIC_API_URL`
- **Troubleshooting** subsection: common issues (port conflicts, image not rebuilding
  after `server.js` change → `make dev` re-runs `--build`, healthcheck timeout, etc.)
- **Note** that the frontend is a placeholder; Item 005 replaces it with Next.js

---

## Testing Strategy

| Layer | Tool | When |
|-------|------|------|
| Backend build/vet/test | `go build/vet/test` | Always (no changes to Go code) |
| Docker Compose up | `docker compose up --build -d` | Manual — requires Docker |
| Health endpoints | `curl` / `wget` | Manual — part of validation checklist |
| Service ordering | `docker compose ps` (all healthy) | Manual |
| Stub page renders | Browser / `curl localhost:3000` | Manual |

No new automated tests in this item. The correctness of the compose file and
startup ordering is verified manually.

---

## Dependencies

- **Upstream:** Item 001 ✅, Item 002 ✅, Item 003 ✅ (docker-compose.yml + Makefile
  base already created in Item 003's scope expansion).
- **Downstream (enables):**
  - Item 005 (Next.js PWA Skeleton): replaces `frontend/server.js` and updates
    `frontend/Dockerfile`; `docker-compose.yml` frontend service may need minor
    env-var additions.
  - Item 006 (CI Pipeline): `docker compose up` can be used in CI to spin up the
    full stack for integration tests.
- **New runtime dependencies:** None (Node.js is inside the Docker image only).
- **Infrastructure:** Docker Engine + Docker Compose plugin (already required by
  Item 003).

---

## Testing Prerequisites

### Required Services

| Service | Version | Start Command | Port |
|---------|---------|---------------|------|
| Docker Engine | 24+ | (desktop app or daemon) | — |
| Docker Compose | v2 | bundled with Docker Desktop or `docker compose` plugin | — |

All services are started by `docker compose up --build -d` — no manual DB or API
startup needed for the full-stack path.

### Environment Configuration

**Environment variables (local developer machine):**

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://barber:secret@localhost:5432/...` | Read by `make dev-local` and `make migrate-*` from `backend/.env` |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Used by `make frontend-local` |

**No new secrets are required.** The compose file uses hard-coded dev credentials
(same as Item 003).

**Ports that must be available:**
- `5432` — PostgreSQL (unchanged)
- `8080` — Go API (unchanged)
- `3000` — Frontend stub (new)

### Manual Validation Checklist

- [ ] **Build succeeds (backend):** `cd backend && go build ./...`
- [ ] **Backend tests pass:** `cd backend && APP_ENV=test go test -short ./...`
- [ ] **Full stack starts:** `cd backend && make dev` (or `docker compose up --build -d` from root)
  - All three containers appear in `docker compose ps`
  - All three show **healthy** status (may take ~30s for all health checks to pass)
- [ ] **API healthy:** `curl -i localhost:8080/health` → `200 {"status":"ok"}`
- [ ] **API ready:** `curl -i localhost:8080/ready` → `200 {"status":"ready"}`
- [ ] **Frontend stub serves:** `curl -i localhost:3000` → `200` with pt-BR HTML
- [ ] **Frontend health endpoint:** `curl -i localhost:3000/health` → `200 {"status":"ok"}`
- [ ] **Service ordering verified:** restart only the frontend container and confirm
  it waits for api to be healthy before starting:
  `docker compose stop frontend && docker compose start frontend`
- [ ] **Logs work:** `make dev-logs` (API) and `make frontend-logs` (frontend) both
  stream without errors
- [ ] **Teardown:** `make dev-down` stops all three containers cleanly; no errors
- [ ] **Rebuild after stub change:** edit `frontend/server.js` (e.g., change page text),
  `make dev` → `--build` picks up the change

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `docker compose ps` | 3 services: postgres, api, frontend — all healthy |
| `GET localhost:8080/health` | `200 {"status":"ok"}` |
| `GET localhost:8080/ready` | `200 {"status":"ready"}` |
| `GET localhost:3000` | `200` — HTML page with "💈 Barbearia" heading in pt-BR |
| `GET localhost:3000/health` | `200 {"status":"ok"}` |
| `make dev-down` | All containers stopped, exit 0 |

### Validation Results

```markdown
## Validation Results
- [x] Service started: PostgreSQL 16 (Docker) ✅
- [x] Service started: Go API (Docker, multi-stage Alpine build) ✅
- [x] Service started: Frontend stub (Docker, node:20-alpine) ✅
- [x] All three services show healthy in `docker compose ps` ✅
- [x] Application started successfully (API + frontend logs clean) ✅
- [x] API health verified: GET /health → 200 {"status":"ok"} ✅
- [x] API ready verified: GET /ready → 200 {"status":"ready"} ✅
- [x] Frontend stub verified: GET localhost:3000 → 200 pt-BR HTML (💈 Barbearia heading) ✅
- [x] Frontend health verified: GET localhost:3000/health → 200 {"status":"ok"} ✅
- [x] make dev-down: all three containers and network removed cleanly, exit 0 ✅
- [x] Backend go build/vet/test-short all pass (no Go changes in this item) ✅
- [x] Screenshots captured: N/A (no visual design review required)
```

---

## Decisions & Trade-offs

**Resolved during create-item:**

- **Frontend stub = zero-dep Node.js HTTP server (not a Next.js placeholder)** — using
  a tiny `server.js` with only Node's built-in `http` module means the Docker image
  is minimal and builds in seconds. No npm install, no lock file churn.
  Item 005 replaces this file entirely, so there's no tech debt.

- **`NEXT_PUBLIC_API_URL` env var pre-named for Next.js** — naming the env var with
  the `NEXT_PUBLIC_` prefix now (even in the stub) means Item 005 can consume it
  without changing docker-compose. Documented clearly in both files.

- **`depends_on: api: condition: service_healthy`** — requires the `api` service to
  expose a healthcheck. Adding the healthcheck to `api` is a small addition that also
  makes `docker compose ps` more informative.

- **`make frontend-local`** placed in `backend/Makefile` — keeps all dev targets in
  one place; path is relative to `backend/` (noted in Makefile comment).

**Resolved during implementation:**

- **`wget` + `127.0.0.1` not `localhost` in healthchecks** — busybox `wget` on Alpine
  resolves `localhost` to `::1` (IPv6) first, but Node.js listening on `0.0.0.0`
  only accepts IPv4. Healthchecks using `wget -qO- http://localhost:3000/health`
  failed with "Connection refused" even though the port was open. Fix: use
  `http://127.0.0.1:<port>/health` in both the `docker-compose.yml` healthcheck
  directives and the `frontend/Dockerfile` HEALTHCHECK. The same applies to the `api`
  service healthcheck (Alpine-based runtime image). Applied consistently everywhere.

- **Dockerfile HEALTHCHECK and compose healthcheck** — both are set for the frontend
  service (the compose one takes precedence when both exist). This ensures the
  healthcheck works whether the image is run via `docker compose` or standalone
  `docker run`. Both use `127.0.0.1`.

- **`server.listen(PORT, '0.0.0.0', ...)` explicit bind** — binding explicitly to
  `0.0.0.0` rather than the default (which may be `::` on some Node versions)
  ensures IPv4 is always available inside the container.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Move the Stage 0 deliverable **"`docker-compose` local dev"** from 🚧 → ✅
  (all three services now wired: API ✅, Postgres ✅, frontend stub ✅).
- Stage 0 remains 🚧 (Items 005 and 006 still pending).
- Do not tick deliverables owned by other items.

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 004` to implement
this work item.
