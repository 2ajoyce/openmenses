# Agent Instructions — opencycle

This file provides guidance for AI coding agents working in this repository.
Read this file carefully before making any changes.

---

## Project Overview

**opencycle** is an offline-first, privacy-first cycle tracking application.

- All data stays on the user's device.
- There is **no central server** and **no backend deployment**.
- There is **no telemetry, analytics, remote logging, or mandatory cloud service**.
- Privacy is non-negotiable; do not add anything that transmits user data.

---

## Architecture Rules

### Layers (in dependency order)

1. **Proto** (`proto/`) — canonical contract and source of truth.
2. **Generated code** (`gen/`) — produced from proto; never edited by hand.
3. **Go domain engine** (`engine/`) — all domain logic lives here.
4. **TypeScript UI** (`ui/`) — presentation only; no domain logic.
5. **Native mobile wrappers** (`mobile/`) — thin hosting shells; no logic.

### Strict boundaries

| Rule | Details |
|------|---------|
| Domain logic → Go only | Cycle rules, predictions, insights, validation, and storage all belong in `engine/`. |
| UI → presentation only | `ui/` must not implement business rules. It calls engine via generated bindings. |
| Proto is canonical | Data model and service interface changes start in `.proto` files. |
| No hand-editing generated code | Files under `gen/` are produced by `buf generate`; never edit them manually. |
| Offline-first | No feature may require a network connection to function. |
| No server | Do not create server apps, API deployments, Dockerfiles, or cloud configs. |
| No telemetry | Do not add analytics, crash reporting, or remote logging of any kind. |

---

## Canonical Commands

Always use these `make` targets. Do not run the underlying tools directly unless debugging.

| Target | Purpose |
|--------|---------|
| `make proto-lint` | Lint proto files with buf |
| `make proto-generate` | Regenerate code from proto with buf |
| `make proto-breaking` | Check for breaking proto changes vs main |
| `make engine-lint` | Lint Go engine code |
| `make engine-test` | Run Go engine tests |
| `make ui-lint` | Lint TypeScript UI code |
| `make ui-test` | Run TypeScript UI tests |
| `make lint` | Run all linters (proto + engine + ui) |
| `make test` | Run all tests (engine + ui) |
| `make ci` | Run all CI validation steps |

---

## Working on Proto

- Edit `.proto` files in `proto/opencycle/v1/`.
- Run `make proto-lint` to validate.
- Run `make proto-generate` to regenerate `gen/`.
- Commit both the `.proto` changes **and** the updated `gen/` files together.
- Never edit files under `gen/` directly.
- Check for breaking changes with `make proto-breaking` before merging.

## Working on the Engine

- All business logic belongs in `engine/internal/`.
- Public API surface for consumers lives in `engine/pkg/opencycle/`.
- Run `make engine-lint` before committing.
- Run `make engine-test` to validate correctness.
- The engine is a library/domain model, not an HTTP server.

## Working on the UI

- UI code lives in `ui/src/`.
- Generated TypeScript bindings come from `gen/ts/` — do not duplicate them in `ui/src/generated/`.
- Run `make ui-lint` before committing.
- Run `make ui-test` to validate.

---

## Things Agents Must Not Do

- Do not add Docker, Kubernetes, Terraform, or any cloud infrastructure config.
- Do not add server-side frameworks (no Express, Gin HTTP server, etc.).
- Do not add telemetry, analytics, or remote logging libraries.
- Do not manually edit any file under `gen/`.
- Do not implement domain logic in `ui/`.
- Do not implement domain logic in `mobile/`.
- Do not add mandatory network dependencies.
