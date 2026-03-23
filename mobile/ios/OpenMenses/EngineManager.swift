import Foundation
import Engine

/// Manages the Go engine lifecycle. Call `start()` early in app launch
/// and `stop()` on termination. Access `port` and `authToken` after
/// a successful `start()` call.
final class EngineManager {
    static let shared = EngineManager()

    private(set) var port: Int = 0
    private(set) var authToken: String = ""

    /// Base URL for the engine's HTTP server (e.g. "http://127.0.0.1:51234").
    var engineBaseURL: String {
        "http://127.0.0.1:\(port)"
    }

    private init() {}

    /// Start the Go engine with a SQLite database in the app's Documents directory.
    /// UI assets are loaded from the app bundle's "ui" subfolder (copied there
    /// by the "Copy UI Assets" build phase script).
    ///
    /// Throws an NSError if the engine fails to start.
    func start() throws {
        let documentsDir = FileManager.default.urls(
            for: .documentDirectory,
            in: .userDomainMask
        )[0]
        let dbPath = documentsDir.appendingPathComponent("openmenses.db").path

        // UI assets are copied into the app bundle under a "ui/" subfolder
        // by the "Copy UI Assets" Xcode build phase. If the folder is absent
        // (e.g. during development before running `make ui-bundle`), pass an
        // empty string so the engine still serves the RPC API without a UI.
        let uiAssetsDir: String
        if let resourcePath = Bundle.main.resourcePath {
            let candidate = resourcePath + "/ui"
            uiAssetsDir = FileManager.default.fileExists(atPath: candidate) ? candidate : ""
        } else {
            uiAssetsDir = ""
        }

        var startError: NSError?
        let ok = MobileStart(dbPath, uiAssetsDir, &startError)
        if !ok {
            throw startError ?? NSError(
                domain: "EngineManager",
                code: -1,
                userInfo: [NSLocalizedDescriptionKey: "MobileStart returned false with no error detail"]
            )
        }

        // gomobile exposes Port and AuthToken as separate getter functions
        // because gomobile restricts exported functions to a single return value
        // (plus an optional error). See engine/mobile/bridge.go for details.
        self.port = MobilePort()
        self.authToken = MobileAuthToken()
    }

    /// Gracefully stop the Go engine and release its resources. Safe to call
    /// multiple times. Errors during shutdown are logged but not propagated
    /// (the app is likely terminating).
    func stop() {
        var stopError: NSError?
        let ok = MobileStop(&stopError)
        if !ok, let error = stopError {
            NSLog("EngineManager: MobileStop error: %@", error.localizedDescription)
        }
    }
}
