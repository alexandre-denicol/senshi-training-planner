---
name: senshi-e2e
description: Safely operate the isolated Senshi Playwright E2E environment.
---

# Senshi E2E

E2E environment:

- Angular: 4200
- normal backend: 18080
- E2E backend: 18081
- E2E PostgreSQL: localhost:55432
- container: senshi-e2e-postgres

Never touch the unrelated service on port 8080.

Mutable Playwright tests must use ONLY the isolated E2E database.

Never run E2E_ALLOW_MUTATION=true against the normal Supabase database.

Expected full result:

9 passed
0 skipped
0 failed

Do not recreate the E2E environment unless necessary.
