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

## UI-to-Engine Bridge

The React UI runs inside a WebView hosted by the native mobile wrapper. To communicate with the Go engine (which runs as a native library in the same process), the app uses a localhost-only Connect-RPC HTTP listener as an in-process IPC mechanism.

### How it works

1. The native mobile wrapper starts the Go engine in-process.
2. The engine binds a Connect-RPC HTTP listener to `127.0.0.1` on a random port.
3. The React UI (rendered in a WebView) makes standard Connect-RPC HTTP calls to this localhost listener.
4. Responses flow back through the same HTTP path.

This means the UI uses the same generated Connect-RPC client code regardless of whether it is running in development or in a production mobile build.

### "No server" clarification

When project documentation says "no server," it means no remote or cloud server. The localhost listener is an in-process IPC mechanism that never leaves the device. It is not a deployed service and accepts no connections from the network.

### Security

In production, the listener is secured by:

- Binding to `127.0.0.1` only (not `0.0.0.0`), so it is unreachable from the network.
- Using a random port, so other apps cannot predict the endpoint.
- Using an auth token passed from the native shell to the WebView, so other apps on the device cannot access the listener.

### Development mode

In development, `engine/cmd/engine-dev/main.go` plays the role of the native shell. It starts the same localhost Connect-RPC listener so the UI can be developed and tested in a browser without a mobile device or emulator.

## Key Constraints

| Constraint | Rationale |
|------------|-----------|
| No central server | Privacy-first; data stays on device |
| No telemetry | User data must not leave the device |
| Proto is canonical | Prevents drift between layers |
| Generated code not edited | Ensures regeneration is always safe |
| Domain logic in Go only | Single implementation, avoids divergence |
