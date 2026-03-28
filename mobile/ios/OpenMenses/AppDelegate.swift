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
            // The app cannot function without the engine — crash with a
            // descriptive message rather than silently presenting a broken UI.
            fatalError("Failed to start engine: \(error.localizedDescription)")
        }

        // Import HealthKit menstrual flow samples added since the last sync.
        // Runs asynchronously so it never delays app launch. Returns immediately
        // if sync is disabled or HealthKit is unavailable.
        Task {
            await HealthKitSyncService.shared.performStartupSync()
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

    func application(
        _ application: UIApplication,
        didDiscardSceneSessions sceneSessions: Set<UISceneSession>
    ) {}
}
