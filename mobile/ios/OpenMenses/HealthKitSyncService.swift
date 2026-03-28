import Foundation
import HealthKit

/// Orchestrates bidirectional HealthKit sync for menstrual flow data.
///
/// **Import flow** (on app launch): Fetches HealthKit menstrual flow samples
/// added since the last sync date, and creates `BleedingObservation` records
/// in the engine via the Connect-RPC JSON API. The last-sync cursor is
/// persisted in `UserDefaults` so each launch only fetches new samples.
///
/// **Export flow** (called via WebView message): Writes a bleeding observation
/// to HealthKit when one is created in the app, so it appears in the Health app.
///
/// This class contains no domain logic — it only translates between HealthKit
/// data model values and the engine's Connect-RPC API.
final class HealthKitSyncService {
    static let shared = HealthKitSyncService()

    private let userDefaults = UserDefaults.standard
    private static let syncEnabledKey = "healthKitSyncEnabled"
    private static let lastSyncDateKey = "healthKitLastSyncDate"

    /// Whether the user has enabled HealthKit sync. Persisted in `UserDefaults`.
    var syncEnabled: Bool {
        get { userDefaults.bool(forKey: Self.syncEnabledKey) }
        set { userDefaults.set(newValue, forKey: Self.syncEnabledKey) }
    }

    /// The last time a successful import sync completed.
    /// Falls back to `.distantPast` on first run so all available samples are imported.
    private var lastSyncDate: Date {
        get { userDefaults.object(forKey: Self.lastSyncDateKey) as? Date ?? .distantPast }
        set { userDefaults.set(newValue, forKey: Self.lastSyncDateKey) }
    }

    private init() {}

    // MARK: - Startup sync

    /// Request HealthKit authorization (if not already granted) and import any
    /// menstrual flow samples logged since the last sync.
    ///
    /// Safe to call unconditionally on every launch — returns immediately if
    /// HealthKit is unavailable, sync is disabled, or the engine is not yet
    /// running. Errors are logged and swallowed so HealthKit failures never
    /// prevent the app from starting.
    func performStartupSync() async {
        guard syncEnabled else { return }
        guard HKHealthStore.isHealthDataAvailable() else { return }
        guard EngineManager.shared.port > 0 else { return }

        do {
            try await HealthKitManager.shared.requestAuthorization()
            try await importFromHealthKit()
        } catch {
            NSLog("HealthKitSyncService: startup sync failed: %@", error.localizedDescription)
        }
    }

    // MARK: - Import (HealthKit → engine)

    /// Fetch HealthKit menstrual flow samples added since the last sync and
    /// create a `BleedingObservation` in the engine for each one.
    ///
    /// The last-sync cursor advances to `now` after the batch even if some
    /// individual samples fail, preventing infinite retries on transient errors.
    /// Samples that already exist in the engine (HTTP 409) are skipped silently.
    func importFromHealthKit() async throws {
        let since = lastSyncDate
        let samples = try await HealthKitManager.shared.fetchMenstrualFlow(since: since)
        guard !samples.isEmpty else { return }

        let engineBaseURL = EngineManager.shared.engineBaseURL
        let authToken = EngineManager.shared.authToken

        for sample in samples {
            do {
                try await createBleedingObservation(
                    date: sample.startDate,
                    flow: sample.flowLevel,
                    engineBaseURL: engineBaseURL,
                    authToken: authToken
                )
            } catch SyncError.alreadyExists {
                // Observation was previously imported — skip silently.
            } catch {
                NSLog(
                    "HealthKitSyncService: failed to import sample at %@: %@",
                    "\(sample.startDate)",
                    error.localizedDescription
                )
                // Continue with the remaining samples; don't abort the whole batch.
            }
        }

        // Advance the sync cursor so the next launch only fetches new samples.
        lastSyncDate = Date()
    }

    // MARK: - Export (engine → HealthKit)

    /// Write a bleeding observation to HealthKit when the user logs one in the app.
    ///
    /// Called from `WebViewController`'s healthkit message handler when the UI
    /// sends an `exportObservation` message.
    ///
    /// - Parameters:
    ///   - isoDate: ISO-8601 timestamp string, e.g. `"2024-06-15T10:00:00Z"`.
    ///   - flow: Proto `BleedingFlow` enum integer (1–4).
    func exportToHealthKit(isoDate: String, flow: Int) async {
        guard syncEnabled else { return }
        guard HKHealthStore.isHealthDataAvailable() else { return }

        let formatter = ISO8601DateFormatter()
        guard let date = formatter.date(from: isoDate) else {
            NSLog("HealthKitSyncService: invalid ISO date string for export: %@", isoDate)
            return
        }

        do {
            try await HealthKitManager.shared.writeMenstrualFlow(
                date: date,
                flow: flow,
                isStartOfCycle: false
            )
        } catch {
            NSLog("HealthKitSyncService: export to HealthKit failed: %@", error.localizedDescription)
        }
    }

    // MARK: - Connect-RPC helpers

    /// POST a `CreateBleedingObservation` request to the engine using the
    /// Connect-RPC JSON protocol.
    ///
    /// - Throws: `SyncError.alreadyExists` when the engine returns HTTP 409
    ///   (the observation was previously imported). Throws `SyncError.serverError`
    ///   for any other non-2xx response.
    private func createBleedingObservation(
        date: Date,
        flow: Int,
        engineBaseURL: String,
        authToken: String
    ) async throws {
        let flowName = protoFlowName(flow)
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        let isoDate = formatter.string(from: date)

        let body: [String: Any] = [
            "parent": "users/default",
            "observation": [
                "timestamp": ["value": isoDate],
                "flow": flowName,
            ],
        ]

        guard let url = URL(
            string: "\(engineBaseURL)/openmenses.v1.CycleTrackerService/CreateBleedingObservation"
        ) else {
            throw SyncError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("1", forHTTPHeaderField: "Connect-Protocol-Version")
        request.setValue("Bearer \(authToken)", forHTTPHeaderField: "Authorization")
        request.httpBody = try JSONSerialization.data(withJSONObject: body)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw SyncError.serverError
        }

        // 409 = AlreadyExists — observation was previously imported; skip it.
        if httpResponse.statusCode == 409 {
            throw SyncError.alreadyExists
        }
        guard (200..<300).contains(httpResponse.statusCode) else {
            throw SyncError.serverError
        }
    }
}

// MARK: - Value mapping

/// Return the proto3 JSON string name for a `BleedingFlow` integer value.
private func protoFlowName(_ protoInt: Int) -> String {
    switch protoInt {
    case 1: return "BLEEDING_FLOW_SPOTTING"
    case 2: return "BLEEDING_FLOW_LIGHT"
    case 3: return "BLEEDING_FLOW_MEDIUM"
    case 4: return "BLEEDING_FLOW_HEAVY"
    default: return "BLEEDING_FLOW_UNSPECIFIED"
    }
}

// MARK: - Error types

enum SyncError: LocalizedError {
    case invalidURL
    case alreadyExists
    case serverError

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid engine URL for HealthKit sync request."
        case .alreadyExists:
            return "Observation already exists in the engine."
        case .serverError:
            return "Engine returned a non-2xx response during HealthKit sync."
        }
    }
}
