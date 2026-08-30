---
name: senshi-test
description: Run the appropriate validation level for Senshi while avoiding unnecessary expensive test runs.
---

# Senshi Test Strategy

Choose the cheapest validation appropriate for the change.

## Focused

For small frontend changes:

- run related Angular tests
- npm run build

Do not run backend or Playwright.

## Backend

For backend-specific changes:

- run tests for affected Go package first
- run go test ./... only when broader regression is justified

## Full regression

Only when explicitly requested:

- go test ./...
- npm test -- --run
- npm run build
- isolated Playwright E2E

Do not repeatedly run the full suite after every small edit.
