# ui — TypeScript UI

This directory contains the **TypeScript UI layer** for openmenses.

## Architecture rules

- This layer is **presentation only**. It must not implement business or domain rules.
- Domain operations are delegated to the Go engine via generated TypeScript bindings in `../gen/ts/`.
- Do not copy or re-implement logic that belongs in `engine/`.

## Directory layout

| Path | Purpose |
|------|---------|
| `src/app/` | Application shell and routing |
| `src/components/` | Reusable UI components |
| `src/features/` | Feature modules (each feature = one directory) |
| `src/pages/` | Top-level page components |
| `src/lib/` | Utility helpers and UI-layer adapters |
| `src/generated/` | Reserved for any UI-build-time generated assets |
| `public/` | Static assets |

## Commands

```bash
# From the repo root:
make ui-lint
make ui-test

# Or directly:
cd ui && npm run lint
cd ui && npm run typecheck
cd ui && npm run test
```
