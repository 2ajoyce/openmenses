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

## Sub-Phase 7B: UI Adaptations

Small changes to the UI's Connect-RPC client to support dynamic engine URL and auth token injection. These changes are backward-compatible — when the injected config is absent (development mode), behavior is identical to the current implementation.

### Step 6: Window type augmentation

Create a TypeScript declaration file so `window.__OPENMENSES_ENGINE__` is typed.

Create `ui/src/types/global.d.ts`:

```typescript
export {};

declare global {
  interface Window {
    /**
     * Injected by the native shell (iOS/Android) at startup.
     * Absent in browser-based development mode.
     */
    __OPENMENSES_ENGINE__?: {
      /** TCP port the Go engine's HTTP server is listening on. */
      port: number;
      /** Bearer token required for Connect-RPC requests. */
      authToken: string;
    };
  }
}
```

- [ ] Create `ui/src/types/global.d.ts` as shown above
- [ ] Verify `make ui-lint` passes (typecheck picks up the declaration)

### Step 7: Dynamic engine URL and auth token in Connect transport

Modify `ui/src/lib/client.ts` to:

1. Check `window.__OPENMENSES_ENGINE__` for port/token
2. If present (mobile): use `http://127.0.0.1:<port>` as `baseUrl` and add an interceptor that attaches `Authorization: Bearer <token>` to every request
3. If absent (dev): use `window.location.origin` as `baseUrl` with no auth (existing behavior)

Current file (`ui/src/lib/client.ts`):

```typescript
import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CycleTrackerService } from "@gen/openmenses/v1/service_pb";

const transport = createConnectTransport({
  baseUrl: window.location.origin,
});

export const client = createClient(CycleTrackerService, transport);

export const DEFAULT_PARENT = "users/default";
```

Replace with:

```typescript
import { createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CycleTrackerService } from "@gen/openmenses/v1/service_pb";

// When running inside a native shell, the engine injects its config into
// window.__OPENMENSES_ENGINE__. In dev mode this is undefined and we fall
// back to the Vite proxy at window.location.origin.
const engineConfig = window.__OPENMENSES_ENGINE__;

const baseUrl = engineConfig
  ? `http://127.0.0.1:${engineConfig.port}`
  : window.location.origin;

const interceptors: Interceptor[] = [];

if (engineConfig?.authToken) {
  interceptors.push((next) => async (req) => {
    req.header.set("Authorization", `Bearer ${engineConfig.authToken}`);
    return next(req);
  });
}

const transport = createConnectTransport({ baseUrl, interceptors });

export const client = createClient(CycleTrackerService, transport);

export const DEFAULT_PARENT = "users/default";
```

- [ ] Modify `ui/src/lib/client.ts` as shown above
- [ ] Run `make ui-lint` — must pass
- [ ] Run `make ui-test` — must pass (no behavioral change in test environment)

---

## Sub-Phase 7C: Xcode Project & iOS Shell

Create the Xcode project and Swift code for the iOS native shell. This sub-phase depends on Sub-Phase 7A (the `.xcframework` must exist) and Sub-Phase 7B (the UI must support dynamic engine URL).

### Project structure

```
mobile/ios/
├── OpenMenses.xcodeproj/           ← Xcode project file
├── OpenMenses/
│   ├── AppDelegate.swift           ← App entry point, engine lifecycle
│   ├── SceneDelegate.swift         ← Window/scene setup
│   ├── EngineManager.swift         ← Go engine lifecycle singleton
│   ├── WebViewController.swift     ← WkWebView hosting
│   ├── Info.plist                  ← App metadata, privacy descriptions
│   ├── Assets.xcassets/            ← App icon, colors
│   └── LaunchScreen.storyboard    ← Required by iOS
├── Engine.xcframework/             ← gomobile output (gitignored)
└── ui/                             ← Copied from ui/dist/ (gitignored)
```

Add to `.gitignore`:

```
mobile/ios/Engine.xcframework/
mobile/ios/ui/
```

### Step 8: Create Xcode project

Create the Xcode project manually or via `xcodegen` / `tuist`. Target configuration:

- **Product Name**: OpenMenses
- **Bundle Identifier**: org.openmenses.app (or similar)
- **Deployment Target**: iOS 16.0
- **Interface**: Storyboard (for LaunchScreen only, all other UI is programmatic)
- **Language**: Swift
- **Device**: iPhone + iPad
- **Frameworks**: Link `Engine.xcframework` (from `mobile/ios/Engine.xcframework/`)
- **Resources**: Add `mobile/ios/ui/` folder reference (the built UI assets; gitignored, copied as a build phase)

- [ ] Create Xcode project at `mobile/ios/OpenMenses.xcodeproj`
- [ ] Configure deployment target iOS 16.0
- [ ] Link `Engine.xcframework` as an embedded framework
- [ ] Add `mobile/ios/ui/` as a resource folder reference
- [ ] Add build phase script to copy UI assets: `cp -R ../../ui/dist/ ${SRCROOT}/ui/` (or use `make ui-bundle`)
- [ ] Add entries to `.gitignore` for `Engine.xcframework/` and `ui/`
- [ ] Verify project builds (even without engine — just the Swift code compiling)

### Step 9: EngineManager.swift

Singleton managing the Go engine lifecycle. Calls the exported functions from the gomobile-generated `Engine.xcframework`.

```swift
import Foundation
import Engine  // gomobile-generated framework

/// Manages the Go engine lifecycle. Call start() early in app launch
/// and stop() on termination.
final class EngineManager {
    static let shared = EngineManager()

    private(set) var port: Int = 0
    private(set) var authToken: String = ""

    /// Base URL for the engine's HTTP server.
    var engineBaseURL: String {
        "http://127.0.0.1:\(port)"
    }

    private init() {}

    /// Start the Go engine with a SQLite database in the app's Documents directory.
    /// Also sets up the static file server to serve the bundled UI assets.
    func start() throws {
        let documentsDir = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask).first!
        let dbPath = documentsDir.appendingPathComponent("openmenses.db").path

        // UI assets are bundled as a resource folder named "ui"
        let uiAssetsDir = Bundle.main.resourcePath.map { $0 + "/ui" } ?? ""

        var error: NSError?
        let result = MobileStart(dbPath, uiAssetsDir, &error)
        if let error = error {
            throw error
        }

        // MobileStart returns port via the gomobile binding
        // The exact API shape depends on gomobile's Go-to-ObjC mapping.
        // gomobile maps (int, string, error) returns into ObjC output params.
        // Consult the generated Engine framework headers for the exact signature.
        self.port = result.port   // Adapt to actual generated API
        self.authToken = result.token  // Adapt to actual generated API
    }

    func stop() {
        var error: NSError?
        MobileStop(&error)
        // Log error if non-nil, but don't crash — we're likely shutting down
    }
}
```

**Important**: gomobile maps multi-return Go functions into ObjC/Swift in a specific way. A Go function `func Start(a, b string) (int, string, error)` becomes an ObjC method with output parameters. The exact Swift signature will be visible in the generated `Engine.xcframework` headers. **Adapt the Swift code above to match the actual generated headers** — the logic is correct but the calling convention may differ.

An alternative approach: Instead of returning multiple values from `Start`, consider making the bridge API return a single struct-like string (JSON) that Swift can parse. Or split into `Start(dbPath, uiAssetsDir string) error`, `Port() int`, `AuthToken() string` — three separate exported functions. This avoids multi-return gomobile complexity.

If splitting the API, update `engine/mobile/bridge.go` to add:

```go
func Port() int       { mu.Lock(); defer mu.Unlock(); if running == nil { return 0 }; return running.ln.Addr().(*net.TCPAddr).Port }
func AuthToken() string { mu.Lock(); defer mu.Unlock(); if running == nil { return "" }; return running.token }
```

- [ ] Create `mobile/ios/OpenMenses/EngineManager.swift`
- [ ] Verify it compiles against the Engine framework
- [ ] If needed, adjust `engine/mobile/bridge.go` API to use separate getter functions instead of multi-return

### Step 10: WebViewController.swift

UIViewController hosting a WkWebView. Injects the engine config and loads the UI.

```swift
import UIKit
import WebKit

final class WebViewController: UIViewController {

    private var webView: WKWebView!

    override func viewDidLoad() {
        super.viewDidLoad()

        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true

        // Inject engine config before any page script runs
        let engineConfig = EngineManager.shared
        let script = """
        window.__OPENMENSES_ENGINE__ = {
            port: \(engineConfig.port),
            authToken: "\(engineConfig.authToken)"
        };
        """
        let userScript = WKUserScript(
            source: script,
            injectionTime: .atDocumentStart,
            forMainFrameOnly: true
        )
        config.userContentController.addUserScript(userScript)

        webView = WKWebView(frame: view.bounds, configuration: config)
        webView.autoresizingMask = [.flexibleWidth, .flexibleHeight]

        // Respect safe area (notch, home indicator)
        webView.scrollView.contentInsetAdjustmentBehavior = .automatic

        view.addSubview(webView)

        // Load the UI from the Go HTTP server
        if let url = URL(string: engineConfig.engineBaseURL) {
            webView.load(URLRequest(url: url))
        }
    }
}
```

- [ ] Create `mobile/ios/OpenMenses/WebViewController.swift`
- [ ] Verify it compiles

### Step 11: AppDelegate.swift and SceneDelegate.swift

Wire up engine lifecycle and window creation.

**AppDelegate.swift**:

```swift
import UIKit

@main
class AppDelegate: UIResponder, UIApplicationDelegate {

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        do {
            try EngineManager.shared.start()
        } catch {
            // Fatal — the app cannot function without the engine
            fatalError("Failed to start engine: \(error)")
        }
        return true
    }

    func applicationWillTerminate(_ application: UIApplication) {
        EngineManager.shared.stop()
    }

    // MARK: UISceneSession Lifecycle
    func application(
        _ application: UIApplication,
        configurationForConnecting connectingSceneSession: UISceneSession,
        options: UIScene.ConnectionOptions
    ) -> UISceneConfiguration {
        UISceneConfiguration(name: "Default Configuration", sessionRole: connectingSceneSession.role)
    }
}
```

**SceneDelegate.swift**:

```swift
import UIKit

class SceneDelegate: UIResponder, UIWindowSceneDelegate {

    var window: UIWindow?

    func scene(
        _ scene: UIScene,
        willConnectTo session: UISceneSession,
        options connectionOptions: UIScene.ConnectionOptions
    ) {
        guard let windowScene = scene as? UIWindowScene else { return }
        window = UIWindow(windowScene: windowScene)
        window?.rootViewController = WebViewController()
        window?.makeKeyAndVisible()
    }

    func sceneDidBecomeActive(_ scene: UIScene) {
        // Engine stays running in background — no restart needed
    }

    func sceneDidEnterBackground(_ scene: UIScene) {
        // Engine keeps running — SQLite handles its own locking
    }
}
```

- [ ] Create `mobile/ios/OpenMenses/AppDelegate.swift`
- [ ] Create `mobile/ios/OpenMenses/SceneDelegate.swift`
- [ ] Create `mobile/ios/OpenMenses/Info.plist` with scene configuration
- [ ] Create `mobile/ios/OpenMenses/Assets.xcassets/` with placeholder app icon
- [ ] Create `mobile/ios/OpenMenses/LaunchScreen.storyboard` (blank white screen)
- [ ] Build and run on iOS Simulator — expect: app launches, WebView loads UI, can create/view observations

### Step 12: End-to-end verification

Manual verification checklist (no automated iOS tests for MVP):

- [ ] `make ui-bundle` produces `ui/dist/` with production build
- [ ] `make mobile-ios` produces `mobile/ios/Engine.xcframework/`
- [ ] Xcode build succeeds with both artifacts linked
- [ ] iOS Simulator: app launches → blank LaunchScreen → WebView loads Framework7 UI
- [ ] Can create a user profile
- [ ] Can log a bleeding observation
- [ ] Can view the timeline
- [ ] Can navigate all tabs (Timeline, Cycles, Log, Medications, Settings)
- [ ] Can export data (JSON/CSV)
- [ ] Can view clinician summary
- [ ] App survives backgrounding and foregrounding
- [ ] App data persists across launches (SQLite in Documents)
- [ ] `make ci` still passes (no regressions from UI changes)

---

## Sub-Phase 7D: HealthKit Integration

Add bi-directional sync of menstrual flow data between the app and Apple HealthKit. All HealthKit code lives in the native Swift shell — **no domain logic in Swift**. The shell reads/writes HealthKit and translates to/from Connect-RPC calls to the engine.

### Step 13: HealthKit entitlement and permissions

Configure the Xcode project for HealthKit access.

- [ ] Add HealthKit capability to the Xcode project (Signing & Capabilities → + HealthKit)
- [ ] Add to `Info.plist`:
  - `NSHealthShareUsageDescription`: "OpenMenses reads menstrual flow data from Health to avoid duplicate logging."
  - `NSHealthUpdateUsageDescription`: "OpenMenses writes menstrual flow observations to Health so your cycle data is available across apps."
- [ ] Verify project still builds with HealthKit entitlement

### Step 14: HealthKitManager.swift

New singleton managing HealthKit operations. This class handles authorization, reading, and writing menstrual flow data.

**HealthKit data type**: `HKCategoryTypeIdentifierMenstrualFlow`

**Value mapping** between HealthKit `HKCategoryValueMenstrualFlow` and proto `BleedingFlow`:

| HealthKit Value | Proto `BleedingFlow`   |
| --------------- | ---------------------- |
| `.unspecified`  | skip (don't import)    |
| `.none`         | `BLEEDING_FLOW_NONE`   |
| `.light`        | `BLEEDING_FLOW_LIGHT`  |
| `.medium`       | `BLEEDING_FLOW_MEDIUM` |
| `.heavy`        | `BLEEDING_FLOW_HEAVY`  |

Reverse mapping (proto → HealthKit):

| Proto `BleedingFlow`        | HealthKit Value     |
| --------------------------- | ------------------- |
| `BLEEDING_FLOW_UNSPECIFIED` | skip (don't export) |
| `BLEEDING_FLOW_NONE`        | `.none`             |
| `BLEEDING_FLOW_LIGHT`       | `.light`            |
| `BLEEDING_FLOW_MEDIUM`      | `.medium`           |
| `BLEEDING_FLOW_HEAVY`       | `.heavy`            |

**Key methods**:

```swift
import HealthKit

final class HealthKitManager {
    static let shared = HealthKitManager()
    private let healthStore = HKHealthStore()

    private let menstrualFlowType = HKCategoryType(.menstrualFlow)

    /// Request read/write authorization for menstrual flow.
    func requestAuthorization() async throws { ... }

    /// Query HealthKit for menstrual flow samples since the given date.
    /// Returns samples that can be converted to BleedingObservation RPCs.
    func fetchMenstrualFlow(since: Date) async throws -> [MenstrualFlowSample] { ... }

    /// Write a menstrual flow sample to HealthKit.
    func writeMenstrualFlow(date: Date, flow: Int, isStartOfCycle: Bool) async throws { ... }
}

struct MenstrualFlowSample {
    let startDate: Date
    let endDate: Date
    let flowLevel: Int  // maps to BleedingFlow enum value
    let isStartOfCycle: Bool
}
```

**Import flow** (HealthKit → Engine):

1. Query HealthKit for `HKCategoryTypeIdentifierMenstrualFlow` samples since last sync date
2. For each sample, map `HKCategoryValueMenstrualFlow` to `BleedingFlow` enum value
3. Call `CreateBleedingObservation` Connect-RPC endpoint for each observation
4. Engine-side validation will reject duplicates (by date) — this is expected and safe
5. Store last sync date in `UserDefaults`

**Export flow** (Engine → HealthKit):

1. When a new bleeding observation is created in the app, also write it to HealthKit
2. This is triggered from the native layer, not the UI — use a notification or polling approach
3. Set `HKMetadataKeyMenstrualCycleStart` on samples where the observation corresponds to the start of a new cycle

- [ ] Create `mobile/ios/OpenMenses/HealthKitManager.swift` with the structure above
- [ ] Implement `requestAuthorization()` — request read/write for menstrual flow
- [ ] Implement `fetchMenstrualFlow(since:)` — query HealthKit, map values
- [ ] Implement `writeMenstrualFlow(date:flow:isStartOfCycle:)` — write sample to HealthKit
- [ ] Test on physical device (HealthKit is not available in Simulator by default — use Health app on device)

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
