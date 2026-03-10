# ADR 0001 — Monorepo

**Date:** 2026-03-09
**Status:** Accepted

## Context

openmenses has several distinct components: a proto contract, a Go domain engine, a TypeScript UI, and native mobile wrappers. These components share interfaces and generated code, and need to evolve together.

## Decision

Use a single Git repository (monorepo) to host all components.

## Consequences

**Positive:**
- Atomic commits that span proto + engine + UI + generated code.
- Single place for shared tooling, CI, and documentation.
- Easier to enforce architecture boundaries and code review policies.
- Generated code changes are visible alongside the proto changes that caused them.

**Negative:**
- All contributors work in the same repo (acceptable at this scale).
- Build tools must be aware of which components changed (mitigated by targeted Makefile targets and CI jobs).

## Alternatives Considered

- **Multiple repos:** Rejected because cross-component changes require coordinated multi-repo PRs, which adds friction and risks drift between the proto contract and its consumers.
