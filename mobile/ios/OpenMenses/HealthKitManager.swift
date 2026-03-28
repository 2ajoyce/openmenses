import Foundation
import HealthKit

/// Manages HealthKit read/write operations for menstrual flow data.
///
/// This class is a thin HealthKit adapter. All domain logic (duplicate
/// detection, cycle analysis, etc.) lives in the Go engine. The manager
/// surfaces raw HealthKit samples as `MenstrualFlowSample` values that
/// callers (Step 15 sync orchestration) translate into Connect-RPC calls.
///
/// **Flow level mapping** (HealthKit ↔ proto `BleedingFlow` enum int):
///
/// | HealthKit                   | Proto int | Proto name                |
/// |-----------------------------|-----------|---------------------------|
/// | `.unspecified`              | —         | skip on import            |
/// | `.none`                     | —         | skip on import            |
/// | `.light`                    | 2         | `BLEEDING_FLOW_LIGHT`     |
/// | `.medium`                   | 3         | `BLEEDING_FLOW_MEDIUM`    |
/// | `.heavy`                    | 4         | `BLEEDING_FLOW_HEAVY`     |
///
/// Reverse (export): proto `BLEEDING_FLOW_SPOTTING (1)` maps to HK `.light`
/// since HealthKit has no direct spotting value.
final class HealthKitManager {
    static let shared = HealthKitManager()
    private let healthStore = HKHealthStore()
    private let menstrualFlowType = HKCategoryType(.menstrualFlow)

    private init() {}

    // MARK: - Authorization

    /// Request read/write authorization for menstrual flow data.
    ///
    /// Must be called before any read or write operations. Presents the
    /// system HealthKit authorization sheet on first call; subsequent calls
    /// return immediately if authorization has already been determined.
    ///
    /// - Throws: An error if HealthKit is unavailable on this device, or if
    ///   the authorization request itself fails.
    func requestAuthorization() async throws {
        guard HKHealthStore.isHealthDataAvailable() else {
            throw HealthKitError.unavailable
        }
        let shareTypes: Set<HKSampleType> = [menstrualFlowType]
        let readTypes: Set<HKObjectType> = [menstrualFlowType]
        try await healthStore.requestAuthorization(toShare: shareTypes, read: readTypes)
    }

    // MARK: - Read

    /// Query HealthKit for menstrual flow samples on or after `since`.
    ///
    /// Samples with `.unspecified` or `.none` flow values are silently
    /// skipped — they have no representation in the proto `BleedingFlow`
    /// enum (the engine requires a non-unspecified flow value).
    ///
    /// - Parameter since: Lower bound for sample start dates (inclusive).
    /// - Returns: Samples converted to proto-compatible `MenstrualFlowSample`
    ///   values, sorted ascending by date.
    /// - Throws: ``HealthKitError/unavailable`` if HealthKit is not available,
    ///   or a wrapped HK error on query failure.
    func fetchMenstrualFlow(since: Date) async throws -> [MenstrualFlowSample] {
        guard HKHealthStore.isHealthDataAvailable() else {
            throw HealthKitError.unavailable
        }

        let predicate = HKQuery.predicateForSamples(
            withStart: since,
            end: nil,
            options: .strictStartDate
        )
        let sortDescriptor = NSSortDescriptor(
            key: HKSampleSortIdentifierStartDate,
            ascending: true
        )

        return try await withCheckedThrowingContinuation { continuation in
            let query = HKSampleQuery(
                sampleType: menstrualFlowType,
                predicate: predicate,
                limit: HKObjectQueryNoLimit,
                sortDescriptors: [sortDescriptor]
            ) { _, samples, error in
                if let error {
                    continuation.resume(throwing: HealthKitError.queryFailed(error))
                    return
                }

                let results: [MenstrualFlowSample] = (samples ?? []).compactMap { sample in
                    guard let categorySample = sample as? HKCategorySample else { return nil }
                    guard let hkFlow = HKCategoryValueMenstrualFlow(rawValue: categorySample.value) else {
                        return nil
                    }
                    guard let protoFlow = hkFlowToProtoInt(hkFlow) else {
                        // .unspecified and .none have no proto equivalent — skip.
                        return nil
                    }
                    let isStartOfCycle = categorySample.metadata?[HKMetadataKeyMenstrualCycleStart] as? Bool ?? false
                    return MenstrualFlowSample(
                        startDate: categorySample.startDate,
                        endDate: categorySample.endDate,
                        flowLevel: protoFlow,
                        isStartOfCycle: isStartOfCycle
                    )
                }
                continuation.resume(returning: results)
            }
            healthStore.execute(query)
        }
    }

    // MARK: - Write

    /// Write a single menstrual flow sample to HealthKit.
    ///
    /// `flow` is a proto `BleedingFlow` enum integer value (1–4). Values of
    /// 0 (`BLEEDING_FLOW_UNSPECIFIED`) are rejected and cause the method to
    /// throw ``HealthKitError/unsupportedFlowValue``.
    ///
    /// - Parameters:
    ///   - date: The date of the observation. Used as both start and end date
    ///     (HealthKit menstrual flow samples are point-in-time observations).
    ///   - flow: Proto `BleedingFlow` integer (1 = Spotting → HK Light,
    ///     2 = Light, 3 = Medium, 4 = Heavy).
    ///   - isStartOfCycle: Whether this observation marks the start of a new
    ///     cycle. Sets `HKMetadataKeyMenstrualCycleStart` on the sample.
    /// - Throws: ``HealthKitError/unavailable``, ``HealthKitError/unsupportedFlowValue``,
    ///   or a wrapped HK error on save failure.
    func writeMenstrualFlow(date: Date, flow: Int, isStartOfCycle: Bool) async throws {
        guard HKHealthStore.isHealthDataAvailable() else {
            throw HealthKitError.unavailable
        }
        guard let hkFlow = protoIntToHKFlow(flow) else {
            throw HealthKitError.unsupportedFlowValue(flow)
        }

        var metadata: [String: Any] = [
            HKMetadataKeyMenstrualCycleStart: isStartOfCycle,
        ]
        // Suppress the isStartOfCycle key when false to keep metadata clean.
        if !isStartOfCycle {
            metadata.removeValue(forKey: HKMetadataKeyMenstrualCycleStart)
        }

        let sample = HKCategorySample(
            type: menstrualFlowType,
            value: hkFlow.rawValue,
            start: date,
            end: date,
            metadata: metadata.isEmpty ? nil : metadata
        )

        try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            healthStore.save(sample) { _, error in
                if let error {
                    continuation.resume(throwing: HealthKitError.saveFailed(error))
                } else {
                    continuation.resume()
                }
            }
        }
    }
}

// MARK: - Value mapping helpers

/// Convert a HealthKit menstrual flow value to the proto `BleedingFlow` integer.
///
/// Returns `nil` for `.unspecified` and `.none` — these have no safe mapping
/// to a non-zero proto enum value.
private func hkFlowToProtoInt(_ hkFlow: HKCategoryValueMenstrualFlow) -> Int? {
    switch hkFlow {
    case .unspecified:
        return nil
    case .none:
        return nil
    case .light:
        return 2 // BLEEDING_FLOW_LIGHT
    case .medium:
        return 3 // BLEEDING_FLOW_MEDIUM
    case .heavy:
        return 4 // BLEEDING_FLOW_HEAVY
    @unknown default:
        return nil
    }
}

/// Convert a proto `BleedingFlow` integer to a HealthKit flow enum value.
///
/// Returns `nil` for 0 (`BLEEDING_FLOW_UNSPECIFIED`), which must not be
/// written to HealthKit. `BLEEDING_FLOW_SPOTTING (1)` is mapped to `.light`
/// because HealthKit has no distinct spotting value.
private func protoIntToHKFlow(_ protoInt: Int) -> HKCategoryValueMenstrualFlow? {
    switch protoInt {
    case 0: // BLEEDING_FLOW_UNSPECIFIED — cannot be written
        return nil
    case 1: // BLEEDING_FLOW_SPOTTING — closest HK equivalent is light
        return .light
    case 2: // BLEEDING_FLOW_LIGHT
        return .light
    case 3: // BLEEDING_FLOW_MEDIUM
        return .medium
    case 4: // BLEEDING_FLOW_HEAVY
        return .heavy
    default:
        return nil
    }
}

// MARK: - Supporting types

/// A HealthKit menstrual flow sample translated into proto-compatible values.
struct MenstrualFlowSample {
    /// Start of the sample window (use for the observation timestamp).
    let startDate: Date
    /// End of the sample window.
    let endDate: Date
    /// Proto `BleedingFlow` enum integer (2 = Light, 3 = Medium, 4 = Heavy).
    /// Never 0 (unspecified) or 1 (spotting) — HealthKit has no spotting value.
    let flowLevel: Int
    /// Whether this sample was marked as the start of a menstrual cycle in
    /// HealthKit (`HKMetadataKeyMenstrualCycleStart`).
    let isStartOfCycle: Bool
}

/// Errors surfaced by `HealthKitManager`.
enum HealthKitError: LocalizedError {
    case unavailable
    case queryFailed(Error)
    case saveFailed(Error)
    case unsupportedFlowValue(Int)

    var errorDescription: String? {
        switch self {
        case .unavailable:
            return "HealthKit is not available on this device."
        case .queryFailed(let error):
            return "HealthKit query failed: \(error.localizedDescription)"
        case .saveFailed(let error):
            return "HealthKit save failed: \(error.localizedDescription)"
        case .unsupportedFlowValue(let value):
            return "BleedingFlow value \(value) cannot be written to HealthKit (UNSPECIFIED is not allowed)."
        }
    }
}
