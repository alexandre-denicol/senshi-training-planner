---
name: senshi-ui-polish
description: Polish a specific Senshi frontend area without changing domain behavior.
---

# Senshi UI Polish

Use this skill only for UI/UX polish.

## Rules

- Inspect only files relevant to the requested screen.
- Preserve existing business behavior.
- Preserve the approved dark navy/cobalt identity.
- Do not change database schema.
- Do not change backend unless explicitly required.
- Do not add new features.
- Do not redesign unrelated screens.

Focus on concrete problems with:
- visual hierarchy
- spacing
- forms
- action hierarchy
- feedback
- empty/loading/error states
- responsive behavior
- accessibility
- pt-BR copy

Prefer existing PrimeNG components and project patterns.

## Validation

For a small UI increment:
- run focused Angular tests for changed components
- run `npm run build`

Do not run the complete Playwright or Go suite unless explicitly requested.

## Git

Never add, commit, push, merge, reset, restore, or stash unless explicitly requested.

	Keep the final report concise.
