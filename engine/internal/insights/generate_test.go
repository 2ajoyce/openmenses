package insights_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/insights"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// Helper to create a complete user profile.
func makeProfile() *v1.UserProfile {
	return &v1.UserProfile{
		Name:             "test-profile",
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_PATTERN_ANALYSIS},
	}
}

// Helper to create a cycle with start and end dates.
func makeCycle(name, userID, startDate, endDate string) *v1.Cycle {
	return &v1.Cycle{
		Name:      name,
		UserId:    userID,
		StartDate: &v1.LocalDate{Value: startDate},
		EndDate:   &v1.LocalDate{Value: endDate},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
}

// Helper to create a symptom observation.
func makeSymptomObs(name, userID, timestamp string, symptomType v1.SymptomType) *v1.SymptomObservation {
	return &v1.SymptomObservation{
		Name:      name,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: timestamp + "T00:00:00Z"},
		Symptom:   symptomType,
	}
}

// Helper to create a medication.
func makeMedication(name, userID string, active bool) *v1.Medication {
	return &v1.Medication{
		Name:        name,
		UserId:      userID,
		DisplayName: name,
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      active,
	}
}

// Helper to create a medication event.
func makeMedicationEvent(name, userID, medicationID, timestamp string) *v1.MedicationEvent {
	return &v1.MedicationEvent{
		Name:         name,
		UserId:       userID,
		MedicationId: medicationID,
		Timestamp:    &v1.DateTime{Value: timestamp + "T00:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
		Dose:         "1 tablet",
	}
}

// Helper to create a bleeding observation.
func makeBleedingObs(name, userID, timestamp string, flow v1.BleedingFlow) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Name:      name,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: timestamp + "T00:00:00Z"},
		Flow:      flow,
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestGenerateNoCycles: No cycles → returns empty.
func TestGenerateNoCycles(t *testing.T) {
	profile := makeProfile()
	insights := insights.Generate("u1", nil, nil, nil, nil, nil, profile)
	if len(insights) != 0 {
		t.Errorf("expected 0 insights, got %d", len(insights))
	}
}

// TestGenerateNoProfile: No profile → returns nil.
func TestGenerateNoProfile(t *testing.T) {
	cycles := []*v1.Cycle{makeCycle("c1", "u1", "2026-01-01", "2026-01-28")}
	insights := insights.Generate("u1", cycles, nil, nil, nil, nil, nil)
	if insights != nil {
		t.Errorf("expected nil, got %v", insights)
	}
}

// TestGenerateIncompleteProfile: Profile missing required fields → returns nil.
func TestGenerateIncompleteProfile(t *testing.T) {
	// Profile without BiologicalCycleModel
	profile := &v1.UserProfile{
		Name:             "incomplete",
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_UNSPECIFIED,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_PATTERN_ANALYSIS},
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
	}
	cycles := []*v1.Cycle{makeCycle("c1", "u1", "2026-01-01", "2026-01-28")}
	insights := insights.Generate("u1", cycles, nil, nil, nil, nil, profile)
	if insights != nil {
		t.Errorf("expected nil, got %v", insights)
	}
}

// TestCycleLengthPatternTooFewCycles: 1–2 completed cycles → no CYCLE_LENGTH_PATTERN.
func TestCycleLengthPatternTooFewCycles(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
	}
	insights := insights.Generate("u1", cycles, nil, nil, nil, nil, profile)

	for _, ins := range insights {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN {
			t.Errorf("expected no CYCLE_LENGTH_PATTERN, got one")
		}
	}
}

// TestCycleLengthPatternDecreasing: 3+ cycles with decreasing lengths → SHORTENING pattern.
func TestCycleLengthPatternDecreasing(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-31"), // 31 days
		makeCycle("c2", "u1", "2026-02-01", "2026-02-25"), // 25 days
		makeCycle("c3", "u1", "2026-03-01", "2026-03-20"), // 20 days
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, nil, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN {
			found = true
			if !contains(ins.Summary, "decreasing") && !contains(ins.Summary, "shorter") {
				t.Errorf("expected 'decreasing' or 'shorter' in summary, got: %s", ins.Summary)
			}
		}
	}
	if !found {
		t.Errorf("expected CYCLE_LENGTH_PATTERN insight, got none")
	}
}

// TestCycleLengthPatternIncreasing: 3+ cycles with increasing lengths → LENGTHENING pattern.
func TestCycleLengthPatternIncreasing(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-20"), // 20 days
		makeCycle("c2", "u1", "2026-02-01", "2026-02-25"), // 25 days
		makeCycle("c3", "u1", "2026-03-01", "2026-03-31"), // 31 days
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, nil, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN {
			found = true
			if !contains(ins.Summary, "increasing") && !contains(ins.Summary, "longer") {
				t.Errorf("expected 'increasing' or 'longer' in summary, got: %s", ins.Summary)
			}
		}
	}
	if !found {
		t.Errorf("expected CYCLE_LENGTH_PATTERN insight, got none")
	}
}

// TestCycleLengthPatternStable: 3+ cycles with stable lengths → STABLE pattern.
func TestCycleLengthPatternStable(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-29"), // 29 days
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"), // 28 days
		makeCycle("c3", "u1", "2026-03-01", "2026-03-29"), // 29 days
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, nil, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN {
			found = true
			if !contains(ins.Summary, "stable") {
				t.Errorf("expected 'stable' in summary, got: %s", ins.Summary)
			}
		}
	}
	if !found {
		t.Errorf("expected CYCLE_LENGTH_PATTERN insight, got none")
	}
}

// TestCycleLengthPatternAllOutliers: All completed cycles are outliers → no CYCLE_LENGTH_PATTERN.
func TestCycleLengthPatternAllOutliers(t *testing.T) {
	profile := makeProfile()
	// All cycles are shorter than the 15-day minimum outlier threshold.
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-05"), // 5 days — outlier
		makeCycle("c2", "u1", "2026-02-01", "2026-02-07"), // 7 days — outlier
		makeCycle("c3", "u1", "2026-03-01", "2026-03-08"), // 8 days — outlier
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, nil, profile)

	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN {
			t.Errorf("expected no CYCLE_LENGTH_PATTERN when all cycles are outliers, got one")
		}
	}
}

// TestSymptomPatternTooFewCycles: < 3 cycles → no SYMPTOM_PATTERN.
func TestSymptomPatternTooFewCycles(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
	}
	symptoms := []*v1.SymptomObservation{
		makeSymptomObs("s1", "u1", "2026-01-10", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
		makeSymptomObs("s2", "u1", "2026-01-11", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
	}

	insightsResult := insights.Generate("u1", cycles, symptoms, nil, nil, nil, profile)

	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_SYMPTOM_PATTERN {
			t.Errorf("expected no SYMPTOM_PATTERN (< 3 cycles), got one")
		}
	}
}

// TestSymptomPatternTooFewObservations: < 3 observations → no SYMPTOM_PATTERN.
func TestSymptomPatternTooFewObservations(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"),
		makeCycle("c3", "u1", "2026-03-01", "2026-03-28"),
	}
	symptoms := []*v1.SymptomObservation{
		makeSymptomObs("s1", "u1", "2026-01-10", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
		makeSymptomObs("s2", "u1", "2026-02-11", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
	}

	insightsResult := insights.Generate("u1", cycles, symptoms, nil, nil, nil, profile)

	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_SYMPTOM_PATTERN {
			t.Errorf("expected no SYMPTOM_PATTERN (< 3 observations), got one")
		}
	}
}

// TestSymptomPattern: 3+ cycles with 3+ matching symptom observations on similar cycle days.
func TestSymptomPattern(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"),
		makeCycle("c3", "u1", "2026-03-01", "2026-03-28"),
	}
	// Day 10 in each cycle (relative day)
	symptoms := []*v1.SymptomObservation{
		makeSymptomObs("s1", "u1", "2026-01-10", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
		makeSymptomObs("s2", "u1", "2026-02-10", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
		makeSymptomObs("s3", "u1", "2026-03-10", v1.SymptomType_SYMPTOM_TYPE_HEADACHE),
	}

	insightsResult := insights.Generate("u1", cycles, symptoms, nil, nil, nil, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_SYMPTOM_PATTERN {
			found = true
			if !contains(ins.Summary, "10") {
				t.Errorf("expected cycle day 10 in summary, got: %s", ins.Summary)
			}
		}
	}
	if !found {
		t.Errorf("expected SYMPTOM_PATTERN insight, got none")
	}
}

// TestMedicationAdherencePatternTooFewDays: < 14 days of data → no MEDICATION_ADHERENCE_PATTERN.
func TestMedicationAdherencePatternTooFewDays(t *testing.T) {
	profile := makeProfile()
	medications := []*v1.Medication{
		makeMedication("ibuprofen", "u1", true),
	}
	medicationEvents := []*v1.MedicationEvent{
		makeMedicationEvent("me1", "u1", "ibuprofen", "2026-01-01"),
		makeMedicationEvent("me2", "u1", "ibuprofen", "2026-01-02"),
		makeMedicationEvent("me3", "u1", "ibuprofen", "2026-01-03"),
		makeMedicationEvent("me4", "u1", "ibuprofen", "2026-01-10"),
	}

	insightsResult := insights.Generate("u1", nil, nil, medications, medicationEvents, nil, profile)

	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN {
			t.Errorf("expected no MEDICATION_ADHERENCE_PATTERN (< 14 days), got one")
		}
	}
}

// TestMedicationAdherencePatternInactive: Inactive medication → no MEDICATION_ADHERENCE_PATTERN.
func TestMedicationAdherencePatternInactive(t *testing.T) {
	profile := makeProfile()
	medications := []*v1.Medication{
		makeMedication("old-med", "u1", false),
	}
	medicationEvents := []*v1.MedicationEvent{
		makeMedicationEvent("me1", "u1", "old-med", "2026-01-01"),
		makeMedicationEvent("me2", "u1", "old-med", "2026-01-10"),
	}

	insightsResult := insights.Generate("u1", nil, nil, medications, medicationEvents, nil, profile)

	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN {
			t.Errorf("expected no MEDICATION_ADHERENCE_PATTERN (inactive), got one")
		}
	}
}

// TestMedicationAdherencePatternHigh: 20+ days with daily events → HIGH adherence.
func TestMedicationAdherencePatternHigh(t *testing.T) {
	profile := makeProfile()
	medications := []*v1.Medication{
		makeMedication("ibuprofen", "u1", true),
	}
	// Daily events for 20 days
	var events []*v1.MedicationEvent
	for i := 1; i <= 20; i++ {
		dateStr := fmt.Sprintf("2026-01-%02d", i+9) // Jan 10-29
		events = append(events, makeMedicationEvent(fmt.Sprintf("me%d", i), "u1", "ibuprofen", dateStr))
	}

	insightsResult := insights.Generate("u1", nil, nil, medications, events, nil, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN {
			found = true
			// Check for high adherence indication
			if !contains(ins.Summary, "HIGH") {
				t.Errorf("expected 'HIGH' adherence in summary, got: %s", ins.Summary)
			}
		}
	}
	if !found {
		t.Errorf("expected MEDICATION_ADHERENCE_PATTERN insight, got none")
	}
}

// TestMedicationAdherencePatternModerate: 70-89% adherence.
func TestMedicationAdherencePatternModerate(t *testing.T) {
	profile := makeProfile()
	medications := []*v1.Medication{
		makeMedication("aspirin", "u1", true),
	}
	// 80% adherence: 16 days with events over 20 days (Jan 1-20)
	var events []*v1.MedicationEvent
	days := []int{1, 2, 3, 4, 5, 7, 8, 9, 10, 12, 14, 15, 16, 18, 19, 20}
	for i, day := range days {
		dateStr := fmt.Sprintf("2026-01-%02d", day)
		events = append(events, makeMedicationEvent(fmt.Sprintf("me%d", i), "u1", "aspirin", dateStr))
	}

	insightsResult := insights.Generate("u1", nil, nil, medications, events, nil, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN {
			found = true
			// At least verify it was created
		}
	}
	if !found {
		t.Errorf("expected MEDICATION_ADHERENCE_PATTERN insight, got none")
	}
}

// TestBleedingPatternTooFewCycles: < 3 cycles with bleeding → no BLEEDING_PATTERN.
func TestBleedingPatternTooFewCycles(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
	}
	bleedingObs := []*v1.BleedingObservation{
		makeBleedingObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b2", "u1", "2026-01-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, bleedingObs, profile)

	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_BLEEDING_PATTERN {
			t.Errorf("expected no BLEEDING_PATTERN (< 3 cycles), got one")
		}
	}
}

// TestBleedingPatternStable: 3+ cycles with stable bleed duration.
func TestBleedingPatternStable(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"),
		makeCycle("c3", "u1", "2026-03-01", "2026-03-28"),
	}
	// 5 days of bleeding in each cycle
	bleedingObs := []*v1.BleedingObservation{
		makeBleedingObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b2", "u1", "2026-01-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b3", "u1", "2026-01-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b4", "u1", "2026-01-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b5", "u1", "2026-01-05", v1.BleedingFlow_BLEEDING_FLOW_LIGHT),
		makeBleedingObs("b6", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b7", "u1", "2026-02-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b8", "u1", "2026-02-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b9", "u1", "2026-02-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b10", "u1", "2026-02-05", v1.BleedingFlow_BLEEDING_FLOW_LIGHT),
		makeBleedingObs("b11", "u1", "2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b12", "u1", "2026-03-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b13", "u1", "2026-03-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b14", "u1", "2026-03-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b15", "u1", "2026-03-05", v1.BleedingFlow_BLEEDING_FLOW_LIGHT),
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, bleedingObs, profile)

	found := false
	for _, ins := range insightsResult {
		if ins.Kind == v1.InsightType_INSIGHT_TYPE_BLEEDING_PATTERN {
			found = true
			if !contains(ins.Summary, "stable") {
				t.Errorf("expected 'stable' in summary, got: %s", ins.Summary)
			}
		}
	}
	if !found {
		t.Errorf("expected BLEEDING_PATTERN insight, got none")
	}
}

// TestConfidenceLevels: Verify confidence is properly assigned.
func TestConfidenceLevels(t *testing.T) {
	profile := makeProfile()
	profile.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_REGULAR

	// 5+ cycles → HIGH confidence
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"),
		makeCycle("c3", "u1", "2026-03-01", "2026-03-28"),
		makeCycle("c4", "u1", "2026-04-01", "2026-04-28"),
		makeCycle("c5", "u1", "2026-05-01", "2026-05-28"),
	}

	bleedingObs := []*v1.BleedingObservation{
		makeBleedingObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b3", "u1", "2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b4", "u1", "2026-04-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b5", "u1", "2026-05-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, bleedingObs, profile)

	if len(insightsResult) == 0 {
		t.Fatalf("expected insights, got none")
	}

	for _, ins := range insightsResult {
		if ins.Confidence != v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH {
			t.Errorf("expected HIGH confidence for 5+ cycles, got %v", ins.Confidence)
		}
	}
}

// TestConfidenceLevelsCapped: Very irregular profile → LOW confidence cap.
func TestConfidenceLevelsCapped(t *testing.T) {
	profile := makeProfile()
	profile.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR

	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"),
		makeCycle("c3", "u1", "2026-03-01", "2026-03-28"),
		makeCycle("c4", "u1", "2026-04-01", "2026-04-28"),
		makeCycle("c5", "u1", "2026-05-01", "2026-05-28"),
	}

	bleedingObs := []*v1.BleedingObservation{
		makeBleedingObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b3", "u1", "2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b4", "u1", "2026-04-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b5", "u1", "2026-05-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, bleedingObs, profile)

	if len(insightsResult) == 0 {
		t.Fatalf("expected insights, got none")
	}

	for _, ins := range insightsResult {
		if ins.Confidence != v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW {
			t.Errorf("expected LOW confidence (VERY_IRREGULAR), got %v", ins.Confidence)
		}
	}
}

// TestEvidenceRecordRefs: Verify evidence references are populated.
func TestEvidenceRecordRefs(t *testing.T) {
	profile := makeProfile()
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-02-01", "2026-02-28"),
		makeCycle("c3", "u1", "2026-03-01", "2026-03-28"),
	}

	bleedingObs := []*v1.BleedingObservation{
		makeBleedingObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
		makeBleedingObs("b3", "u1", "2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM),
	}

	insightsResult := insights.Generate("u1", cycles, nil, nil, nil, bleedingObs, profile)

	for _, ins := range insightsResult {
		if len(ins.EvidenceRecordRefs) == 0 {
			t.Errorf("expected evidence record refs, got none for insight %v", ins.Kind)
		}
	}
}
