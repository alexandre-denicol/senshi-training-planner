# Senshi Training Planner

## Architecture

This repository is a monorepo with two main applications:

- `backend/`: Go HTTP API.
- `frontend/`: Angular web application.

## Current stack

- Backend: Go.
- Frontend: Angular 21.
- UI: PrimeNG and PrimeIcons.
- Database: PostgreSQL hosted on Supabase.
- Backend deployment: Render.
- Frontend deployment: Vercel.

## Domain

Core flow:

```text
Category -> Block -> Workout -> Schedule -> History
```

The application also manages students and professor accounts.

History represents an immutable snapshot of completed training and must not depend on later catalog changes.

## Development rules

- Keep changes small, coherent, and reviewable.
- Prefer existing patterns before introducing new abstractions.
- Do not add dependencies unless needed.
- Keep the UI in pt-BR.
- Preserve the responsive dark-mode visual identity.
- PostgreSQL must be accessed only by the Go backend.
- The frontend must never access Supabase or PostgreSQL directly.
- Never commit real credentials, secrets, passwords, session tokens, or password hashes.
- Never log `DATABASE_URL`, credentials, session tokens, password hashes, or secrets.
- Authentication uses opaque server-side sessions.
- Do not introduce JWT unless the architecture is intentionally changed.
- Authentication secrets must not be stored in browser `localStorage` or `sessionStorage`.
- Session cookies must remain `HttpOnly`.
- Server-side session invalidation is mandatory.
- Passwords must use Argon2id.
- Authentication failures must not disclose whether an account exists.
- Professor account administration is ADMIN-only.
- Professor-management endpoints must never modify ADMIN accounts.
- Disabling a professor or resetting a professor password must invalidate that professor's active sessions.
- Mutable E2E tests must use an isolated test database.
