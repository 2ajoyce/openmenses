import UIKit
import WebKit

/// Hosts a WKWebView that loads the bundled React UI from the Go engine's
/// local HTTP server. The engine config (port and auth token) is injected
/// into the page as `window.__OPENMENSES_ENGINE__` before any page scripts
/// run, so the Connect-RPC client can find the server at runtime.
///
/// Native message handlers:
///   "export" — receives { format: "json"|"csv", data: string, filename: string }
///              and presents a UIActivityViewController share sheet.
///   "print"  — triggers native printing of the current web view.
final class WebViewController: UIViewController, WKScriptMessageHandler {

    private var webView: WKWebView!

    override func viewDidLoad() {
        super.viewDidLoad()

        let webConfig = WKWebViewConfiguration()
        webConfig.allowsInlineMediaPlayback = true

        // Register native message handlers so the UI can trigger iOS actions.
        webConfig.userContentController.add(self, name: "export")
        webConfig.userContentController.add(self, name: "print")
        webConfig.userContentController.add(self, name: "healthkit")

        // Inject engine config before any page script runs.
        // The auth token is embedded inline — this is safe because the script
        // runs in a local WKWebView context with no external network access,
        // and the token is only valid on 127.0.0.1 (auth middleware in bridge.go).
        let engine = EngineManager.shared
        // Sanitise the token before embedding in a JS string literal.
        // The token is a 64-char hex string (0-9a-f only) so no escaping is
        // needed, but we guard defensively against any unexpected characters.
        let safeToken = engine.authToken
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
        let syncEnabled = HealthKitSyncService.shared.syncEnabled ? "true" : "false"
        let injectionSource = """
        window.__OPENMENSES_ENGINE__ = {
            port: \(engine.port),
            authToken: "\(safeToken)",
            healthKitSyncEnabled: \(syncEnabled)
        };
        """
        let userScript = WKUserScript(
            source: injectionSource,
            injectionTime: .atDocumentStart,
            forMainFrameOnly: true
        )
        webConfig.userContentController.addUserScript(userScript)

        webView = WKWebView(frame: view.bounds, configuration: webConfig)
        webView.autoresizingMask = [.flexibleWidth, .flexibleHeight]

        // Let the web content handle safe areas via CSS env(safe-area-inset-*).
        // Framework7's "safe-areas" class + viewport-fit=cover already apply
        // the correct padding. Using .automatic here would double-pad.
        webView.scrollView.contentInsetAdjustmentBehavior = .never

        // Disable rubber-band bounce to make the app feel more native.
        webView.scrollView.bounces = false

        view.addSubview(webView)

        // Load the UI from the Go HTTP server.
        if let url = URL(string: engine.engineBaseURL + "/") {
            webView.load(URLRequest(url: url))
        }
    }

    // MARK: - WKScriptMessageHandler

    func userContentController(
        _ userContentController: WKUserContentController,
        didReceive message: WKScriptMessage
    ) {
        switch message.name {
        case "export":
            handleExportMessage(message)
        case "print":
            handlePrintMessage()
        case "healthkit":
            handleHealthKitMessage(message)
        default:
            break
        }
    }

    // MARK: - Export

    /// Receives file data from the UI layer and presents a share sheet.
    /// Expected message body: { format: "json"|"csv", data: String, filename: String }
    private func handleExportMessage(_ message: WKScriptMessage) {
        guard let body = message.body as? [String: Any],
              let data = body["data"] as? String,
              let filename = body["filename"] as? String else {
            NSLog("WebViewController: invalid export message body")
            return
        }

        guard let fileData = data.data(using: .utf8) else {
            NSLog("WebViewController: could not encode export data as UTF-8")
            return
        }

        // Write to a temporary file so UIActivityViewController can access it.
        let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent(filename)
        do {
            try fileData.write(to: tempURL, options: .atomic)
        } catch {
            NSLog("WebViewController: failed to write temp export file: %@", error.localizedDescription)
            return
        }

        let activityVC = UIActivityViewController(activityItems: [tempURL], applicationActivities: nil)

        // On iPad, UIActivityViewController must be presented in a popover.
        if let popover = activityVC.popoverPresentationController {
            popover.sourceView = view
            popover.sourceRect = CGRect(x: view.bounds.midX, y: view.bounds.midY, width: 0, height: 0)
            popover.permittedArrowDirections = []
        }

        present(activityVC, animated: true)
    }

    // MARK: - HealthKit

    /// Handles HealthKit-related messages from the UI layer.
    ///
    /// Supported actions:
    /// - `setSyncEnabled` — toggles HealthKit sync on/off. Body: `{ action, enabled: Bool }`.
    ///   If enabling, immediately kicks off an import sync in the background.
    /// - `exportObservation` — writes a bleeding entry to HealthKit.
    ///   Body: `{ action, date: String (ISO-8601), flow: Int }`.
    private func handleHealthKitMessage(_ message: WKScriptMessage) {
        guard let body = message.body as? [String: Any],
              let action = body["action"] as? String else {
            NSLog("WebViewController: invalid healthkit message body")
            return
        }

        switch action {
        case "setSyncEnabled":
            let enabled = body["enabled"] as? Bool ?? false
            HealthKitSyncService.shared.syncEnabled = enabled
            if enabled {
                Task { await HealthKitSyncService.shared.performStartupSync() }
            }
        case "exportObservation":
            guard let date = body["date"] as? String,
                  let flow = body["flow"] as? Int else {
                NSLog("WebViewController: exportObservation missing date or flow")
                return
            }
            Task { await HealthKitSyncService.shared.exportToHealthKit(isoDate: date, flow: flow) }
        case "requestAuth":
            Task {
                do {
                    try await HealthKitManager.shared.requestAuthorization()
                } catch {
                    NSLog("WebViewController: HealthKit auth request failed: %@", error.localizedDescription)
                }
            }
        case "import":
            Task {
                do {
                    try await HealthKitSyncService.shared.importFromHealthKit()
                } catch {
                    NSLog("WebViewController: HealthKit import failed: %@", error.localizedDescription)
                }
            }
        default:
            break
        }
    }

    // MARK: - Print

    /// Triggers native printing of the current WKWebView contents.
    private func handlePrintMessage() {
        let printController = UIPrintInteractionController.shared
        let printInfo = UIPrintInfo(dictionary: nil)
        printInfo.outputType = .general
        printInfo.jobName = "OpenMenses Clinician Summary"
        printController.printInfo = printInfo
        printController.printFormatter = webView.viewPrintFormatter()
        printController.present(animated: true, completionHandler: nil)
    }
}

