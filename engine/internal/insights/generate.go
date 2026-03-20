package insights

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// Generate produces backward-looking insights for the given user based on
// completed cycles, observations, and profile settings.
//
// Returns nil if insufficient data exists or profile is incomplete (per domain rules §6.9).
func Generate(
	userID string,
	cycles []*v1.Cycle,
	symptoms []*v1.SymptomObservation,
	medications []*v1.Medication,
	medicationEvents []*v1.MedicationEvent,
	bleedingObs []*v1.BleedingObservation,
	profile *v1.UserProfile,
) []*v1.Insight {
	// Profile completeness check (§6.9).
	if profile == nil || !isProfileComplete(profile) {
		return nil
	}

	completed := rules.CompletedCycles(cycles)
	stats := rules.Stats(cycles)
	confidence := rules.ComputeConfidence(len(completed), profile)

	var insights []*v1.Insight

	// CYCLE_LENGTH_PATTERN: ≥ 3 completed non-outlier cycles (§6.2).
	if stats.Count >= 3 {
		if ins := cycleLengthPattern(userID, completed, stats); ins != nil {
			insights = append(insights, ins)
		}
	}

	// SYMPTOM_PATTERN: ≥ 3 completed cycles with ≥ 3 matching observations (§6.3).
	if len(completed) >= 3 && len(symptoms) > 0 {
		symptomInsights := symptomPattern(userID, completed, symptoms, len(completed))
		insights = append(insights, symptomInsights...)
	}

	// MEDICATION_ADHERENCE_PATTERN: ≥ 1 active medication with ≥ 14 days of events (§6.4).
	if len(medications) > 0 && len(medicationEvents) > 0 {
		adherenceInsights := medicationAdherencePattern(userID, medications, medicationEvents)
		insights = append(insights, adherenceInsights...)
	}

	// BLEEDING_PATTERN: ≥ 3 completed cycles with bleeding data (§6.5).
	if len(completed) >= 3 && len(bleedingObs) > 0 {
		if ins := bleedingPattern(userID, completed, bleedingObs); ins != nil {
			insights = append(insights, ins)
		}
	}

	// Apply computed confidence to all insights (§6.6).
	for _, ins := range insights {
		ins.Confidence = confidence
	}

	return insights
}

// isProfileComplete checks if profile has required fields for insight generation (§6.9).
func isProfileComplete(profile *v1.UserProfile) bool {
	if profile == nil {
		return false
	}
	// biological_cycle and cycle_regularity must be non-UNSPECIFIED.
	// tracking_focus must have at least one value (enforced by proto).
	bioCycle := profile.GetBiologicalCycle()
	regularity := profile.GetCycleRegularity()
	focus := profile.GetTrackingFocus()

	return bioCycle != v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_UNSPECIFIED &&
		regularity != v1.CycleRegularity_CYCLE_REGULARITY_UNSPECIFIED &&
		len(focus) > 0
}

// cycleLengthPattern analyzes cycle length trends via linear regression.
// Requires ≥ 3 completed non-outlier cycles (§6.2).
func cycleLengthPattern(
	userID string,
	completed []*v1.Cycle,
	stats rules.CycleStats,
) *v1.Insight {
	if stats.Count < 3 {
		return nil
	}

	// Collect non-outlier cycles in chronological order.
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].GetStartDate().GetValue() < completed[j].GetStartDate().GetValue()
	})

	var lengths []int
	var cycleRefs []*v1.RecordRef
	for _, c := range completed {
		if len := rules.CycleLength(c); len > 0 && !rules.IsOutlierLength(c) {
			lengths = append(lengths, len)
			cycleRefs = append(cycleRefs, &v1.RecordRef{Name: c.GetName()})
		}
	}

	if len(lengths) < 3 {
		return nil
	}

	// Perform linear regression: y = mx + b.
	slope := computeSlope(lengths)
	cv := stats.StdDev / stats.Average

	var patternType string
	switch {
	case math.Abs(slope) > 0.5 && slope < 0:
		patternType = "SHORTENING"
	case slope > 0.5:
		patternType = "LENGTHENING"
	case cv < 0.08:
		patternType = "STABLE"
	default:
		patternType = "IRREGULAR"
	}

	summary := generateCycleLengthSummary(patternType, int(math.Round(stats.Average)), len(lengths))

	return &v1.Insight{
		Name:               ulid.Make().String(),
		UserId:             userID,
		Kind:               v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN,
		Summary:            summary,
		EvidenceRecordRefs: cycleRefs,
		Confidence:         v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM, // Overridden in Generate()
	}
}

// computeSlope performs simple linear regression and returns the slope (m).
// x is cycle index (0, 1, 2, ...), y is cycle length.
func computeSlope(lengths []int) float64 {
	n := float64(len(lengths))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, len := range lengths {
		x := float64(i)
		y := float64(len)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	return slope
}

// generateCycleLengthSummary creates a human-readable summary for cycle length patterns.
func generateCycleLengthSummary(patternType string, avgLen int, count int) string {
	switch patternType {
	case "SHORTENING":
		return fmt.Sprintf("Your cycle length has been gradually decreasing over the last %d cycles, currently around %d days", count, avgLen)
	case "LENGTHENING":
		return fmt.Sprintf("Your cycle length has been gradually increasing over the last %d cycles, currently around %d days", count, avgLen)
	case "STABLE":
		return fmt.Sprintf("Your cycle length has remained stable at around %d days", avgLen)
	default:
		return fmt.Sprintf("Your cycle length has been irregular, ranging around %d days on average", avgLen)
	}
}

// symptomPattern analyzes symptom observations for recurring patterns on similar cycle days.
// Requires ≥ 3 completed cycles with ≥ 3 matching observations (§6.3).
func symptomPattern(
	userID string,
	completed []*v1.Cycle,
	symptoms []*v1.SymptomObservation,
	_ int, // completedCount (unused, kept for consistency with predictions pattern)
) []*v1.Insight {
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

	// Group symptom observations by type.
	symptomObs := make(map[v1.SymptomType][]struct {
		cycleIndex       int
		relativeCycleDay int
		name             string
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
			name             string
		}{cycleIdx, relDay, obs.GetName()})
	}

	// Find qualifying symptoms: ≥3 observations from ≥3 distinct cycles on similar days.
	var insights []*v1.Insight

	for symptomType, obs := range symptomObs {
		if len(obs) < 3 {
			continue
		}

		// Find center day using same algorithm as predictions.
		centerDay := findSymptomCenterDay(obs)
		if centerDay == 0 {
			continue
		}

		// Collect evidence refs for observations within ±2 of centerDay.
		var evidenceRefs []*v1.RecordRef
		for _, o := range obs {
			if o.relativeCycleDay >= centerDay-2 && o.relativeCycleDay <= centerDay+2 {
				evidenceRefs = append(evidenceRefs, &v1.RecordRef{Name: o.name})
			}
		}

		summary := fmt.Sprintf("%s tends to occur around cycle day %d", symptomType.String(), centerDay)

		insight := &v1.Insight{
			Name:               ulid.Make().String(),
			UserId:             userID,
			Kind:               v1.InsightType_INSIGHT_TYPE_SYMPTOM_PATTERN,
			Summary:            summary,
			EvidenceRecordRefs: evidenceRefs,
			Confidence:         v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM, // Overridden in Generate()
		}
		insights = append(insights, insight)
	}

	return insights
}

// findSymptomCenterDay finds a cycle day where ≥3 observations cluster within ±2 days,
// from ≥3 distinct cycles. Returns 0 if no such cluster exists.
func findSymptomCenterDay(obs []struct {
	cycleIndex       int
	relativeCycleDay int
	name             string
}) int {
	// Sort observations by relative cycle day.
	sort.Slice(obs, func(i, j int) bool {
		return obs[i].relativeCycleDay < obs[j].relativeCycleDay
	})

	// Sliding window: check all groups of consecutive days within ±2 range.
	for i := range len(obs) {
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

// medicationAdherencePattern analyzes adherence for active medications.
// Requires ≥ 1 active medication with ≥ 14 days of events (§6.4).
func medicationAdherencePattern(
	userID string,
	medications []*v1.Medication,
	medicationEvents []*v1.MedicationEvent,
) []*v1.Insight {
	var insights []*v1.Insight

	for _, med := range medications {
		// Only process active medications.
		if !med.GetActive() {
			continue
		}

		// Collect events for this medication.
		var medEvents []*v1.MedicationEvent
		for _, evt := range medicationEvents {
			if evt.GetMedicationId() == med.GetName() {
				medEvents = append(medEvents, evt)
			}
		}

		if len(medEvents) == 0 {
			continue
		}

		// Count unique dates with at least one event.
		dateSet := make(map[string]struct{})
		var earliestTime time.Time
		var latestTime time.Time

		for _, evt := range medEvents {
			ts := evt.GetTimestamp().GetValue()
			if len(ts) < 10 {
				continue
			}
			dateStr := ts[:10]
			dateSet[dateStr] = struct{}{}

			evtTime, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			if earliestTime.IsZero() || evtTime.Before(earliestTime) {
				earliestTime = evtTime
			}
			if evtTime.After(latestTime) {
				latestTime = evtTime
			}
		}

		if earliestTime.IsZero() || latestTime.IsZero() {
			continue
		}

		// Compute adherence ratio.
		daysWithEvents := len(dateSet)
		totalDays := int(latestTime.Sub(earliestTime).Hours()/24) + 1
		if totalDays < 14 {
			continue // Eligibility threshold: ≥ 14 days of data.
		}

		adherenceRatio := float64(daysWithEvents) / float64(totalDays)
		adherencePercent := int(math.Round(adherenceRatio * 100))

		// Classify adherence.
		var adherenceType string
		switch {
		case adherenceRatio >= 0.90:
			adherenceType = "HIGH"
		case adherenceRatio >= 0.70:
			adherenceType = "MODERATE"
		default:
			adherenceType = "LOW"
		}

		summary := fmt.Sprintf("Your adherence to %s has been %s at %d%%", med.GetDisplayName(), adherenceType, adherencePercent)

		var eventRefs []*v1.RecordRef
		for _, evt := range medEvents {
			eventRefs = append(eventRefs, &v1.RecordRef{Name: evt.GetName()})
		}

		insight := &v1.Insight{
			Name:               ulid.Make().String(),
			UserId:             userID,
			Kind:               v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN,
			Summary:            summary,
			EvidenceRecordRefs: eventRefs,
			Confidence:         v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM, // Overridden in Generate()
		}
		insights = append(insights, insight)
	}

	return insights
}

// bleedingPattern analyzes bleed duration and flow trends.
// Requires ≥ 3 completed cycles with bleeding data (§6.5).
func bleedingPattern(
	userID string,
	completed []*v1.Cycle,
	bleedingObs []*v1.BleedingObservation,
) *v1.Insight {
	// Sort cycles chronologically.
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].GetStartDate().GetValue() < completed[j].GetStartDate().GetValue()
	})

	// Build cycle date-range lookup.
	type cycleRange struct {
		startTime time.Time
		endTime   time.Time
		cycleIdx  int
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

	// Group observations by cycle.
	type cycleBleedData struct {
		dates    []time.Time
		flows    []v1.BleedingFlow
		obsNames []string
	}
	cycleData := make(map[int]*cycleBleedData)

	for _, obs := range bleedingObs {
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
				cycleIdx = cr.cycleIdx
				found = true
				break
			}
		}
		if !found {
			continue
		}

		if cycleData[cycleIdx] == nil {
			cycleData[cycleIdx] = &cycleBleedData{}
		}
		cycleData[cycleIdx].dates = append(cycleData[cycleIdx].dates, obsTime)
		cycleData[cycleIdx].flows = append(cycleData[cycleIdx].flows, obs.GetFlow())
		cycleData[cycleIdx].obsNames = append(cycleData[cycleIdx].obsNames, obs.GetName())
	}

	if len(cycleData) < 3 {
		return nil // Eligibility threshold: ≥ 3 cycles.
	}

	// Compute bleed duration and flow intensity per cycle.
	var durations []int
	var flowIntensities []float64
	var allEvidenceRefs []*v1.RecordRef

	for cycleIdx := range len(completed) {
		data, ok := cycleData[cycleIdx]
		if !ok || len(data.dates) == 0 {
			continue
		}

		// Sort dates to find consecutive bleeding days.
		sort.Slice(data.dates, func(i, j int) bool {
			return data.dates[i].Before(data.dates[j])
		})

		// Duration: consecutive days from start of cycle (first bleed).
		cycleStart := cycleRanges[cycleIdx].startTime
		duration := int(data.dates[0].Sub(cycleStart).Hours()/24) + 1

		// Find how many consecutive days from the first bleed date.
		lastBleedDate := data.dates[0]
		for i := 1; i < len(data.dates); i++ {
			// If gap > 1 day, break.
			if int(data.dates[i].Sub(lastBleedDate).Hours()/24) > 1 {
				break
			}
			lastBleedDate = data.dates[i]
			duration++
		}

		durations = append(durations, duration)

		// Flow intensity: numeric score per observation.
		flowScore := 0.0
		for _, flow := range data.flows {
			flowScore += float64(flowToScore(flow))
		}
		avgFlow := flowScore / float64(len(data.flows))
		flowIntensities = append(flowIntensities, avgFlow)

		for _, name := range data.obsNames {
			allEvidenceRefs = append(allEvidenceRefs, &v1.RecordRef{Name: name})
		}
	}

	if len(durations) < 3 {
		return nil
	}

	// Analyze trends.
	durationTrend := analyzeTrendInt(durations)
	flowTrend := analyzeTrend(flowIntensities)

	avgDuration := 0
	for _, d := range durations {
		avgDuration += d
	}
	avgDuration /= len(durations)

	summary := generateBleedingSummary(durationTrend, flowTrend, avgDuration)

	return &v1.Insight{
		Name:               ulid.Make().String(),
		UserId:             userID,
		Kind:               v1.InsightType_INSIGHT_TYPE_BLEEDING_PATTERN,
		Summary:            summary,
		EvidenceRecordRefs: allEvidenceRefs,
		Confidence:         v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM, // Overridden in Generate()
	}
}

// flowToScore converts BleedingFlow to a numeric score.
func flowToScore(flow v1.BleedingFlow) float64 {
	switch flow {
	case v1.BleedingFlow_BLEEDING_FLOW_SPOTTING:
		return 1
	case v1.BleedingFlow_BLEEDING_FLOW_LIGHT:
		return 2
	case v1.BleedingFlow_BLEEDING_FLOW_MEDIUM:
		return 3
	case v1.BleedingFlow_BLEEDING_FLOW_HEAVY:
		return 4
	default:
		return 0
	}
}

// analyzeTrendInt performs linear regression on an integer slice and returns trend type.
func analyzeTrendInt(values []int) string {
	if len(values) < 2 {
		return "STABLE"
	}

	floatVals := make([]float64, len(values))
	for i, v := range values {
		floatVals[i] = float64(v)
	}

	slope := computeSlopeFloat(floatVals)

	if math.Abs(slope) > 0.1 {
		if slope < 0 {
			return "SHORTENING"
		}
		return "LENGTHENING"
	}
	return "STABLE"
}

// analyzeTrend performs linear regression on a float slice and returns trend type.
func analyzeTrend(values []float64) string {
	if len(values) < 2 {
		return "STABLE"
	}

	slope := computeSlopeFloat(values)

	if math.Abs(slope) > 0.1 {
		if slope < 0 {
			return "SHORTENING"
		}
		return "LENGTHENING"
	}
	return "STABLE"
}

// computeSlopeFloat performs linear regression on float values.
func computeSlopeFloat(values []float64) float64 {
	n := float64(len(values))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}

	slope := (n*sumXY - sumX*sumY) / denom
	return slope
}

// generateBleedingSummary creates a human-readable summary for bleeding patterns.
func generateBleedingSummary(durationTrend, flowTrend string, avgDuration int) string {
	switch {
	case durationTrend == "STABLE" && flowTrend == "STABLE":
		return fmt.Sprintf("Your period duration has been stable at around %d days with consistent flow", avgDuration)
	case durationTrend == "STABLE" && flowTrend == "SHORTENING":
		return fmt.Sprintf("Your period duration has been stable at around %d days, but flow has been getting lighter", avgDuration)
	case durationTrend == "STABLE" && flowTrend == "LENGTHENING":
		return fmt.Sprintf("Your period duration has been stable at around %d days, but flow has been getting heavier", avgDuration)
	case durationTrend == "SHORTENING":
		return fmt.Sprintf("Your period duration has been getting shorter, averaging %d days recently", avgDuration)
	case durationTrend == "LENGTHENING":
		return fmt.Sprintf("Your period duration has been getting longer, averaging %d days recently", avgDuration)
	default:
		return fmt.Sprintf("Your period pattern has been variable, averaging around %d days", avgDuration)
	}
}
