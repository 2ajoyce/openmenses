package rules_test

import (
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// regularProfile returns a profile with an ovulatory, regular cycle model.
func regularProfile() *v1.UserProfile {
	return &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
	}
}

// countPhase counts how many estimates have the given phase.
func countPhase(ests []*v1.PhaseEstimate, phase v1.CyclePhase) int {
	n := 0
	for _, e := range ests {
		if e.GetPhase() == phase {
			n++
		}
	}
	return n
}

// allConfidence returns true when every estimate has the given confidence.
func allConfidence(ests []*v1.PhaseEstimate, c v1.ConfidenceLevel) bool {
	for _, e := range ests {
		if e.GetConfidence() != c {
			return false
		}
	}
	return true
}

// ---- Basic ovulatory model ------------------------------------------------ //

func TestPhases_Ovulatory_28Day(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 5)

	if len(ests) != 28 {
		t.Fatalf("expected 28 estimates, got %d", len(ests))
	}

	// O = 28-14 = 14. Phases:
	// Menstruation: days 1-5     → 5  days
	// Follicular:   days 6-12    → 7  days (o-2=12)
	// Ovulation:    days 13-15   → 3  days (o-1=13, o+1=15)
	// Luteal:       days 16-28   → 13 days
	gotMen := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
	gotFol := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
	gotOvl := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
	gotLut := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_LUTEAL)

	if gotMen != 5 {
		t.Errorf("menstruation = %d days, want 5", gotMen)
	}
	if gotFol != 7 {
		t.Errorf("follicular = %d days, want 7", gotFol)
	}
	if gotOvl != 3 {
		t.Errorf("ovulation_window = %d days, want 3", gotOvl)
	}
	if gotLut != 13 {
		t.Errorf("luteal = %d days, want 13", gotLut)
	}
}

func TestPhases_Ovulatory_30Day(t *testing.T) {
	// O = 30-14 = 16. Follicular: days 6-14 (9 days). Ovulation: 15-17 (3 days). Luteal: 18-30 (13 days).
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-30")
	ests := rules.EstimatePhases(cycle, regularProfile(), 30, 5)

	if len(ests) != 30 {
		t.Fatalf("expected 30 estimates, got %d", len(ests))
	}
	gotOvl := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
	if gotOvl != 3 {
		t.Errorf("ovulation_window = %d days, want 3", gotOvl)
	}
}

// ---- Open-ended cycle ---------------------------------------------------- //

func TestPhases_OpenEnded_UsesAvgLen(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "")
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 3)
	// Should estimate 28 days when open-ended with avgCycleLen=28.
	if len(ests) != 28 {
		t.Fatalf("expected 28 estimates for open-ended cycle (avg=28), got %d", len(ests))
	}
}

func TestPhases_OpenEnded_DefaultAvg(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "")
	// avgCycleLen=0 → default 28.
	ests := rules.EstimatePhases(cycle, regularProfile(), 0, 1)
	if len(ests) != 28 {
		t.Fatalf("expected 28 estimates (default avg), got %d", len(ests))
	}
}

// ---- Hormonally suppressed model ----------------------------------------- //

func TestPhases_HormonallySuppressed_NoOvulationWindow(t *testing.T) {
	profile := &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
	}
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, profile, 28, 5)

	if len(ests) != 28 {
		t.Fatalf("expected 28 estimates, got %d", len(ests))
	}
	ovl := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
	if ovl != 0 {
		t.Errorf("suppressed model should have 0 ovulation window days, got %d", ovl)
	}
	men := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
	if men != 5 {
		t.Errorf("menstruation = %d days, want 5", men)
	}
	fol := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
	if fol != 23 {
		t.Errorf("follicular = %d days, want 23", fol)
	}
}

// ---- Confidence assignment ----------------------------------------------- //

func TestPhases_Confidence_LowWhenFewCycles(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 1)
	if !allConfidence(ests, v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW) {
		t.Error("expected LOW confidence for <2 completed cycles")
	}
}

func TestPhases_Confidence_MediumFor2to4Cycles(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 3)
	if !allConfidence(ests, v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM) {
		t.Error("expected MEDIUM confidence for 3 completed cycles (regular)")
	}
}

func TestPhases_Confidence_HighFor5PlusCycles(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 5)
	if !allConfidence(ests, v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH) {
		t.Error("expected HIGH confidence for ≥5 completed cycles (regular)")
	}
}

func TestPhases_Confidence_VeryIrregularCapsAtLow(t *testing.T) {
	profile := &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR,
	}
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	// 10 cycles → base HIGH, but VERY_IRREGULAR caps at LOW.
	ests := rules.EstimatePhases(cycle, profile, 28, 10)
	if !allConfidence(ests, v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW) {
		t.Error("expected LOW confidence for VERY_IRREGULAR regardless of cycle count")
	}
}

func TestPhases_Confidence_SomewhatIrregularCapsAtMedium(t *testing.T) {
	profile := &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR,
	}
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, profile, 28, 10)
	if !allConfidence(ests, v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM) {
		t.Error("expected MEDIUM confidence for SOMEWHAT_IRREGULAR with many cycles")
	}
}

func TestPhases_Confidence_IrregularModelCapsAtMedium(t *testing.T) {
	profile := &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
	}
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, profile, 28, 10)
	// §2.3: ovulation window is always LOW; all other phases are capped at MEDIUM.
	for _, e := range ests {
		if e.GetPhase() == v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW {
			if e.GetConfidence() != v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW {
				t.Errorf("ovulation window phase: expected LOW confidence, got %v", e.GetConfidence())
			}
		} else {
			if e.GetConfidence() != v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM {
				t.Errorf("non-ovulation phase %v: expected MEDIUM confidence, got %v", e.GetPhase(), e.GetConfidence())
			}
		}
	}
}

// ---- Irregular model phase widening (§2.3) -------------------------------- //

// irregularProfile returns a profile with BIOLOGICAL_CYCLE_MODEL_IRREGULAR.
func irregularProfile() *v1.UserProfile {
	return &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
	}
}

// TestPhases_Irregular_28Day verifies the ±3-day widening for a 28-day cycle.
// O = 28-14 = 14.  Expected widened boundaries:
//   - Menstruation:     days 1–8      (5+3)
//   - Follicular:       day 9         (O-5 = 9; 1 day)
//   - Ovulation window: days 10–18    (O-4=10, O+4=18; 9 days)
//   - Luteal:           days 19–28    (O+5=19; 10 days)
func TestPhases_Irregular_28Day(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
	ests := rules.EstimatePhases(cycle, irregularProfile(), 28, 5)

	if len(ests) != 28 {
		t.Fatalf("expected 28 estimates, got %d", len(ests))
	}

	gotMen := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
	gotFol := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
	gotOvl := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
	gotLut := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_LUTEAL)

	if gotMen != 8 {
		t.Errorf("menstruation = %d days, want 8 (widened from 5)", gotMen)
	}
	if gotFol != 1 {
		t.Errorf("follicular = %d days, want 1 (compressed for 28-day cycle)", gotFol)
	}
	if gotOvl != 9 {
		t.Errorf("ovulation_window = %d days, want 9 (widened ±3 from 3)", gotOvl)
	}
	if gotLut != 10 {
		t.Errorf("luteal = %d days, want 10", gotLut)
	}
}

// TestPhases_Irregular_30Day verifies the ±3-day widening for a 30-day cycle.
// O = 30-14 = 16.  Expected:
//   - Menstruation:     days 1–8      (8 days)
//   - Follicular:       days 9–11     (O-5=11; 3 days)
//   - Ovulation window: days 12–20    (O-4=12, O+4=20; 9 days)
//   - Luteal:           days 21–30    (O+5=21; 10 days)
func TestPhases_Irregular_30Day(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-30")
	ests := rules.EstimatePhases(cycle, irregularProfile(), 30, 5)

	if len(ests) != 30 {
		t.Fatalf("expected 30 estimates, got %d", len(ests))
	}

	gotMen := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
	gotFol := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
	gotOvl := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
	gotLut := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_LUTEAL)

	if gotMen != 8 {
		t.Errorf("menstruation = %d days, want 8", gotMen)
	}
	if gotFol != 3 {
		t.Errorf("follicular = %d days, want 3", gotFol)
	}
	if gotOvl != 9 {
		t.Errorf("ovulation_window = %d days, want 9", gotOvl)
	}
	if gotLut != 10 {
		t.Errorf("luteal = %d days, want 10", gotLut)
	}
}

// TestPhases_Irregular_WiderThanOvulatory verifies that the irregular model's
// ovulation window is always wider than the standard ovulatory model.
func TestPhases_Irregular_WiderThanOvulatory(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")

	ovulatoryEsts := rules.EstimatePhases(cycle, regularProfile(), 28, 5)
	irregularEsts := rules.EstimatePhases(cycle, irregularProfile(), 28, 5)

	stdOvl := countPhase(ovulatoryEsts, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
	irrOvl := countPhase(irregularEsts, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)

	if irrOvl <= stdOvl {
		t.Errorf("irregular ovulation window (%d days) should be wider than standard (%d days)", irrOvl, stdOvl)
	}

	stdMen := countPhase(ovulatoryEsts, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
	irrMen := countPhase(irregularEsts, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)

	if irrMen <= stdMen {
		t.Errorf("irregular menstruation (%d days) should be wider than standard (%d days)", irrMen, stdMen)
	}
}

// ---- Dates fields populated correctly ------------------------------------ //

func TestPhases_DatesSequential(t *testing.T) {
	cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-03")
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 5)
	if len(ests) != 3 {
		t.Fatalf("expected 3 estimates, got %d", len(ests))
	}
	wantDates := []string{"2026-01-01", "2026-01-02", "2026-01-03"}
	for i, e := range ests {
		if e.GetDate().GetValue() != wantDates[i] {
			t.Errorf("estimate[%d] date = %q, want %q", i, e.GetDate().GetValue(), wantDates[i])
		}
		if e.GetUserId() != "u1" {
			t.Errorf("estimate[%d] user_id = %q, want u1", i, e.GetUserId())
		}
		if e.GetName() == "" {
			t.Errorf("estimate[%d] has empty name", i)
		}
	}
}

// ---- No start date --------------------------------------------------------- //

func TestPhases_MissingStartDate_Empty(t *testing.T) {
	cycle := &v1.Cycle{Name: "cy1", UserId: "u1"}
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 5)
	if len(ests) != 0 {
		t.Errorf("expected 0 estimates for cycle with no start_date, got %d", len(ests))
	}
}
