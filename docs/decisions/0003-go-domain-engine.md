# ADR 0003 — Go as Domain Engine

**Date:** 2026-03-09
**Status:** Accepted

## Context

The application requires a domain layer that handles cycle rules, predictions, insights, and local storage. This layer must run on-device (iOS and Android) and must not require a network connection.

## Decision

All domain logic is implemented in Go (`engine/`). Go is compiled into a native library and linked into each mobile platform's thin wrapper. The TypeScript UI calls the engine via a bridge; it does not implement domain logic independently.

## Consequences

**Positive:**
- Single implementation of all business rules — no risk of iOS/Android/web divergence.
- Go compiles to native libraries for iOS (`xcframework`) and Android (`aar`) via [gomobile](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile).
- Strong typing and testability.
- Clear boundary: if it's domain logic, it's in Go.

**Negative:**
- Requires gomobile toolchain for mobile builds.
- Bridge layer adds some complexity vs. a pure-JS solution.
- Go developers needed for domain changes.

## Alternatives Considered

- **Domain logic in TypeScript:** Rejected — would duplicate logic across platforms and violate the single-source-of-truth principle.
- **Rust via WASM:** Considered but rejected — higher toolchain complexity with fewer benefits at this stage.
- **Native Swift/Kotlin per platform:** Rejected — would require maintaining two separate domain implementations.
