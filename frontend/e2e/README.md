# Playwright E2E

The E2E suite runs the real Angular app in Chromium and uses the backend through the existing `/api` proxy.

## Prerequisites

- Backend running locally on `http://localhost:18080` for safe read-only checks, or an isolated E2E backend for mutable tests.
- A dedicated E2E/test database if mutable tests will be executed.
- Real E2E users created through the normal backend flow.

Never run mutable E2E tests against production or normal user data.

## Environment

Required for authenticated tests:

```bash
E2E_ADMIN_EMAIL=
E2E_ADMIN_PASSWORD=
E2E_PROFESSOR_EMAIL=
E2E_PROFESSOR_PASSWORD=
```

Required for tests that create/update data:

```bash
E2E_ALLOW_MUTATION=true
E2E_DATABASE_URL=postgres://senshi_e2e:...@localhost:55432/senshi_e2e?sslmode=disable
E2E_API_PROXY_TARGET=http://localhost:18081
```

Optional:

```bash
E2E_FRONTEND_HOST=127.0.0.1
E2E_FRONTEND_PORT=4200
E2E_BASE_URL=http://127.0.0.1:4200
```

## Run

```bash
cd frontend
npm run e2e
npm run e2e:ui
```

The Playwright config starts Angular with the existing proxy. For normal local development, `/api` defaults to `http://localhost:18080`.

For mutable E2E, start a separate backend on a separate port and point the proxy at it:

```bash
cd backend
set -a
source ../.env.e2e.local
set +a
go run ./cmd/server
```

Then run Playwright with the same E2E environment loaded:

```bash
cd frontend
set -a
source ../.env.e2e.local
set +a
npm run e2e
```

Screenshots/videos are retained on failure. HTML reports and traces are written under Playwright's default report/test-results directories, which are ignored by Git.
