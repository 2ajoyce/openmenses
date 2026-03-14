# ADR 0004 — Localhost Connect-RPC Bridge

**Date:** 2026-03-13
**Status:** Accepted

## Context

The React UI runs in a WebView hosted by the native mobile wrapper. The Go engine runs as a native library in the same process. A communication mechanism is needed between these two layers that works identically on iOS and Android without platform-specific code.

## Decision

Use a localhost-only Connect-RPC HTTP listener as the bridge between the WebView and the Go engine.

On startup, the native shell initializes the Go engine and binds a Connect-RPC HTTP listener to `127.0.0.1` on a random port. The WebView makes standard Connect-RPC HTTP calls to this listener. In production, an auth token is passed from the native shell to the WebView so that other apps on the device cannot access the listener.

In development, `engine/cmd/engine-dev/main.go` starts the same listener so the UI can be developed in a browser without a mobile device.

## Consequences

**Positive:**
- Platform-agnostic: the same HTTP-based bridge works on both iOS and Android with no platform-specific code.
- Reuses generated Connect-RPC client and server bindings from the proto schema, avoiding custom serialization or bridging code.
- Simple development workflow: the UI can be developed and tested in a desktop browser against a local Go process.
- Standard tooling: HTTP traffic can be inspected with standard debugging tools.

**Negative:**
- The localhost listener must be secured (bound to `127.0.0.1`, random port, auth token) to prevent other apps on the device from accessing it.
- Adds a small amount of HTTP serialization overhead compared to a direct in-process function call, though this is negligible for the expected request volume.

## Alternatives Considered

- **WebView JavaScript bridge (`window.webkit.messageHandlers` / `@JavascriptInterface`):** Rejected because the bridge API differs between iOS and Android, requiring platform-specific serialization and dispatch code on both sides.
- **Go compiled to WebAssembly (WASM):** Rejected because running the full Go engine in WASM adds significant complexity (threading limitations, larger bundle size, SQLite compatibility issues) without meaningful benefits for this use case.
- **Protobuf-over-WebView-bridge (serialize protobuf bytes through the native JS bridge):** Rejected because it still requires platform-specific bridge plumbing and custom code to route messages to the correct RPC handler, duplicating what Connect-RPC already provides.
