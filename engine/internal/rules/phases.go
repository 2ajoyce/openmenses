package rules

import (
	"time"

	"github.com/oklog/ulid/v2"

	v1 "github.com/2ajoyce/opencycle/gen/go/opencycle/v1"
)

// defaultCycleLen is assumed when no historical average is available.
const defaultCycleLen = 28

// phaseFn maps a 1-indexed cycle day number to a CyclePhase.
type phaseFn func(dayNum int) v1.CyclePhase

// EstimatePhases generates one PhaseEstimate per calendar day for the given
// cycle, using the biological cycle model from the user profile.
//
// avgCycleLen is the average history-based cycle length (0 → 28-day default).
// completedCycles is the number of completed cycles available, used to assign
// confidence per domain rules §2.4.
//
// Open-ended cycles (no end_date) are estimated up to
// start_date + avgCycleLen - 1.
func EstimatePhases(
	cycle *v1.Cycle,
	profile *v1.UserProfile,
	avgCycleLen int,
	completedCycles int,
) []*v1.PhaseEstimate {
	start := cycle.GetStartDate().GetValue()
	if start == "" {
		return nil
	}

	if avgCycleLen <= 0 {
		avgCycleLen = defaultCycleLen
	}

	startTime, err := time.Parse("2006-01-02", start)
	if err != nil {
		return nil
	}

	var endTime time.Time
	if end := cycle.GetEndDate().GetValue(); end != "" {
		et, err2 := time.Parse("2006-01-02", end)
		if err2 != nil {
			return nil
		}
		endTime = et
	} else {
		endTime = startTime.AddDate(0, 0, avgCycleLen-1)
	}

	confidence := computeConfidence(completedCycles, profile)
	fn := ovulatoryPhaseFn(avgCycleLen)
	if profile.GetBiologicalCycle() == v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED {
		fn = suppressedPhaseFn()
	}

	entropy := ulid.DefaultEntropy()
	var estimates []*v1.PhaseEstimate
	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		dayNum := int(d.Sub(startTime).Hours()/24) + 1
		estimates = append(estimates, &v1.PhaseEstimate{
			Id:         ulid.MustNew(ulid.Now(), entropy).String(),
			UserId:     cycle.GetUserId(),
			Date:       &v1.LocalDate{Value: d.Format("2006-01-02")},
			Phase:      fn(dayNum),
			Confidence: confidence,
		})
	}
	return estimates
}

// ovulatoryPhaseFn returns a phase function for the ovulatory model.
// Ovulation day O = cycleLen - 14. Phases:
//   - Days 1–5:        MENSTRUATION
//   - Days 6–(O-2):   FOLLICULAR
//   - Days (O-1)–(O+1): OVULATION_WINDOW
//   - Days (O+2)–end: LUTEAL
//
// The irregular model reuses this function; confidence capping is applied in
// computeConfidence rather than modifying phase boundaries.
func ovulatoryPhaseFn(cycleLen int) phaseFn {
	o := cycleLen - 14
	if o < 6 {
		o = 6 // safety clamp: ovulation cannot precede follicular phase
	}
	return func(dayNum int) v1.CyclePhase {
		switch {
		case dayNum <= 5:
			return v1.CyclePhase_CYCLE_PHASE_MENSTRUATION
		case dayNum <= o-2:
			return v1.CyclePhase_CYCLE_PHASE_FOLLICULAR
		case dayNum <= o+1:
			return v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW
		default:
			return v1.CyclePhase_CYCLE_PHASE_LUTEAL
		}
	}
}

// suppressedPhaseFn returns a phase function for the hormonally-suppressed
// model: days 1–5 are menstruation (withdrawal bleed); all remaining days
// are stored as FOLLICULAR (displayed by the UI as "Pill-free / Active pill
// days" per domain rules §2.2).
func suppressedPhaseFn() phaseFn {
	return func(dayNum int) v1.CyclePhase {
		if dayNum <= 5 {
			return v1.CyclePhase_CYCLE_PHASE_MENSTRUATION
		}
		return v1.CyclePhase_CYCLE_PHASE_FOLLICULAR
	}
}

// computeConfidence computes the ConfidenceLevel for phase estimates per
// domain rules §2.4. The lowest applicable cap wins.
func computeConfidence(completedCycles int, profile *v1.UserProfile) v1.ConfidenceLevel {
	var base v1.ConfidenceLevel
	switch {
	case completedCycles < 2:
		base = v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW
	case completedCycles < 5:
		base = v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM
	default:
		base = v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH
	}

	// VERY_IRREGULAR caps at LOW.
	if profile.GetCycleRegularity() == v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR {
		if base > v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW {
			base = v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW
		}
	} else if profile.GetCycleRegularity() == v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR {
		// SOMEWHAT_IRREGULAR caps at MEDIUM.
		if base > v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM {
			base = v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM
		}
	}

	// IRREGULAR model caps at MEDIUM.
	if profile.GetBiologicalCycle() == v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR {
		if base > v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM {
			base = v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM
		}
	}

	return base
}
