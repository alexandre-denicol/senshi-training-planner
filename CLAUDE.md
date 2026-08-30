# Senshi Training Planner

## Stack
- Backend: Go + PostgreSQL
- Frontend: Angular 21 + PrimeNG
- UI language: pt-BR
- Dark navy/cobalt visual identity
- Frontend dev port: 4200
- Normal backend: 18080

## Domain
Category -> Block -> Workout -> Schedule -> History

Blocks are generic and may contain:
- optional free-text description
- optional ordered free-text sequence

Do not create a martial-arts technique catalog.

History is an immutable snapshot. Never read live catalog data to represent completed training.

## Authorization
ADMIN and PROFESSOR can operate:
- Categories
- Blocks
- Workouts
- Agenda
- completion
- History

Only professor-account administration is ADMIN-only.

## Scope rules
- Preserve existing architecture and business rules.
- Prefer existing patterns over new abstractions.
- Do not add dependencies unless necessary.
- Do not redesign approved UI without explicit request.
- Do not modify database schema during UI polish.
- Do not add Students/attendance.
- Do not git add/commit/push/merge unless explicitly requested.
- Never reset/restore/stash unrelated work.

## Testing strategy
For small UI increments:
- run only focused Angular tests
- run `npm run build`

Do NOT run full Go + Angular + Playwright after every small change.

Full regression is performed only when explicitly requested.

## E2E
Isolated E2E environment:
- PostgreSQL: localhost:55432
- E2E backend: 18081
- normal backend 18080 must not be touched
- unrelated Docker service on 8080 must not be touched
- mutable Playwright must never target normal Supabase

## Working style
- Inspect only files relevant to the requested scope.
- Do not recursively audit the whole repository unless explicitly requested.
- Make the smallest coherent change.
- Do not invent features.
- Keep final reports concise.