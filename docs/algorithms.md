# How OpenMenses Uses Your Data

This document explains how OpenMenses processes the observations you log to detect cycles, estimate cycle phases, generate predictions, and surface insights. All processing happens entirely on your device — no data is ever sent to a server.

Each section includes links to the source code that implements the described logic, so you can verify exactly how your data is processed.

---

## Table of Contents

- [Overview](#overview)
- [1. Cycle Detection](#1-cycle-detection)
- [2. Cycle Statistics](#2-cycle-statistics)
- [3. Phase Estimation](#3-phase-estimation)
- [4. Predictions](#4-predictions)
- [5. Insights](#5-insights)
- [6. Confidence Levels](#6-confidence-levels)
- [7. When Results Are Recalculated](#7-when-results-are-recalculated)

---

## Overview

OpenMenses processes your data through several layers, each building on the last:

1. **Cycle Detection** — Your bleeding observations are analyzed to identify where each menstrual cycle starts and ends.
2. **Cycle Statistics** — Completed cycles are measured to compute averages, medians, and variability.
3. **Phase Estimation** — Each day within a cycle is assigned to a phase (menstruation, follicular, ovulation window, or luteal).
4. **Predictions** — Forward-looking estimates for your next period, PMS window, ovulation window, and recurring symptoms.
5. **Insights** — Backward-looking summaries that identify trends and patterns in your historical data.

Your manually confirmed cycle boundaries always take priority over automatically detected ones. The app never overwrites data you have explicitly set.

---

## 1. Cycle Detection

> Source: [`engine/internal/rules/detect.go`](../engine/internal/rules/detect.go)

### How cycles are identified

A new cycle begins on the **first day of a bleeding episode** that is preceded by at least [**3 consecutive non-bleeding days**](../engine/internal/rules/detect.go#L19-L23). This gap requirement prevents brief pauses in bleeding from being misinterpreted as a new cycle. The full detection algorithm is implemented in [`DetectCycles`](../engine/internal/rules/detect.go#L49) and episode boundaries are identified in [`findEpisodeStarts`](../engine/internal/rules/detect.go#L187).

### Spotting handling

If a day has only [spotting-level flow](../engine/internal/rules/detect.go#L217) (no light, medium, or heavy bleeding), it is **not** treated as the start of a new cycle unless [heavier bleeding occurs within the following 2 days](../engine/internal/rules/detect.go#L228). Isolated spotting mid-cycle is recorded but does not trigger a new cycle boundary.

### Cycle boundaries

- **Start date:** The first day of the qualifying bleeding episode.
- **End date:** The day before the next cycle's start date. For example, if Cycle A starts January 1 and Cycle B starts January 29, then Cycle A ends January 28 (a 28-day cycle). This is computed in [`computeDerivedCycles`](../engine/internal/rules/detect.go#L124).
- **Open-ended cycles:** Your most recent cycle has no end date until a subsequent period begins. The app never guesses an end date for your current cycle.

### Outlier cycles

Cycles shorter than **15 days** or longer than **90 days** are flagged as outliers by [`IsOutlierLength`](../engine/internal/rules/detect.go#L36). These [bounds](../engine/internal/rules/detect.go#L27-L31) are defined as constants. Outlier cycles are still stored and shown to you, but they are excluded from statistical calculations and predictions to avoid skewing results.

### User-confirmed boundaries

If you manually set or confirm a cycle boundary, it takes priority over the automatically detected one. The app [preserves user-confirmed cycles](../engine/internal/rules/detect.go#L101) and will not overwrite your confirmed boundaries. If new bleeding data conflicts with a boundary you confirmed, the conflict is flagged rather than silently resolved.

---

## 2. Cycle Statistics

> Source: [`engine/internal/rules/stats.go`](../engine/internal/rules/stats.go)

Statistics are computed from your completed, non-outlier cycles via [`Stats`](../engine/internal/rules/stats.go#L30). The following measures are calculated in [`buildStats`](../engine/internal/rules/stats.go#L96):

| Statistic              | Description                                       |
| ---------------------- | ------------------------------------------------- |
| **Average length**     | Mean number of days across your completed cycles  |
| **Median length**      | Middle value when cycle lengths are sorted        |
| **Shortest cycle**     | Length of your shortest completed cycle           |
| **Longest cycle**      | Length of your longest completed cycle            |
| **Standard deviation** | How much your cycle lengths vary from the average |

Open-ended (in-progress) cycles and [outlier-length cycles](../engine/internal/rules/stats.go#L65) (shorter than 15 days or longer than 90 days) are excluded from these calculations. Cycle length is computed by [`CycleLength`](../engine/internal/rules/stats.go#L77).

A **windowed** variant of these statistics can also be computed over just your most recent _n_ cycles via [`WindowStats`](../engine/internal/rules/stats.go#L41), giving a more current picture if your cycle length has changed over time.

---

## 3. Phase Estimation

> Source: [`engine/internal/rules/phases.go`](../engine/internal/rules/phases.go)

Once at least **1 completed cycle** exists, the app estimates which phase each day of a cycle falls into via [`EstimatePhases`](../engine/internal/rules/phases.go#L26). The estimation method depends on your biological cycle model setting.

### Ovulatory model (default)

For users with a standard ovulatory cycle, [`ovulatoryPhaseFn`](../engine/internal/rules/phases.go#L101) divides each cycle into four phases:

| Phase                | Default days    | How it's calculated                                    |
| -------------------- | --------------- | ------------------------------------------------------ |
| **Menstruation**     | Days 1–5        | Fixed at the start of each cycle                       |
| **Follicular**       | Days 6 to O−2   | From end of menstruation to before ovulation           |
| **Ovulation window** | Days O−1 to O+1 | A 3-day window centered on the estimated ovulation day |
| **Luteal**           | Days O+2 to end | From after ovulation to the end of the cycle           |

**Estimating ovulation day (O):** The ovulation day is estimated as your cycle length minus 14. For a 28-day cycle, that places ovulation around day 14. The ovulation day is [never placed earlier than day 6](../engine/internal/rules/phases.go#L103), even for very short cycles. When 3 or more completed cycles are available, your personal average cycle length is used; otherwise, a [28-day default](../engine/internal/rules/phases.go#L12) is assumed.

### Hormonally suppressed model

For users on hormonal contraception, [`suppressedPhaseFn`](../engine/internal/rules/phases.go#L158) applies:

- **Menstruation** (withdrawal bleed) is estimated for days 1–5.
- The remaining days are a single non-menstruation phase (displayed as "Pill-free / Active pill days" in the app).
- No ovulation window is estimated.

### Irregular cycle model

For users who have reported irregular cycles, [`irregularPhaseFn`](../engine/internal/rules/phases.go#L131) applies:

- The same four phases are estimated, but all phase boundaries are **widened by ±3 days** to account for greater variability.
- The ovulation window expands from a 3-day window to a **9-day window**.
- Confidence never exceeds Medium for any phase estimate (see [Confidence Levels](#6-confidence-levels)).

| Phase                | Standard range  | Irregular range |
| -------------------- | --------------- | --------------- |
| **Menstruation**     | Days 1–5        | Days 1–8        |
| **Follicular**       | Days 6 to O−2   | Days 9 to O−5   |
| **Ovulation window** | Days O−1 to O+1 | Days O−4 to O+4 |
| **Luteal**           | Days O+2 to end | Days O+5 to end |

For shorter cycles where the follicular phase would be empty, the ovulation window begins immediately after menstruation.

---

## 4. Predictions

> Source: [`engine/internal/predictions/generate.go`](../engine/internal/predictions/generate.go)

Predictions are forward-looking estimates about your upcoming cycle, produced by [`Generate`](../engine/internal/predictions/generate.go#L19). Each prediction type has specific eligibility requirements.

### Next period prediction

- **Requires:** At least 2 completed cycles.
- **Method:** The app calculates your average cycle length from non-outlier cycles, then [adds that number of days](../engine/internal/predictions/generate.go#L127) to the start of your current (or most recent) cycle. The predicted period spans 5 days starting from that date.
- **Example:** If your average cycle is 28 days and your current cycle started March 1, your next period is predicted to start around March 29.

### PMS window prediction

- **Requires:** At least 2 completed cycles.
- **Method:** The PMS window is estimated as the [**10 days before**](../engine/internal/predictions/generate.go#L152) your predicted next period start date (days −10 to −1 relative to the next bleed).

### Ovulation window prediction

- **Requires:** At least 3 completed cycles, a regularity setting of Regular or Somewhat Irregular, and a biological cycle model that is **not** Hormonally Suppressed or Irregular. These restrictions are checked by [`canPredictOvulation`](../engine/internal/predictions/generate.go#L110).
- **Method:** Ovulation day (O) is estimated as your average cycle length minus 14 (with a minimum of day 6). The [predicted window](../engine/internal/predictions/generate.go#L177) spans 3 days: from O−1 to O+1, calculated relative to your predicted next cycle start.

### Symptom window prediction

- **Requires:** At least 3 completed cycles, with at least 3 observations of the same symptom type occurring on similar cycle days (within ±2 days of each other) across at least 3 different cycles.
- **Method:** The app identifies symptoms that recur at consistent points in your cycle using a [clustering algorithm](../engine/internal/predictions/generate.go#L209):
  1. Each symptom observation is mapped to its cycle day (day 1 = first day of the cycle it falls in).
  2. Observations are grouped by symptom type.
  3. For each symptom type, the algorithm searches for a ["center day"](../engine/internal/predictions/generate.go#L318) where at least 3 observations from at least 3 distinct cycles cluster within a ±2-day window.
  4. If a qualifying cluster is found, a 5-day prediction window is generated centered on that cycle day for your next cycle.
- **Example:** If headaches are observed around cycle day 12 in three or more cycles, the app predicts a headache window around day 12 of your next cycle.

### Model and regularity restrictions

- Ovulation window predictions are [**never generated**](../engine/internal/predictions/generate.go#L110) for users with a Hormonally Suppressed or Irregular biological cycle model.
- Next period and PMS predictions are generated for all models once the minimum cycle count is met.

---

## 5. Insights

> Source: [`engine/internal/insights/generate.go`](../engine/internal/insights/generate.go)

Insights are backward-looking summaries that identify trends and patterns in your accumulated data, produced by [`Generate`](../engine/internal/insights/generate.go#L19). They require a [complete user profile](../engine/internal/insights/generate.go#L74) (biological cycle model, cycle regularity, and at least one tracking focus must be set).

### Cycle length pattern

- **Requires:** At least 3 completed non-outlier cycles.
- **Method:** The app performs [simple linear regression](../engine/internal/insights/generate.go#L148) on your cycle lengths (ordered chronologically) in [`cycleLengthPattern`](../engine/internal/insights/generate.go#L91) and classifies the trend:
  - **Shortening:** Your cycles are getting shorter (slope steeper than −0.5 days per cycle).
  - **Lengthening:** Your cycles are getting longer (slope steeper than +0.5 days per cycle).
  - **Stable:** Your cycle lengths vary by less than 8% (coefficient of variation < 0.08).
  - **Irregular:** High variance with no clear directional trend.
- **Example output:** "Your cycle length has remained stable at around 28 days."

### Symptom pattern

- **Requires:** At least 3 completed cycles with at least 3 observations of the same symptom type on similar cycle days (within ±2 days) across at least 3 distinct cycles.
- **Method:** The [clustering algorithm](../engine/internal/insights/generate.go#L186) is the same approach used for symptom predictions (see above), with cluster detection in [`findSymptomCenterDay`](../engine/internal/insights/generate.go#L294). When a recurring pattern is found, the insight describes what symptom tends to occur and around which cycle day.
- **Example output:** "Headaches tend to occur around cycle day 12."

### Medication adherence pattern

- **Requires:** At least 1 active medication with at least 14 days of logged medication events.
- **Method:** For each active medication, [`medicationAdherencePattern`](../engine/internal/insights/generate.go#L332) computes:
  1. Count the unique calendar dates on which you logged at least one event.
  2. Divide by the total number of days from your first logged event to your most recent one.
  3. Classify adherence:
     - **High:** 90% or above
     - **Moderate:** 70–89%
     - **Low:** Below 70%
- **Example output:** "Your adherence to Ibuprofen has been high at 95%."

### Bleeding pattern

- **Requires:** At least 3 completed cycles with at least one bleeding observation per cycle.
- **Method:** For each cycle, [`bleedingPattern`](../engine/internal/insights/generate.go#L430) computes:
  - **Bleed duration:** The number of consecutive bleeding days from the start of the cycle.
  - **Average flow intensity:** Each flow level is [scored numerically](../engine/internal/insights/generate.go#L576) (Spotting = 1, Light = 2, Medium = 3, Heavy = 4) and the mean score is computed across all bleed days.
- Both duration and flow are analyzed for trends using linear regression:
  - Slopes greater than ±0.1 indicate a shortening or lengthening trend; otherwise the metric is considered stable.
- **Example output:** "Your period duration has been stable at around 5 days, but flow has been getting lighter."

---

## 6. Confidence Levels

> Source: [`ComputeConfidence`](../engine/internal/rules/phases.go#L169) in [`engine/internal/rules/phases.go`](../engine/internal/rules/phases.go)

Every phase estimate, prediction, and insight is assigned a confidence level that reflects how much data supports it. Confidence is determined by the number of completed cycles and your regularity profile:

| Condition                                      | Confidence |
| ---------------------------------------------- | ---------- |
| Fewer than 2 completed cycles                  | **Low**    |
| 2–4 completed cycles with regular cycles       | **Medium** |
| 5 or more completed cycles with regular cycles | **High**   |

Confidence may be capped lower based on your profile settings:

| Profile setting           | Maximum confidence |
| ------------------------- | ------------------ |
| Very Irregular cycles     | Low                |
| Somewhat Irregular cycles | Medium             |
| Irregular cycle model     | Medium             |

The lowest applicable cap always wins. For example, if you have 5+ completed cycles (which would normally be High confidence) but your regularity is set to Somewhat Irregular, confidence is capped at Medium.

For the Irregular cycle model, ovulation window phase estimates are [always assigned Low confidence](../engine/internal/rules/phases.go#L72) regardless of cycle count.

---

## 7. When Results Are Recalculated

> Source: [`redetectAndStoreCycles`](../engine/internal/service/cycles.go#L170) and [`regenerateAndStoreInsights`](../engine/internal/service/cycles.go#L275) in [`engine/internal/service/cycles.go`](../engine/internal/service/cycles.go)

Cycle detection, predictions, and insights are **automatically recalculated** whenever:

- You [log a new bleeding observation](../engine/internal/service/observations.go#L14) that starts a new cycle.
- You manually confirm or modify a cycle boundary.
- You [update your biological cycle model or cycle regularity](../engine/internal/service/profiles.go#L104) in your profile.
- You log a new symptom observation or medication event (for insights).
- You [activate or deactivate a medication](../engine/internal/service/medications.go#L249) (for insights).

When recalculation is triggered, all existing predictions and insights for your profile are cleared and regenerated from scratch using the latest data. This ensures your results always reflect the most current information.
