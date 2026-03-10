# Architecture — openmenses

## Overview

openmenses is a monorepo structured around a clear set of layers with strict boundaries.
The application runs entirely on-device. There is no central server.

```
┌─────────────────────────────────────┐
│           Mobile Wrapper            │  iOS / Android (thin native shell)
│  hosts Go engine + renders UI       │
└───────────────┬─────────────────────┘
                │
        ┌───────┴───────┐
        │               │
┌───────▼──────┐  ┌─────▼──────────┐
│  Go Engine   │  │  TypeScript UI  │
│  (domain)    │  │  (presentation) │
│  engine/     │  │  ui/            │
└───────┬──────┘  └─────────────────┘
        │ uses generated bindings from
┌───────▼──────────────────────────┐
│      gen/ (generated code)        │
│  produced by: buf generate        │
└───────┬──────────────────────────┘
        │ generated from
┌───────▼──────────────────────────┐
│      proto/ (canonical contract)  │
│  proto/openmenses/v1/*.proto       │
└──────────────────────────────────┘
```

## Layer Responsibilities

### Proto (`proto/`)
- Single source of truth for data models and service interfaces.
- Changes here flow downstream to all generated code.
- Must be backward-compatible unless a major version bump is justified.

### Generated code (`gen/`)
- Produced by `buf generate` from proto definitions.
- Contains Go and TypeScript source files.
- **Must never be edited by hand.**
- Committed to the repo so CI can verify they are up-to-date.

### Go Engine (`engine/`)
- Contains all business logic: cycle rules, predictions, insights, storage.
- Runs as a library linked into the mobile wrappers.
- No HTTP server. No network listener. No telemetry.
- Internal packages are not part of the public API; `pkg/openmenses/` is.

### TypeScript UI (`ui/`)
- Presentation layer only.
- Calls the Go engine (via generated bindings and/or a bridge) for all data operations.
- Does not implement domain rules.

### Mobile Wrappers (`mobile/`)
- Thin native shells for iOS and Android.
- Host the Go engine binary and render the TypeScript UI in a WebView.
- No domain logic.

## Data Flow

```
User action → UI → Go Engine (via bridge) → local storage (SQLite)
                ↑                         ↓
                └────── updated state ────┘
```

## Key Constraints

| Constraint | Rationale |
|------------|-----------|
| No central server | Privacy-first; data stays on device |
| No telemetry | User data must not leave the device |
| Proto is canonical | Prevents drift between layers |
| Generated code not edited | Ensures regeneration is always safe |
| Domain logic in Go only | Single implementation, avoids divergence |
