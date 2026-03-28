# TODO — openmenses

This file tracks implementation tasks. Items are grouped by phase and step.
Check off tasks as they are completed.

Phases 1–6 are complete (domain model, core logging, cycle detection, predictions, insights, UI expansion).

---

## Phase 7: iOS Native Shell

Phase 7 builds a thin iOS native shell that hosts the Go engine as an in-process library (via `gomobile bind`) and renders the existing React UI in a WkWebView. The Go engine serves **both** the Connect-RPC API and the bundled UI static files from a single `http://127.0.0.1:<random_port>` listener, so the WebView loads everything from the same origin (avoiding CORS issues with `file://` URLs). HealthKit menstrual flow integration follows as a second sub-phase after the shell is working.

This phase spans Go, TypeScript, and Swift. It does **not** modify the proto schema, generated code, or existing engine internals. It adds a new Go package, edits one UI file, and creates a new Xcode project.

### Background / Key Decisions

- **gomobile bind** produces an `.xcframework` containing a native iOS binary of the Go engine. The native shell links it and calls exported Go functions directly — no subprocess, no FFI marshalling. See https://go.dev/wiki/Mobile#sdk-applications-and-generating-bindings.
- **gomobile restrictions**: Exported functions can only use primitives (`int`, `string`, `bool`, `error`, `[]byte`). No `context.Context`, `http.Handler`, interfaces, channels, or complex structs. The bridge package wraps the existing `engine.Engine` API into gomobile-safe functions.
- **Static file serving from Go**: The Go HTTP mux serves both the Connect-RPC handler (at its service path) and a static file server (at `/`) for the bundled `ui/dist/` output. A SPA fallback ensures client-side routes resolve to `index.html`.
- **Auth token**: A 32-byte random hex token is generated at engine startup and injected into the WebView via JavaScript (`window.__OPENMENSES_ENGINE__`). All Connect-RPC requests must include `Authorization: Bearer <token>`. Static file requests are unauthenticated (they are the UI itself).
- **iOS 16+ minimum deployment target**: Provides modern WkWebView APIs, HealthKit improvements, and covers 95%+ of active devices.
- **UIKit, not SwiftUI**: WkWebView hosting is simpler and better documented in UIKit. No storyboards beyond LaunchScreen (programmatic layout).
- **No external Swift dependencies**: No CocoaPods, SPM packages, or Carthage for the MVP shell. HealthKit is a system framework.
- **Domain logic stays in Go**: Per `AGENT.md` and `PROJECT_PLAN.md`, the native shell contains zero domain logic. HealthKit integration (Phase 7D) reads/writes HealthKit data and funnels it through the Connect-RPC API.
- **iCloud backup is automatic**: All persistent data is in a SQLite file in the app's Documents directory. iOS backs this up to iCloud without any custom integration code.

### Reference Files

Read these before implementing:

| File                                              | Purpose                                                                      |
| ------------------------------------------------- | ---------------------------------------------------------------------------- |
| `AGENT.md`                                        | Mandatory agent workflow, architecture boundaries, things agents must not do |
| `docs/architecture.md`                            | Full architecture diagram, bridge design, security model                     |
| `docs/decisions/0004-localhost-connect-bridge.md` | ADR for the localhost Connect-RPC bridge approach                            |
| `docs/decisions/0005-framework7-react.md`         | ADR for Framework7 React (adaptive iOS/Material styling)                     |
| `engine/pkg/openmenses/engine.go`                 | Public `Engine` API: `NewEngine()`, `Handler()`, `Close()`                   |
| `engine/cmd/engine-dev/main.go`                   | Reference implementation of HTTP server setup (mux, listener, CORS)          |
| `ui/src/lib/client.ts`                            | Current Connect-RPC client — will be modified for dynamic baseUrl/auth       |
| `PROJECT_PLAN.md`                                 | Phase definitions, data model philosophy, native shell responsibilities      |

---

## Sub-Phase 7A: Go Mobile Bindings ✓

`engine/mobile/bridge.go` and `engine/mobile/bridge_test.go` are complete. The bridge package implements `Start`/`Stop`/`Port`/`AuthToken`, auth middleware, and SPA static file serving. All tests pass. `make mobile-setup` and `make mobile-ios` verified working on macOS.

---

## Sub-Phase 7C: Xcode Project & iOS Shell

Steps 8–11 (Xcode project, EngineManager, WebViewController, AppDelegate/SceneDelegate) are complete. The iOS shell is built with xcodegen (`project.yml`), links `Engine.xcframework` as a static framework, copies UI assets via a post-build script, and injects engine config (`port`, `authToken`) into the WKWebView at document start. The bridge API uses separate `Port()`/`AuthToken()` getters (in `engine/mobile/bridge.go`) to avoid gomobile multi-return complexity. `.gitignore` covers `Engine.xcframework/` and `ui/`. All engine tests pass; `go vet` and `gofmt` are clean. `NSAppTransportSecurity` uses `NSAllowsLocalNetworking` (not `NSAllowsArbitraryLoads`).

---

## Sub-Phase 7D: HealthKit Integration

Add bi-directional sync of menstrual flow data between the app and Apple HealthKit. All HealthKit code lives in the native Swift shell — **no domain logic in Swift**. The shell reads/writes HealthKit and translates to/from Connect-RPC calls to the engine.

### Step 15: Sync orchestration

Wire up the HealthKit sync triggers.

**On app launch** (in `AppDelegate.didFinishLaunchingWithOptions`, after engine starts):

1. Request HealthKit authorization (if not already granted)
2. Fetch menstrual flow samples since last sync
3. Import each into the engine via Connect-RPC

**On new bleeding observation** (requires native↔WebView communication):

1. WebView notifies native layer when a bleeding observation is created
2. Native layer reads the observation from the engine (or receives it via message)
3. Native layer writes it to HealthKit

**User settings toggle**:

1. Add a toggle in the Settings page: "Sync with Apple Health"
2. Store preference in `UserDefaults`
3. Only run sync when enabled

- [ ] Add post-launch sync logic to `AppDelegate` or `SceneDelegate`
- [ ] Store/read last sync date in `UserDefaults`
- [ ] Add HealthKit sync toggle to settings (requires WebView↔native message)
- [ ] Test import: add menstrual flow in Health app → launch OpenMenses → verify observation appears
- [ ] Test export: log bleeding in OpenMenses → verify it appears in Health app

### Step 16: WebView ↔ Native messaging for HealthKit

Add a message channel so the UI can trigger HealthKit operations and receive results.

**Native side** (in `WebViewController.swift`):

```swift
// Add WKScriptMessageHandler for "healthkit" messages
config.userContentController.add(self, name: "healthkit")

// Handle messages
func userContentController(_ controller: WKUserContentController,
                          didReceive message: WKScriptMessage) {
    guard let body = message.body as? [String: Any],
          let action = body["action"] as? String else { return }

    switch action {
    case "import":
        Task { await importFromHealthKit() }
    case "requestAuth":
        Task { await requestHealthKitAuth() }
    default:
        break
    }
}
```

**UI side** (new utility in `ui/src/lib/`):

```typescript
// Check if running in native iOS shell with HealthKit support
export function isHealthKitAvailable(): boolean {
  return (
    "webkit" in window &&
    "messageHandlers" in (window as any).webkit &&
    "healthkit" in (window as any).webkit.messageHandlers
  );
}

// Request HealthKit authorization
export function requestHealthKitAuth(): void {
  (window as any).webkit.messageHandlers.healthkit.postMessage({
    action: "requestAuth",
  });
}

// Trigger HealthKit import
export function importFromHealthKit(): void {
  (window as any).webkit.messageHandlers.healthkit.postMessage({
    action: "import",
  });
}
```

- [ ] Add `WKScriptMessageHandler` to `WebViewController` for "healthkit" messages
- [ ] Create `ui/src/lib/healthkit.ts` with native messaging helpers
- [ ] Add "Import from Health" button to Settings page (visible only when `isHealthKitAvailable()`)
- [ ] Run `make ui-lint` — must pass
- [ ] Run `make ui-test` — must pass
- [ ] Test on physical device: tap "Import from Health" → HealthKit permission prompt → data imported

---

## Implementation Order

```
7A: Step 1 (bridge.go) → Step 2 (auth middleware) → Step 3 (SPA file server) → Step 4 (tests) → Step 5 (Makefile)
7B: Step 6 (global.d.ts) → Step 7 (client.ts changes) — can run PARALLEL with 7A
7C: Step 8 (Xcode project) → Step 9 (EngineManager) → Step 10 (WebViewController) → Step 11 (AppDelegate) → Step 12 (e2e verification) — depends on 7A + 7B
7D: Step 13 (entitlement) → Step 14 (HealthKitManager) → Step 15 (sync) → Step 16 (WebView messaging) — depends on 7C
```

Sub-phases 7A and 7B are fully independent and can be worked on in parallel.
Sub-phase 7C requires 7A (xcframework) and 7B (UI supports dynamic URL).
Sub-phase 7D requires 7C (working iOS shell).

---

## Key Reusable Code

- `engine/pkg/openmenses/engine.go` — `NewEngine()`, `Handler()`, `Close()` — the bridge wraps these
- `engine/cmd/engine-dev/main.go` — reference HTTP server setup, CORS middleware pattern
- `ui/src/lib/client.ts` — Connect-RPC client, modified in Step 7 for dynamic URL/auth
- `gen/go/openmenses/v1/openmensesv1connect/` — Connect-RPC generated handler/client code (used by bridge for path prefix)
- `ui/dist/` — Vite production build output, served by Go static file server

## Out of Scope (future phases)

- **Android shell** (Phase 8) — Kotlin + WebView + Google Fit, mirrors iOS architecture
- **Local notifications** — period/PMS reminders via UNUserNotificationCenter
- **App Store submission** — provisioning, signing, metadata, review
- **TestFlight distribution** — beta testing pipeline
- **Automated iOS tests** (XCTest) — manual verification for MVP
- **CI for iOS builds** — requires macOS runner (GitHub Actions or similar)
- **Background App Refresh** — periodic HealthKit sync when app is backgrounded
- **Siri Shortcuts / Widgets** — quick logging from home screen
