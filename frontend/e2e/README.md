# Playwright E2E

The E2E suite runs the real Angular app in Chromium and uses the backend through the existing `/api` proxy.

## Prerequisites

- Backend running locally on `http://localhost:18080`.
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
```

Optional:

```bash
E2E_BASE_URL=http://127.0.0.1:4200
E2E_FRONTEND_HOST=127.0.0.1
E2E_FRONTEND_PORT=4200
```

## Run

```bash
cd frontend
npm run e2e
npm run e2e:ui
```

The Playwright config starts Angular with the existing proxy. Start the Go backend separately:

```bash
cd backend
set -a
source ../.env
set +a
APP_ENV=development PORT=18080 go run ./cmd/server
```

Screenshots/videos are retained on failure. HTML reports and traces are written under Playwright's default report/test-results directories, which are ignored by Git.
