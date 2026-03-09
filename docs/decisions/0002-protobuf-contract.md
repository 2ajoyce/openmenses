# ADR 0002 — Protobuf as Canonical Contract

**Date:** 2026-03-09
**Status:** Accepted

## Context

The Go engine and TypeScript UI need a shared data model. Without a canonical source of truth, the two layers risk drifting apart as the application evolves.

## Decision

Protobuf (via [Buf](https://buf.build)) is the canonical contract for all data models and service interfaces. Generated code (Go and TypeScript) is produced from `.proto` files using `buf generate` and must never be edited by hand.

## Consequences

**Positive:**
- Single definition of every type, shared across Go and TypeScript.
- Schema evolution is explicit and checkable (buf breaking change detection).
- Generated code is always consistent with the proto definition.
- Buf lint enforces naming conventions and style.

**Negative:**
- Proto changes require a regeneration step (`make proto-generate`) and committing the generated output.
- Contributors must have `buf` installed (or rely on CI to catch issues).

## Alternatives Considered

- **TypeScript types as source of truth:** Rejected — Go cannot consume TypeScript types natively.
- **JSON Schema:** Rejected — less ergonomic for RPC-style interfaces and lacks first-class Go/TS code generation.
- **Manual type duplication:** Rejected — guaranteed to drift.
