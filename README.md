# opencycle

An **offline-first**, **privacy-first** cycle tracking application.

All data stays on your device. There is no central server, no telemetry, and no mandatory cloud services.

---

## Architecture

opencycle is structured as a monorepo with the following major layers:

### Proto contract (`proto/`)
Protobuf definitions are the **canonical contract** and single source of truth for all data models and service interfaces. Generated code lives in `gen/` and must never be edited by hand.

### Go domain engine (`engine/`)
All business logic, domain rules, cycle predictions, and data persistence are implemented here. This is a local domain engine—not a backend service. It runs on-device via a mobile wrapper or local process.

### TypeScript UI (`ui/`)
The user interface layer. It must **not** implement domain or business logic; it delegates everything to the Go engine via generated bindings. UI code should be purely presentational.

### Native mobile wrappers (`mobile/`)
Thin wrappers for iOS and Android that host the Go engine and surface the TypeScript UI via a WebView. No domain logic lives here.

---

## Proto files

Proto schema files will be added in a subsequent step. The `proto/opencycle/v1/` directory is reserved for them.

---

## Quick start

```bash
# Lint everything
make lint

# Run all tests
make test

# Full CI validation
make ci
```

See [AGENT.md](AGENT.md) for canonical commands and architecture rules.
