# engine — Go Domain Engine

This directory contains the **Go domain engine** for opencycle.

## What lives here

| Path | Purpose |
|------|---------|
| `cmd/engine-dev/` | Development entry point (local testing only) |
| `internal/service/` | Service layer — orchestrates domain operations |
| `internal/validation/` | Input and domain object validation |
| `internal/storage/` | Storage abstractions (SQLite, in-memory, migrations) |
| `internal/rules/` | Cycle and health domain rules |
| `internal/predictions/` | Cycle prediction algorithms |
| `internal/insights/` | Derived insights and summaries |
| `internal/timeline/` | Timeline construction and queries |
| `pkg/opencycle/` | Public API surface exposed to consumers (mobile wrappers, tests) |
| `tests/` | Integration and cross-package tests |

## Architecture rules

- **This is a domain engine, not a backend service.** Do not add HTTP servers or network listeners.
- All business logic belongs here. The UI layer must not duplicate it.
- Storage implementations are local-only (SQLite on-device, in-memory for tests).
- Generated protobuf Go code lives in `../gen/go/` — do not duplicate it here.

## Commands

```bash
# From the repo root:
make engine-lint
make engine-test

# Or directly:
cd engine && go test ./...
cd engine && golangci-lint run ./...
```
