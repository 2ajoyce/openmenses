# openmenses

An offline-first, privacy-first menstrual cycle tracking app. All data stays on your device — no server, no telemetry, no cloud.

## How it works

You log bleeding observations, symptoms, moods, and medications. The Go engine detects cycles, estimates phases, generates predictions, and surfaces insights. Everything happens locally. See [docs/algorithms.md](docs/algorithms.md) for exactly how your data is processed.

## Structure

This is a monorepo with four layers:

| Layer            | Location  | What it does                                                                         |
| ---------------- | --------- | ------------------------------------------------------------------------------------ |
| Proto contract   | `proto/`  | Canonical data model and service interface. Single source of truth.                  |
| Go domain engine | `engine/` | All business logic, cycle rules, storage, predictions, and insights. Runs on-device. |
| TypeScript UI    | `ui/`     | Presentation only. Calls the engine via generated bindings — no domain logic here.   |
| Native wrappers  | `mobile/` | Thin iOS/Android shells that host the engine and serve the UI via WebView.           |

Generated code lives in `gen/` and is never edited by hand.

## Quick start

```bash
make lint      # lint everything
make test      # run all tests
make ci        # full CI: lint + generation check + tests
```

For development, see [CLAUDE.md](CLAUDE.md) for all build commands. For architecture rules and agent workflow, see [AGENT.md](AGENT.md).
