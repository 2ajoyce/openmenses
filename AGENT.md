# Agent Instructions — openmenses

This file provides guidance for AI coding agents working in this repository.
Read this file carefully before making any changes.

---

## 1. Project Overview

**openmenses** is an offline-first, privacy-first cycle tracking application.

- All data stays on the user's device.
- There is **no central server** and **no backend deployment**.
- There is **no telemetry, analytics, remote logging, or mandatory cloud service**.
- Privacy is non-negotiable; do not add anything that transmits user data.

---

## 2. Architecture Rules

### Layers (in dependency order)

1. **Proto** (`proto/`) — canonical contract and source of truth.
2. **Generated code** (`gen/`) — produced from proto; never edited by hand.
3. **Go domain engine** (`engine/`) — all domain logic lives here.
4. **TypeScript UI** (`ui/`) — presentation only; no domain logic.
5. **Native mobile wrappers** (`mobile/`) — thin hosting shells; no logic.

### Strict boundaries

| Rule                           | Details                                                                              |
| ------------------------------ | ------------------------------------------------------------------------------------ |
| Domain logic → Go only         | Cycle rules, predictions, insights, validation, and storage all belong in `engine/`. |
| UI → presentation only         | `ui/` must not implement business rules. It calls engine via generated bindings.     |
| Proto is canonical             | Data model and service interface changes start in `.proto` files.                    |
| No hand-editing generated code | Files under `gen/` are produced by `buf generate`; never edit them manually.         |
| Offline-first                  | No feature may require a network connection to function.                             |
| No server                      | Do not create server apps, API deployments, Dockerfiles, or cloud configs.           |
| No telemetry                   | Do not add analytics, crash reporting, or remote logging of any kind.                |

### Technology context

- **Language**: Go 1.21+
- **Transport**: gRPC / Protobuf
- **Style**: AIP-compliant (Google API Improvement Proposals)

---

## 3. Agent Workflow (Mandatory)

When asked to add a feature or generate code, follow these steps in order:

1. **Identify the Resource**: Determine the domain noun (resource) first.
2. **Define the Hierarchy**: Determine the parent-child relationship (e.g., is a `DailyLog` a child of a `Cycle`?).
3. **Choose Lifecycle Methods**: Prefer standard methods (Get, List, Create, Update, Delete) before introducing custom actions.
4. **Draft Proto**: Create or update the `.proto` file using standard methods.
5. **Validate Proto**: Check that `Update` methods use `FieldMask` and `List` methods use pagination.
6. **Generate Code**: Run `make proto-generate` to produce stubs.
7. **Implement Logic**: Write service logic in `engine/internal/`.

All Go code generation and architectural proposals must follow the rules in:

- `Design_Guidelines.md` — resource-oriented design
- `Go_Code_Guidelines.md` — Go implementation conventions

---

## 4. Agent Design Constraints

- **NO Custom Verbs**: Do not create RPCs like `CancelSubscription`. Instead, use `UpdateSubscription` that changes a `state` field to `CANCELLED`.
- **Names over IDs**: Request messages must use a `string name` field, not `int64 id`.
- **Idempotency**: Ensure `Create` and `Delete` operations are idempotent where applicable.
- **Resource-first design**: Model systems around resources (domain nouns), not actions.
- **Consistent naming**: Keep names identical across APIs, services, repositories, handlers, and workflows.
- **Declarative models**: Prefer declarative resource models for workflows.
- **Small interfaces**: Keep interfaces small, behavior-focused, and explicit.
- **Simple code**: Follow Uber Go style principles. Produce simple, readable code. Avoid unnecessary abstractions.

---

## 5. Canonical Commands

Always use these `make` targets. Do not run the underlying tools directly unless debugging.

| Target                | Purpose                                  |
| --------------------- | ---------------------------------------- |
| `make proto-lint`     | Lint proto files with buf                |
| `make proto-generate` | Regenerate code from proto with buf      |
| `make proto-breaking` | Check for breaking proto changes vs main |
| `make engine-lint`    | Lint Go engine code                      |
| `make engine-test`    | Run Go engine tests                      |
| `make ui-lint`        | Lint TypeScript UI code                  |
| `make ui-test`        | Run TypeScript UI tests                  |
| `make lint`           | Run all linters (proto + engine + ui)    |
| `make test`           | Run all tests (engine + ui)              |
| `make ci`             | Run all CI validation steps              |

---

## 6. Working on Proto

- Edit `.proto` files in `proto/openmenses/v1/`.
- Run `make proto-lint` to validate.
- Run `make proto-generate` to regenerate `gen/`.
- Commit both the `.proto` changes **and** the updated `gen/` files together.
- Never edit files under `gen/` directly.
- Check for breaking changes with `make proto-breaking` before merging.

## 7. Working on the Engine

- All business logic belongs in `engine/internal/`.
- Public API surface for consumers lives in `engine/pkg/openmenses/`.
- Run `make engine-lint` before committing.
- Run `make engine-test` to validate correctness.
- The engine is a library/domain model, not an HTTP server.

## 8. Working on the UI

- UI code lives in `ui/src/`.
- Generated TypeScript bindings come from `gen/ts/` — do not duplicate them in `ui/src/generated/`.
- Run `make ui-lint` before committing.
- Run `make ui-test` to validate.

---

## 9. Things Agents Must Not Do

- Do not add Docker, Kubernetes, Terraform, or any cloud infrastructure config.
- Do not add server-side frameworks (no Express, Gin HTTP server, etc.).
- Do not add telemetry, analytics, or remote logging libraries.
- Do not manually edit any file under `gen/`.
- Do not implement domain logic in `ui/`.
- Do not implement domain logic in `mobile/`.
- Do not add mandatory network dependencies.
