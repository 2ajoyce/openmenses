import UIKit
import WebKit

/// Hosts a WKWebView that loads the bundled React UI from the Go engine's
/// local HTTP server. The engine config (port and auth token) is injected
/// into the page as `window.__OPENMENSES_ENGINE__` before any page scripts
/// run, so the Connect-RPC client can find the server at runtime.
final class WebViewController: UIViewController {

    private var webView: WKWebView!

    override func viewDidLoad() {
        super.viewDidLoad()

        let webConfig = WKWebViewConfiguration()
        webConfig.allowsInlineMediaPlayback = true

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
        let injectionSource = """
        window.__OPENMENSES_ENGINE__ = {
            port: \(engine.port),
            authToken: "\(safeToken)"
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
}
