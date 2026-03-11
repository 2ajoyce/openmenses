# Domain Rules — openmenses

This document is the authoritative specification of domain rules for the openmenses engine.
All rules here **must** be implemented in the Go engine (`engine/`). The UI must not re-implement them.

> **Note:** This is a living document. Update it before changing engine behaviour.

---

## Guiding Principles

1. **Offline-first.** All rules must operate without a network connection.
2. **Privacy-first.** No user data leaves the device without explicit user action.
3. **User is the authority.** The app may suggest or predict, but the user's logged data is always correct.
4. **Conservative predictions.** When uncertain, prefer conservative estimates and communicate uncertainty clearly.

---

## 1. Cycle Tracking Rules

### 1.1 Cycle Start Definition

A new cycle begins on the **first calendar day of a bleeding episode** that is preceded by a **non-bleeding gap of at least 3 days**.

- A "bleeding episode" is one or more consecutive days that have at least one `BleedingObservation` with any flow other than `BLEEDING_FLOW_UNSPECIFIED`.
- Days with `BLEEDING_FLOW_SPOTTING` only are **not** considered a cycle start unless they are immediately followed (within 2 days) by at least one day of `BLEEDING_FLOW_LIGHT`, `BLEEDING_FLOW_MEDIUM`, or `BLEEDING_FLOW_HEAVY`.
  - Spotting that is not followed by heavier bleeding within 2 days is treated as mid-cycle spotting and **does not** start a new cycle.
- The minimum non-bleeding gap before a new cycle start is **3 days**. Bleeding that resumes within 3 days of a previous bleed is considered a continuation of the same episode.

### 1.2 Cycle Length Bounds

| Bound                      | Value   |
| -------------------------- | ------- |
| Minimum valid cycle length | 15 days |
| Maximum valid cycle length | 90 days |

Cycles shorter than 15 days or longer than 90 days are **flagged** with a warning but are stored and not rejected. The engine records these outliers and excludes them from statistics and predictions by default.

**Implementation:** The `rules.IsOutlierLength()` function determines whether a completed cycle falls outside the 15–90 day bounds. `rules.Stats()` and `rules.WindowStats()` automatically exclude outlier-length cycles from their computations. Outlier cycles are still stored and returned by `DetectCycles` and `ListCycles`; they are filtered only at the statistics layer.

### 1.3 Cycle `end_date` Semantics

`Cycle.end_date` is the **day before the start of the next cycle**. It is inclusive: the last day that belongs to a cycle is `end_date`.

Example: if cycle A starts on 2026-01-01 and cycle B starts on 2026-01-29, then cycle A has `end_date = 2026-01-28` and a length of 28 days.

### 1.4 Open-Ended (In-Progress) Cycles

The most recent cycle is **open-ended** when no subsequent bleeding has been logged to close it. An open-ended cycle has `end_date` unset (the zero-value `LocalDate` with an empty string). The engine must never assign a speculative `end_date` to the current ongoing cycle.

### 1.5 `CycleSource` Semantics

| Source                               | Meaning                                                         |
| ------------------------------------ | --------------------------------------------------------------- |
| `CYCLE_SOURCE_DERIVED_FROM_BLEEDING` | Engine computed this cycle boundary from bleeding observations. |
| `CYCLE_SOURCE_USER_CONFIRMED`        | The user manually set or confirmed this cycle boundary.         |

Rules:

- The engine derives cycles automatically for all users.
- A `USER_CONFIRMED` cycle boundary **overrides** the engine-derived boundary for the same date range. The engine will not re-derive or overwrite a `USER_CONFIRMED` cycle.
- When a new `BleedingObservation` conflicts with a `USER_CONFIRMED` boundary, the engine preserves the user-confirmed boundary and flags the conflict rather than silently overwriting it.

### 1.6 Initial Onboarding Behavior

When a user logs their **very first** `BleedingObservation`, the engine creates a single open-ended cycle starting on that observation's date with `source = CYCLE_SOURCE_DERIVED_FROM_BLEEDING`.

If the user provides onboarding data indicating a recent past cycle start date (e.g. "my last period started 10 days ago"), the engine creates a cycle with `start_date` set to that past date, `source = CYCLE_SOURCE_USER_CONFIRMED`, and no `end_date`.

No predictions or phase estimates are generated until at least 2 completed cycles exist (see §4).

---

## 2. Phase Estimation Rules

### 2.1 Ovulatory Cycle Phase Model (`BIOLOGICAL_CYCLE_MODEL_OVULATORY`)

Default phase day ranges, anchored to cycle day 1 (first day of bleeding):

| Phase                          | Default day range | Notes                                     |
| ------------------------------ | ----------------- | ----------------------------------------- |
| `CYCLE_PHASE_MENSTRUATION`     | Days 1–5          | Extends if average bleed duration differs |
| `CYCLE_PHASE_FOLLICULAR`       | Days 6–(O-2)      | O = estimated ovulation day               |
| `CYCLE_PHASE_OVULATION_WINDOW` | Days (O-1)–(O+1)  | 3-day window                              |
| `CYCLE_PHASE_LUTEAL`           | Days (O+2)–end    | Luteal phase length ≈ 14 days             |

Ovulation day **O** is estimated as `cycle_length − 14`. For a 28-day cycle: O = 14.

When ≥ 3 completed cycles are available, the engine uses the user's average cycle length to compute O. Otherwise the engine uses 28 days as the assumed cycle length to compute rough estimates, with `CONFIDENCE_LEVEL_LOW`.

### 2.2 Hormonally Suppressed Cycle Phase Model (`BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED`)

- No ovulation window is estimated.
- Phases are limited to `CYCLE_PHASE_MENSTRUATION` (withdrawal bleed, days 1–5) and an unnamed non-menstruation phase for the remainder.
- In storage, the non-menstruation phase is stored as `CYCLE_PHASE_FOLLICULAR` to avoid a new enum value, but the UI should present it as "Pill-free / Active pill days" rather than "Follicular".
- Predictions are limited to `PREDICTION_TYPE_NEXT_BLEED` and `PREDICTION_TYPE_PMS_WINDOW` (withdrawal-bleed window).

### 2.3 Irregular Cycle Phase Model (`BIOLOGICAL_CYCLE_MODEL_IRREGULAR`)

- Ovulation window is estimated with `CONFIDENCE_LEVEL_LOW` regardless of data quantity.
- Confidence never exceeds `CONFIDENCE_LEVEL_MEDIUM` for any phase estimate.
- Phase ranges use wider windows: ±3 days on all boundaries.

**Implementation:** The `rules.irregularPhaseFn` applies ±3-day widening to every phase boundary compared to the standard ovulatory model. For a cycle of length `L` with ovulation day `O = L − 14`:

| Phase            | Standard day range | Widened day range (Irregular) |
| ---------------- | ------------------ | ----------------------------- |
| Menstruation     | 1–5                | 1–8                           |
| Follicular       | 6–(O-2)            | 9–(O-5)                       |
| Ovulation window | (O-1)–(O+1)        | (O-4)–(O+4) ← 9-day window    |
| Luteal           | (O+2)–end          | (O+5)–end                     |

For shorter cycles where `O ≤ 13` (i.e., cycle length ≤ 27 days), the follicular phase is absent and the ovulation window begins immediately after menstruation.

### 2.4 Confidence Assignment

| Condition                             | Confidence                       |
| ------------------------------------- | -------------------------------- |
| < 2 completed cycles                  | `CONFIDENCE_LEVEL_LOW`           |
| 2–4 completed cycles, regular         | `CONFIDENCE_LEVEL_MEDIUM`        |
| ≥ 5 completed cycles, regular         | `CONFIDENCE_LEVEL_HIGH`          |
| `CYCLE_REGULARITY_VERY_IRREGULAR`     | cap at `CONFIDENCE_LEVEL_LOW`    |
| `CYCLE_REGULARITY_SOMEWHAT_IRREGULAR` | cap at `CONFIDENCE_LEVEL_MEDIUM` |
| `BIOLOGICAL_CYCLE_MODEL_IRREGULAR`    | cap at `CONFIDENCE_LEVEL_MEDIUM` |

Rules are cumulative — the lowest applicable cap wins.

### 2.5 Minimum Cycles for Phase Estimation

Phase estimation requires **at least 1 completed cycle**. With exactly 1 cycle, the engine uses that cycle's length as the basis, and assigns `CONFIDENCE_LEVEL_LOW` to all estimates.

---

## 3. Observation Rules

### 3.1 Uniqueness Constraints

- A user **may** log multiple `BleedingObservation` records on the same calendar day (e.g., morning and evening). Each must have a distinct `id` and `timestamp`. There is no uniqueness constraint on (user_id, date) for bleeding.
- A user **may** log multiple `SymptomObservation` records of the **same** `SymptomType` on the same day. This is intentional (e.g., cramps in the morning and again in the evening).
- A user **may** log multiple `MoodObservation` records of the same `MoodType` on the same day.
- There are **no** uniqueness constraints on `MedicationEvent` records beyond the `id` field.

### 3.2 Timestamp Rules

- All `DateTime` timestamps are stored in **UTC** as RFC 3339 strings (e.g., `2026-03-15T14:30:00Z`).
- All `LocalDate` values are stored as `YYYY-MM-DD` calendar dates with no timezone component. They represent the user's local calendar date at the moment of logging (the client is responsible for supplying the correct local date).
- Observations **must not** have a `timestamp` more than **1 minute in the future** relative to the engine's wall clock at the time of submission. The engine rejects such observations with `codes.InvalidArgument`.

### 3.3 ID Generation

- All `id` fields are **ULIDs** (Universally Unique Lexicographically Sortable Identifiers), generated by the engine at the time of record creation.
- Client-supplied IDs are **not accepted**. The engine ignores any `id` field in a Create request and always generates a new ULID.
- ULID generation must be monotonic within a single engine instance (use a monotonic ULID factory with the standard millisecond timestamp).

### 3.4 Referential Integrity for MedicationEvent

- `MedicationEvent.medication_id` must reference an existing `Medication` record for the same `user_id`.
- The referenced `Medication` must be `active = true` at the time of the event. Logging a medication event against an inactive medication is rejected with `codes.InvalidArgument`.

---

## 4. Prediction Eligibility Rules

### 4.1 Minimum Cycles Required

| Prediction Type                    | Minimum completed cycles                                            |
| ---------------------------------- | ------------------------------------------------------------------- |
| `PREDICTION_TYPE_NEXT_BLEED`       | 2                                                                   |
| `PREDICTION_TYPE_PMS_WINDOW`       | 2                                                                   |
| `PREDICTION_TYPE_OVULATION_WINDOW` | 3 with `CYCLE_REGULARITY_REGULAR` or `SOMEWHAT_IRREGULAR`           |
| `PREDICTION_TYPE_SYMPTOM_WINDOW`   | 3 with at least 3 matching symptom observations across those cycles |

### 4.2 Model and Regularity Gating

- `PREDICTION_TYPE_OVULATION_WINDOW` is **not generated** for `BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED` or `BIOLOGICAL_CYCLE_MODEL_IRREGULAR` users.
- `PREDICTION_TYPE_NEXT_BLEED` is always attempted once the minimum cycle count is reached, regardless of model.

### 4.3 Prediction Invalidation

A prediction is **invalidated** (deleted and re-generated) when:

- A new `BleedingObservation` is logged that starts a new cycle (which shifts future cycle boundaries).
- A `USER_CONFIRMED` cycle boundary is added or modified.
- The user updates `UserProfile.biological_cycle` or `UserProfile.cycle_regularity`.

Invalidated predictions are deleted from storage. Fresh predictions are re-generated immediately after the triggering event is processed.

---

## 5. Validation Rules

### 5.1 Schema-Level vs. Engine-Level

Validation is split into two layers:

| Layer        | Tool                    | Coverage                                                                                                                           |
| ------------ | ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| Schema-level | `protovalidate-go`      | Field presence, string lengths, enum membership, list min/max, uniqueness annotations — exactly what is declared in `.proto` files |
| Domain-level | Engine validation logic | Cross-field, cross-record, temporal, referential integrity                                                                         |

The engine always runs **schema-level validation first**. If schema validation fails, domain validation is not attempted.

### 5.2 Cross-Field Validation

| Rule                                                                               | Error                   |
| ---------------------------------------------------------------------------------- | ----------------------- |
| `Cycle.end_date` must be ≥ `Cycle.start_date` when both are set                    | `codes.InvalidArgument` |
| `DateRange.end` must be ≥ `DateRange.start`                                        | `codes.InvalidArgument` |
| `Prediction.predicted_end_date` must be ≥ `predicted_start_date` when both are set | `codes.InvalidArgument` |

### 5.3 Cross-Record Validation

| Rule                                                                                              | Error                   |
| ------------------------------------------------------------------------------------------------- | ----------------------- |
| No two `Cycle` records for the same `user_id` may have overlapping date ranges                    | `codes.InvalidArgument` |
| `MedicationEvent.medication_id` must reference an existing, active `Medication` for the same user | `codes.InvalidArgument` |

### 5.4 Temporal Validation

| Rule                                                                         | Error                   |
| ---------------------------------------------------------------------------- | ----------------------- |
| `BleedingObservation.timestamp` must not be more than 1 minute in the future | `codes.InvalidArgument` |
| `SymptomObservation.timestamp` must not be more than 1 minute in the future  | `codes.InvalidArgument` |
| `MoodObservation.timestamp` must not be more than 1 minute in the future     | `codes.InvalidArgument` |
| `MedicationEvent.timestamp` must not be more than 1 minute in the future     | `codes.InvalidArgument` |

### 5.5 `UserProfile` Completeness

The engine requires the following `UserProfile` fields to be set before generating predictions or phase estimates:

- `biological_cycle` (non-UNSPECIFIED)
- `cycle_regularity` (non-UNSPECIFIED)
- `tracking_focus` (at least one value, enforced by proto)

All other fields are optional enrichment. The engine operates (stores observations, detects cycles) without a complete profile; it simply skips predictions/phase estimates until the required fields are present.

### 5.6 Structured Validation Errors

The engine returns validation errors as a `connect.Error` with code `codes.InvalidArgument`. The error detail includes a list of `FieldViolation` entries (one per failed field), each containing:

- `field`: dot-separated path to the offending field (e.g., `cycle.end_date`)
- `description`: human-readable reason in English (e.g., `"end_date must be on or after start_date"`)

Multiple violations from a single request are **all reported** in a single error response (no fail-fast).
