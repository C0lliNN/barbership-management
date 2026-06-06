# Item 005: Next.js PWA Skeleton (SSR + pt-BR Base)

**Stage:** 0 — Foundation & Scaffolding
**Status:** ✅ Complete
**Queue:** `docs/aide/queue/queue-001.md`
**Date created:** 2026-05-31

---

## Description

Replace the `frontend/server.js` stub (created in Item 004 to satisfy the Docker
Compose dependency) with a real **Next.js application**: TypeScript, App Router,
SSR, an installable PWA manifest + service worker, a pt-BR locale foundation via
`next-intl`, BRL/timezone format helpers, Tailwind CSS, and a simple pt-BR landing
page that calls the Go API's `/health` endpoint on the server side.

After this item:
- `docker compose up` serves the real Next.js app on port 3000.
- `GET http://localhost:3000` returns a server-rendered pt-BR landing page showing
  the API health status.
- `GET http://localhost:3000/api/health` returns `{"status":"ok"}` (satisfies the
  docker-compose healthcheck).
- Lighthouse PWA installability audit passes in Chrome DevTools.
- A BRL formatter and a pt-BR date/timezone formatter are available as shared
  helpers for all future frontend work.
- The `next-intl` infrastructure (message files, request config, layout provider)
  is in place for all future pt-BR strings.

> **Dependency note:** This item fully replaces `frontend/server.js` and
> `frontend/Dockerfile`. Item 006 (CI Pipeline) depends on this item being complete
> so it can build and lint the Next.js app.

---

## Acceptance Criteria

- [ ] `docker compose up --build -d` brings up the full stack and `docker compose ps`
      shows all three services **healthy**.
- [ ] `GET http://localhost:3000` returns 200 with a server-rendered HTML page in
      `lang="pt-BR"` that includes the API health status (e.g., "API online" or
      "API offline").
- [ ] `GET http://localhost:3000/api/health` returns `200 {"status":"ok"}`.
- [ ] `GET http://localhost:3000/manifest.webmanifest` returns the PWA manifest with
      `"display":"standalone"`, `"lang":"pt-BR"`, and two icon entries (192 × 192 and
      512 × 512).
- [ ] Chrome DevTools → Lighthouse → PWA audit passes **installability** (no blocking
      failures; manifest valid, service worker registered, icons present).
- [ ] `src/lib/format.ts` exports `formatBRL(amount: number): string` (uses
      `Intl.NumberFormat` with `pt-BR` / `BRL`) and `formatDateBR(date: Date, options?):
      string` (uses `Intl.DateTimeFormat` with `America/Sao_Paulo`).
- [ ] `messages/pt-BR.json` contains at least the strings used by the landing page;
      the layout wraps the app in `<NextIntlClientProvider>`.
- [ ] `npm run build` completes without errors in the `frontend/` directory.
- [ ] `npm run type-check` exits 0 (no TypeScript errors).
- [ ] `npm run lint` exits 0 (ESLint / next/core-web-vitals rules clean).
- [ ] `make frontend-local` (from `backend/`) starts the dev server at
      `http://localhost:3000` without Docker.
- [ ] The old `frontend/server.js` stub is deleted.

---

## Implementation Steps

### 1. Delete the stub and initialize the Next.js project

Delete `frontend/server.js` (the Node.js placeholder).

Bootstrap a Next.js 14+ project with TypeScript inside `frontend/`:

```bash
cd frontend
npx create-next-app@latest . \
  --typescript \
  --tailwind \
  --eslint \
  --app \
  --src-dir \
  --no-import-alias
```

`create-next-app` sets up: `package.json`, `tsconfig.json`, `next.config.ts` (or
`.mjs`), `tailwind.config.ts`, `postcss.config.mjs`, `src/app/layout.tsx`,
`src/app/page.tsx`, `src/app/globals.css`, and base ESLint config. Accept all
defaults except import alias (disable it).

Add the additional dependencies:

```bash
npm install @ducanh2912/next-pwa next-intl
```

Add a `type-check` script to `package.json`:

```json
"scripts": {
  "dev": "next dev",
  "build": "next build",
  "start": "next start",
  "lint": "next lint",
  "type-check": "tsc --noEmit"
}
```

### 2. Configure `next.config.ts`

Chain `next-intl` and `@ducanh2912/next-pwa`:

```typescript
import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';
import withPWA from '@ducanh2912/next-pwa';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const nextConfig: NextConfig = {
  output: 'standalone',
};

export default withNextIntl(
  withPWA({ dest: 'public', disable: process.env.NODE_ENV === 'development' })(
    nextConfig
  )
);
```

`output: 'standalone'` is required for the multi-stage Docker build.
`disable: ... 'development'` prevents the service worker from running locally
during `npm run dev` (avoids stale cache confusion in development).

### 3. Set up `next-intl` (pt-BR, no URL routing)

This project has a single locale (pt-BR) and does **not** add a locale segment to
the URL. No middleware is required for locale routing.

**`src/i18n/request.ts`:**

```typescript
import { getRequestConfig } from 'next-intl/server';

export default getRequestConfig(async () => {
  const locale = 'pt-BR';
  return {
    locale,
    messages: (await import(`../../messages/${locale}.json`)).default,
  };
});
```

**`messages/pt-BR.json`:**

```json
{
  "landing": {
    "title": "Barbearia",
    "subtitle": "Plataforma de gerenciamento para barbearias",
    "apiStatus": {
      "online": "API online",
      "offline": "API offline",
      "error": "Erro ao verificar API"
    }
  },
  "common": {
    "loading": "Carregando...",
    "error": "Ocorreu um erro"
  }
}
```

**`src/app/layout.tsx`** — wrap the app in `NextIntlClientProvider`:

```typescript
import type { Metadata } from 'next';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import './globals.css';

export const metadata: Metadata = {
  title: 'Barbearia',
  description: 'Plataforma de gerenciamento para barbearias',
  manifest: '/manifest.webmanifest',
  themeColor: '#1a1a1a',
  viewport: { width: 'device-width', initialScale: 1 },
  appleWebApp: { capable: true, statusBarStyle: 'default', title: 'Barbearia' },
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const messages = await getMessages();
  return (
    <html lang="pt-BR">
      <body className="min-h-screen bg-gray-50 text-gray-900 antialiased">
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
```

### 4. Create the PWA manifest and icons

**`public/manifest.webmanifest`:**

```json
{
  "name": "Barbearia",
  "short_name": "Barbearia",
  "description": "Plataforma de gerenciamento para barbearias",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#1a1a1a",
  "orientation": "portrait",
  "lang": "pt-BR",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png",
      "purpose": "any maskable"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png",
      "purpose": "any maskable"
    }
  ]
}
```

**Icons (`public/icons/icon-192.png`, `public/icons/icon-512.png`):**

Use `sharp` to generate placeholder icons. Run this script once during setup
(then commit the resulting PNGs — do not ship `sharp` as a runtime dependency):

```bash
# From frontend/
npm install --save-dev sharp
node scripts/generate-icons.mjs
```

**`frontend/scripts/generate-icons.mjs`:**

```js
import sharp from 'sharp';
import { mkdirSync } from 'fs';

mkdirSync('public/icons', { recursive: true });

const sizes = [192, 512];
for (const size of sizes) {
  await sharp({
    create: {
      width: size,
      height: size,
      channels: 4,
      background: { r: 26, g: 26, b: 26, alpha: 1 }, // #1a1a1a
    },
  })
    .png()
    .toFile(`public/icons/icon-${size}.png`);
  console.log(`Created public/icons/icon-${size}.png`);
}
```

Commit `public/icons/icon-192.png` and `public/icons/icon-512.png`. Remove
`sharp` from `devDependencies` after running (or leave it — CI won't use it
again since the PNGs are committed).

### 5. Create the BRL/timezone format helpers

**`src/lib/format.ts`:**

```typescript
export function formatBRL(amount: number): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(amount);
}

export function formatDateBR(
  date: Date,
  options: Intl.DateTimeFormatOptions = {
    dateStyle: 'short',
    timeStyle: 'short',
  }
): string {
  return new Intl.DateTimeFormat('pt-BR', {
    timeZone: 'America/Sao_Paulo',
    ...options,
  }).format(date);
}
```

No external date library is needed; the native `Intl` API covers BRL and
America/Sao_Paulo correctly on Node 20+.

### 6. Create the `/api/health` route

The docker-compose healthcheck hits `http://127.0.0.1:3000/health`. A Next.js
`/api/health` route satisfies this when the healthcheck is updated.

**`src/app/api/health/route.ts`:**

```typescript
export function GET() {
  return Response.json({ status: 'ok' });
}
```

> **Healthcheck URL update:** The docker-compose.yml currently checks
> `http://127.0.0.1:3000/health`. Next.js does not mount routes at `/health`
> — the route is at `/api/health`. Update the `frontend` service healthcheck
> in `docker-compose.yml` to `/api/health`. Likewise update the
> `HEALTHCHECK` in `frontend/Dockerfile`.

### 7. Build the pt-BR landing page

The landing page is a **React Server Component** (async function). It fetches the
Go API `/health` on the server side during SSR and displays the result in pt-BR.

**`src/app/page.tsx`:**

```typescript
import { getTranslations } from 'next-intl/server';

async function getApiHealth(): Promise<'online' | 'offline' | 'error'> {
  const apiUrl =
    process.env.API_INTERNAL_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    'http://localhost:8080';
  try {
    const res = await fetch(`${apiUrl}/health`, {
      next: { revalidate: 10 },
    });
    return res.ok ? 'online' : 'offline';
  } catch {
    return 'error';
  }
}

export default async function HomePage() {
  const t = await getTranslations('landing');
  const health = await getApiHealth();

  const statusColor =
    health === 'online'
      ? 'text-green-600'
      : health === 'offline'
        ? 'text-red-600'
        : 'text-yellow-600';

  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-6">
      <div className="w-full max-w-sm rounded-2xl bg-white p-8 text-center shadow-md">
        <div className="mb-4 text-5xl">💈</div>
        <h1 className="mb-2 text-2xl font-bold">{t('title')}</h1>
        <p className="mb-6 text-gray-500">{t('subtitle')}</p>
        <span className={`rounded-full bg-gray-100 px-4 py-1.5 text-sm font-medium ${statusColor}`}>
          {t(`apiStatus.${health}`)}
        </span>
      </div>
    </main>
  );
}
```

### 8. Update `frontend/Dockerfile`

Replace the stub Dockerfile with a multi-stage Next.js production build.
The `output: 'standalone'` config generates a self-contained server in
`.next/standalone/` that does not require `node_modules` at runtime.

```dockerfile
# ── Stage 1: Install dependencies ────────────────────────────────────────────
FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

# ── Stage 2: Build the Next.js app ───────────────────────────────────────────
FROM node:20-alpine AS builder
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# ── Stage 3: Minimal production image ────────────────────────────────────────
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
 && adduser  --system --uid 1001 nextjs

# Copy the standalone bundle (includes server.js and node_modules subset)
COPY --from=builder /app/public                        ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static     ./.next/static

USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

# next build --standalone generates .next/standalone/server.js
CMD ["node", "server.js"]
```

### 9. Update `frontend/.dockerignore`

Replace the stub `.dockerignore` with a Next.js-appropriate one:

```
node_modules
.next
.env*
*.md
scripts/
```

### 10. Update `docker-compose.yml`

Two changes to the `frontend` service:

1. **Fix the healthcheck URL** — change from `/health` to `/api/health`.
2. **Add `API_INTERNAL_URL`** for the server-side SSR fetch (separate from the
   client-facing `NEXT_PUBLIC_API_URL`).

```yaml
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      PORT: "3000"
      API_INTERNAL_URL: http://api:8080         # server-to-server (SSR fetch)
      NEXT_PUBLIC_API_URL: http://localhost:8080 # client-side (browser, future use)
    depends_on:
      api:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -qO- http://127.0.0.1:3000/api/health || exit 1"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 30s
    restart: unless-stopped
```

> `start_period` is bumped from 10 s to 30 s — Next.js standalone takes a few
> extra seconds to start compared to the Node.js stub.

### 11. Update `backend/Makefile` — `frontend-local` target

The `frontend-local` target currently runs `node ../frontend/server.js` (the stub).
Update it to start the Next.js dev server:

```makefile
# Run the Next.js frontend in dev mode locally without Docker (requires Node 20+).
# Useful alongside 'make dev-local' when iterating on both API and frontend.
frontend-local:
	cd ../frontend && NEXT_PUBLIC_API_URL=http://localhost:8080 \
	    API_INTERNAL_URL=http://localhost:8080 npm run dev
```

Also update the Makefile header comment from "Run the frontend stub locally" to
"Run the Next.js frontend in dev mode locally".

### 12. Create `frontend/.env.local.example`

Document the variables developers need when running outside Docker:

```env
# Copy to .env.local and adjust if needed.
# Used by `npm run dev` (make frontend-local).

# API URL as seen from the browser (client-side, public)
NEXT_PUBLIC_API_URL=http://localhost:8080

# API URL as seen from the Next.js server process (SSR, server-side only)
API_INTERNAL_URL=http://localhost:8080
```

---

## Testing Strategy

| Layer | Tool | When |
|-------|------|------|
| TypeScript | `npm run type-check` | Always |
| ESLint | `npm run lint` | Always |
| Next.js build | `npm run build` | Always |
| Docker Compose full stack | `docker compose up --build -d` | Manual |
| Health endpoint | `curl localhost:3000/api/health` | Manual |
| Landing page render | Browser — `http://localhost:3000` | Manual |
| PWA installability | Chrome DevTools → Lighthouse → PWA | Manual |
| Manifest valid | `curl localhost:3000/manifest.webmanifest` | Manual |
| Service worker registered | DevTools → Application → Service Workers | Manual |

No Jest/Vitest unit tests in this item — that infrastructure belongs to Item 006
(CI Pipeline). The key verification is the Lighthouse PWA audit.

---

## Dependencies

- **Upstream:** Item 004 ✅ — provides `frontend/` directory (stub), `docker-compose.yml`
  with the `frontend` service already wired, and `backend/Makefile` with
  `frontend-local` target.
- **Downstream (enables):**
  - Item 006 (CI Pipeline): must build and lint the Next.js app; depends on this item.
  - All future frontend items: inherit the `next-intl` + Tailwind + format-helper
    foundation established here.
- **New packages introduced (confirmed with user):**
  - `@ducanh2912/next-pwa` — PWA / service-worker generation
  - `next-intl` — SSR-friendly i18n for App Router
  - `tailwindcss` + `@tailwindcss/postcss` + `postcss` — utility CSS
  - `sharp` (dev-only, for icon generation script) — can be removed after icons are generated and committed

---

## Testing Prerequisites

### Required Services

| Service | Version | Start Command | Port |
|---------|---------|---------------|------|
| Docker Engine | 24+ | (desktop app / daemon) | — |
| Node.js | 20+ | (installed on host) | — |
| Go API + PostgreSQL | via Docker | `docker compose up -d postgres api` | 5432, 8080 |

For the full Docker stack: `docker compose up --build -d` (from repo root or
`cd backend && make dev`).

For local dev server without Docker:
1. `make dev-local` (from `backend/`) — starts postgres + API locally.
2. `make frontend-local` (from `backend/`) — starts Next.js dev server.

### Environment Configuration

**In Docker (docker-compose.yml provides all vars):**
No manual configuration needed.

**Local dev (running `npm run dev` directly):**
Copy `frontend/.env.local.example` to `frontend/.env.local`. Default values work
as-is when using `make dev-local` for the API.

**Ports that must be available:**
- `5432` — PostgreSQL
- `8080` — Go API
- `3000` — Next.js frontend

### Manual Validation Checklist

**Build verification:**
- [ ] `cd frontend && npm install` — installs all dependencies without errors
- [ ] `cd frontend && npm run type-check` — exits 0
- [ ] `cd frontend && npm run lint` — exits 0
- [ ] `cd frontend && npm run build` — completes without errors

**Docker Compose (full stack):**
- [ ] `docker compose up --build -d` — all three services start
- [ ] `docker compose ps` — all three services show **healthy** (may take ~60 s for
      the Next.js container on first build)
- [ ] `curl -i http://localhost:8080/health` → `200 {"status":"ok"}`
- [ ] `curl -i http://localhost:3000/api/health` → `200 {"status":"ok"}`
- [ ] `curl -i http://localhost:3000/` → `200` with `lang="pt-BR"` in the HTML

**Browser verification:**
- [ ] Open `http://localhost:3000` — page renders with "💈 Barbearia" heading
- [ ] Page shows "API online" (green) or "API offline" (red) based on API state
- [ ] Open DevTools → Application → Manifest — manifest loads, no errors
- [ ] Open DevTools → Application → Service Workers — service worker is registered
- [ ] Run Lighthouse audit: PWA → installability passes (no blocking failures)

**Local dev server (without Docker):**
- [ ] `make dev-local` (from `backend/`) starts postgres + API
- [ ] `make frontend-local` (from `backend/`) starts Next.js dev server at port 3000
- [ ] `http://localhost:3000` renders the landing page

### Expected Outcomes

| Check | Expected |
|-------|----------|
| `docker compose ps` | 3 services: postgres, api, frontend — all healthy |
| `GET localhost:3000/api/health` | `200 {"status":"ok"}` |
| `GET localhost:3000/manifest.webmanifest` | JSON with `"display":"standalone"`, `"lang":"pt-BR"`, 2 icon entries |
| `GET localhost:3000/` | 200 — SSR HTML with `lang="pt-BR"`, "💈 Barbearia", API status text |
| Lighthouse PWA installability | Pass (no errors) |
| `npm run build` | Exits 0, no type or lint errors |

### Validation Documentation Template

```markdown
## Validation Results
- [ ] Service started: PostgreSQL 16 (Docker)
- [ ] Service started: Go API (Docker)
- [ ] Service started: Next.js frontend (Docker, multi-stage build)
- [ ] All three services healthy in `docker compose ps`
- [ ] Application started successfully (frontend logs clean, no crashes)
- [ ] API health verified: GET /api/health → 200 {"status":"ok"}
- [ ] Landing page verified: GET / → 200, pt-BR HTML, API status shown
- [ ] PWA manifest verified: manifest.webmanifest loads, display: standalone
- [ ] Service worker registered: DevTools Application → Service Workers
- [ ] Lighthouse PWA installability: [Pass / Fail — note any issues]
- [ ] npm run build: exits 0
- [ ] npm run type-check: exits 0
- [ ] npm run lint: exits 0
- [ ] Screenshots captured: [yes / no]
```

---

## Decisions & Trade-offs

**Confirmed with user before implementation:**

- **PWA library: `@ducanh2912/next-pwa`** — maintained fork of next-pwa;
  near-zero config, Workbox-backed, App Router compatible. Service worker is
  disabled in dev mode (`NODE_ENV === 'development'`) to avoid stale-cache issues
  during iteration.

- **i18n: `next-intl`** — de facto standard for Next.js App Router with SSR.
  Single locale (pt-BR), no URL routing (no `/pt-BR/` prefix). Strings live in
  `messages/pt-BR.json`. All future items add their strings there.

- **CSS: Tailwind CSS** — utility-first, mobile-first, standard in the Next.js
  ecosystem. Configured by `create-next-app --tailwind`.

**Routine decisions (encoded here):**

- **`output: 'standalone'`** — required for the multi-stage Docker build. Generates
  a minimal `.next/standalone/server.js` and a pruned `node_modules` subset. The
  Dockerfile copies this to the runner stage, keeping the production image small.

- **`API_INTERNAL_URL` vs `NEXT_PUBLIC_API_URL`** — `NEXT_PUBLIC_*` variables in
  Next.js are baked into the client bundle at **build time**, not runtime. Inside
  Docker the build happens without the runtime env vars set, so
  `NEXT_PUBLIC_API_URL` would be undefined at build time. For server-side SSR
  fetches (React Server Components), a separate `API_INTERNAL_URL` env var is used
  — it is read at **runtime** from the container's environment. `NEXT_PUBLIC_API_URL`
  is kept for future client-side code and documented as a build-time concern.

- **Service worker disabled in dev** — `@ducanh2912/next-pwa` is configured with
  `disable: process.env.NODE_ENV === 'development'`. This means Lighthouse PWA
  audits must be run against the production build (Docker Compose or `npm run
  build && npm run start`), not the dev server.

- **Placeholder icons via `sharp` script** — a one-time `scripts/generate-icons.mjs`
  creates the 192 × 512 PNGs and commits them. `sharp` is a dev dependency only;
  the generated PNGs are committed to the repo so CI does not need to regenerate them.

- **`/api/health` instead of `/health`** — Next.js App Router mounts API routes
  under `/api/...` by convention. The docker-compose healthcheck is updated from
  `/health` to `/api/health` accordingly.

- **`revalidate: 10` on the health fetch** — the landing page re-fetches the API
  health status at most every 10 seconds on the Next.js server. This avoids
  hammering the API on every page request while keeping the status reasonably fresh.

**Implementation decisions (during execution):**

- **`next build --webpack` required** — Next.js 16 defaults to Turbopack for all
  builds (dev and production). Turbopack does not support webpack plugins, so
  `@ducanh2912/next-pwa` (which uses `workbox-webpack-plugin`) was silently skipped
  and produced no service worker. The `build` script was updated to
  `next build --webpack` to force webpack and enable service worker generation.
  The `dev` script remains unmodified (Turbopack dev server works fine since the
  service worker is disabled in development anyway).

- **`next-intl` v4 `getRequestConfig` signature** — v4 changed the callback
  parameter from no args to `{ requestLocale: Promise<string | undefined> }`.
  `src/i18n/request.ts` awaits `requestLocale` and falls back to `'pt-BR'` when
  undefined (no locale segment in URL for single-locale apps).

- **`viewport` and `themeColor` moved to separate export** — Next.js 14+ deprecated
  `themeColor` and `viewport` inside the `metadata` object. Both are now exported as
  `export const viewport: Viewport` in `src/app/layout.tsx`, separate from
  `metadata`.

- **Generated service worker files excluded from git and ESLint** — `public/sw.js`
  and `public/workbox-*.js` are build artifacts generated by `@ducanh2912/next-pwa`
  on every `npm run build`. They are added to `.gitignore` and the ESLint global
  ignores to prevent spurious lint warnings from minified Workbox internals.

---

## Completion Reminder

When this item is complete, update `docs/aide/progress.md`:
- Change **"Next.js PWA skeleton (SSR, manifest, service worker, pt-BR locale base)"**
  from 📋 → ✅ under Stage 0 deliverables.
- Stage 0 remains 🚧 (Item 006 — CI pipeline — is still pending).

---

## Next Step

Start a **new chat session** and run `/speckit.aide.execute-item 005` to implement
this work item.
