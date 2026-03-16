package predictions

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// Generate produces forward-looking predictions for the given user based on
// completed cycles, symptoms, and profile settings.
//
// Returns nil if insufficient completed cycles exist (per domain rules §2).
func Generate(
	userID string,
	cycles []*v1.Cycle,
	symptoms []*v1.SymptomObservation,
	profile *v1.UserProfile,
) []*v1.Prediction {
	stats := rules.Stats(cycles)
	completed := rules.CompletedCycles(cycles)

	// No valid predictions without completed cycles.
	if stats.Average == 0 {
		return nil
	}

	// Use the count of non-outlier completed cycles from stats.
	completedCount := stats.Count
	avgLen := int(math.Round(stats.Average))

	// Find anchor cycle: open-ended cycle OR last completed cycle.
	var anchorStart time.Time
	openCycle := findOpenCycle(cycles)
	if openCycle != nil {
		t, err := time.Parse("2006-01-02", openCycle.GetStartDate().GetValue())
		if err == nil {
			anchorStart = t
		}
	}
	if anchorStart.IsZero() && len(completed) > 0 {
		// Fall back to last completed cycle start + avgLen.
		sort.Slice(completed, func(i, j int) bool {
			return completed[i].GetStartDate().GetValue() < completed[j].GetStartDate().GetValue()
		})
		last := completed[len(completed)-1]
		t, err := time.Parse("2006-01-02", last.GetStartDate().GetValue())
		if err == nil {
			anchorStart = t.AddDate(0, 0, avgLen)
		}
	}
	if anchorStart.IsZero() {
		return nil
	}

	nextBleedStart := anchorStart
	confidence := rules.ComputeConfidence(completedCount, profile)

	var predictions []*v1.Prediction

	// NEXT_BLEED: requires ≥2 completed cycles.
	if completedCount >= 2 {
		pred := predictNextBleed(userID, nextBleedStart, confidence, avgLen, completedCount)
		predictions = append(predictions, pred)
	}

	// PMS_WINDOW: requires ≥2 completed cycles.
	if completedCount >= 2 {
		pred := predictPMSWindow(userID, nextBleedStart, confidence, avgLen, completedCount)
		predictions = append(predictions, pred)
	}

	// OVULATION_WINDOW: requires ≥3 completed cycles AND (REGULAR or SOMEWHAT_IRREGULAR)
	// AND NOT (HORMONALLY_SUPPRESSED or IRREGULAR).
	if completedCount >= 3 && canPredictOvulation(profile) {
		pred := predictOvulationWindow(userID, nextBleedStart, confidence, avgLen)
		predictions = append(predictions, pred)
	}

	// SYMPTOM_WINDOWS: requires ≥3 completed cycles.
	if completedCount >= 3 && len(symptoms) > 0 {
		symptomPreds := predictSymptomWindows(
			userID, completed, symptoms, nextBleedStart, completedCount, profile,
		)
		predictions = append(predictions, symptomPreds...)
	}

	return predictions
}

// findOpenCycle returns the first open-ended cycle (no end_date).
func findOpenCycle(cycles []*v1.Cycle) *v1.Cycle {
	for _, c := range cycles {
		if c.GetEndDate().GetValue() == "" {
			return c
		}
	}
	return nil
}

// canPredictOvulation checks if profile allows ovulation window predictions.
// Returns true if regularity is REGULAR or SOMEWHAT_IRREGULAR AND
// biological cycle is NOT HORMONALLY_SUPPRESSED or IRREGULAR.
func canPredictOvulation(profile *v1.UserProfile) bool {
	regularity := profile.GetCycleRegularity()
	if regularity != v1.CycleRegularity_CYCLE_REGULARITY_REGULAR &&
		regularity != v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR {
		return false
	}

	bioCycle := profile.GetBiologicalCycle()
	if bioCycle == v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED ||
		bioCycle == v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR {
		return false
	}

	return true
}

// predictNextBleed creates a NEXT_BLEED prediction.
func predictNextBleed(
	userID string,
	nextBleedStart time.Time,
	confidence v1.ConfidenceLevel,
	avgLen int,
	completedCount int,
) *v1.Prediction {
	startDate := nextBleedStart.Format("2006-01-02")
	endDate := nextBleedStart.AddDate(0, 0, 5).Format("2006-01-02")

	return &v1.Prediction{
		Name:               ulid.Make().String(),
		UserId:             userID,
		Kind:               v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
		PredictedStartDate: &v1.LocalDate{Value: startDate},
		PredictedEndDate:   &v1.LocalDate{Value: endDate},
		Confidence:         confidence,
		Rationale: []string{
			fmt.Sprintf("Based on %d completed cycles with average length %d days",
				completedCount, avgLen),
		},
	}
}

// predictPMSWindow creates a PMS_WINDOW prediction.
func predictPMSWindow(
	userID string,
	nextBleedStart time.Time,
	confidence v1.ConfidenceLevel,
	avgLen int,
	completedCount int,
) *v1.Prediction {
	startDate := nextBleedStart.AddDate(0, 0, -10).Format("2006-01-02")
	endDate := nextBleedStart.AddDate(0, 0, -1).Format("2006-01-02")

	return &v1.Prediction{
		Name:               ulid.Make().String(),
		UserId:             userID,
		Kind:               v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
		PredictedStartDate: &v1.LocalDate{Value: startDate},
		PredictedEndDate:   &v1.LocalDate{Value: endDate},
		Confidence:         confidence,
		Rationale: []string{
			fmt.Sprintf("Based on %d completed cycles with average length %d days",
				completedCount, avgLen),
		},
	}
}

// predictOvulationWindow creates an OVULATION_WINDOW prediction.
func predictOvulationWindow(
	userID string,
	nextBleedStart time.Time,
	confidence v1.ConfidenceLevel,
	avgLen int,
) *v1.Prediction {
	// O = max(avgLen - 14, 6) — ovulation day relative to cycle start.
	O := avgLen - 14
	if O < 6 {
		O = 6
	}

	// Predicted window is ±1 day around O.
	startDate := nextBleedStart.AddDate(0, 0, O-2).Format("2006-01-02")
	endDate := nextBleedStart.AddDate(0, 0, O).Format("2006-01-02")

	return &v1.Prediction{
		Name:               ulid.Make().String(),
		UserId:             userID,
		Kind:               v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW,
		PredictedStartDate: &v1.LocalDate{Value: startDate},
		PredictedEndDate:   &v1.LocalDate{Value: endDate},
		Confidence:         confidence,
		Rationale: []string{
			fmt.Sprintf("Ovulation predicted on day %d of average %d-day cycle",
				O, avgLen),
		},
	}
}

// predictSymptomWindows analyzes symptom observations and creates SYMPTOM_WINDOW
// predictions for frequently observed symptoms on consistent cycle days.
func predictSymptomWindows(
	userID string,
	completed []*v1.Cycle,
	symptoms []*v1.SymptomObservation,
	nextBleedStart time.Time,
	completedCount int,
	profile *v1.UserProfile,
) []*v1.Prediction {
	// Build cycle date-range lookup.
	type cycleRange struct {
		startTime time.Time
		endTime   time.Time
		index     int
	}
	var cycleRanges []cycleRange
	for i, c := range completed {
		startStr := c.GetStartDate().GetValue()
		endStr := c.GetEndDate().GetValue()
		startTime, err1 := time.Parse("2006-01-02", startStr)
		endTime, err2 := time.Parse("2006-01-02", endStr)
		if err1 != nil || err2 != nil {
			continue
		}
		cycleRanges = append(cycleRanges, cycleRange{startTime, endTime, i})
	}

	// Group observations by symptom type.
	symptomObs := make(map[v1.SymptomType][]struct {
		cycleIndex       int
		relativeCycleDay int
	})

	for _, obs := range symptoms {
		ts := obs.GetTimestamp().GetValue()
		if len(ts) < 10 {
			continue
		}
		dateStr := ts[:10]
		obsTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Find containing cycle.
		var cycleIdx int
		found := false
		for _, cr := range cycleRanges {
			if !obsTime.Before(cr.startTime) && !obsTime.After(cr.endTime) {
				cycleIdx = cr.index
				found = true
				break
			}
		}
		if !found {
			continue
		}

		// Compute relative cycle day (1-indexed).
		cr := cycleRanges[cycleIdx]
		relDay := int(obsTime.Sub(cr.startTime).Hours()/24) + 1

		symptomType := obs.GetSymptom()
		symptomObs[symptomType] = append(symptomObs[symptomType], struct {
			cycleIndex       int
			relativeCycleDay int
		}{cycleIdx, relDay})
	}

	// Find qualifying symptoms: ≥3 observations from ≥3 distinct cycles on similar days.
	confidence := rules.ComputeConfidence(completedCount, profile)
	var predictions []*v1.Prediction

	for symptomType, obs := range symptomObs {
		if len(obs) < 3 {
			continue
		}

		// Find "center day" using sliding window: ≥3 observations within ±2 of center,
		// from ≥3 distinct cycles.
		centerDay := findSymptomCenterDay(obs)
		if centerDay == 0 {
			continue
		}

		// Create prediction relative to nextBleedStart.
		// centerDay is relative to some completed cycle; compute relative to next cycle.
		startDate := nextBleedStart.AddDate(0, 0, centerDay-3).Format("2006-01-02")
		endDate := nextBleedStart.AddDate(0, 0, centerDay+1).Format("2006-01-02")

		pred := &v1.Prediction{
			Name:               ulid.Make().String(),
			UserId:             userID,
			Kind:               v1.PredictionType_PREDICTION_TYPE_SYMPTOM_WINDOW,
			PredictedStartDate: &v1.LocalDate{Value: startDate},
			PredictedEndDate:   &v1.LocalDate{Value: endDate},
			Confidence:         confidence,
			Rationale: []string{
				fmt.Sprintf("Observed %s on cycle day %d across %d cycles",
					symptomType.String(), centerDay, len(obs)),
			},
		}
		predictions = append(predictions, pred)
	}

	return predictions
}

// findSymptomCenterDay finds a cycle day where ≥3 observations cluster within ±2 days,
// from ≥3 distinct cycles. Returns 0 if no such cluster exists.
func findSymptomCenterDay(obs []struct {
	cycleIndex       int
	relativeCycleDay int
}) int {
	// Sort observations by relative cycle day.
	sort.Slice(obs, func(i, j int) bool {
		return obs[i].relativeCycleDay < obs[j].relativeCycleDay
	})

	// Sliding window: check all groups of consecutive days within ±2 range.
	for i := 0; i < len(obs); i++ {
		// Try center at each observed day.
		for j := i; j < len(obs); j++ {
			center := obs[j].relativeCycleDay
			var inWindow []int
			cycleSet := make(map[int]struct{})

			for _, ob := range obs {
				// Within ±2 of center.
				if ob.relativeCycleDay >= center-2 && ob.relativeCycleDay <= center+2 {
					inWindow = append(inWindow, ob.cycleIndex)
					cycleSet[ob.cycleIndex] = struct{}{}
				}
			}

			// Check: ≥3 observations from ≥3 distinct cycles.
			if len(inWindow) >= 3 && len(cycleSet) >= 3 {
				return center
			}
		}
	}

	return 0
}
