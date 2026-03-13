package rules_test

import (
	"fmt"
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

// irregularProfile returns a profile with BIOLOGICAL_CYCLE_MODEL_IRREGULAR.
func irregularProfile() *v1.UserProfile {
	return &v1.UserProfile{
		Name:            "u1",
		BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR,
		CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
	}
}

// ============================================================================
// TestEstimatePhases_Basic — Basic cycle length and open-ended behavior
// ============================================================================

type basicPhasesTestCase struct {
	name                string
	cycleDays           int
	cycleEnd            string
	avgCycleLen         int
	completedCycleCount int

	wantEstimateCount int
	wantMenDays       int
	wantFolDays       int
	wantOvlDays       int
	wantLutDays       int
}

func TestEstimatePhases_Basic(t *testing.T) {
	tests := []basicPhasesTestCase{
		{
			name:                "28DayRegular",
			cycleDays:           28,
			cycleEnd:            "2026-01-28",
			avgCycleLen:         28,
			completedCycleCount: 5,
			wantEstimateCount:   28,
			wantMenDays:         5,  // Days 1-5
			wantFolDays:         7,  // Days 6-12 (O-2=12)
			wantOvlDays:         3,  // Days 13-15 (O-1 to O+1)
			wantLutDays:         13, // Days 16-28
		},
		{
			name:                "30DayRegular",
			cycleDays:           30,
			cycleEnd:            "2026-01-30",
			avgCycleLen:         30,
			completedCycleCount: 5,
			wantEstimateCount:   30,
			wantOvlDays:         3, // Still 3 days for ovulation window
		},
		{
			name:                "OpenEndedWithAvg",
			cycleDays:           0,
			cycleEnd:            "",
			avgCycleLen:         28,
			completedCycleCount: 3,
			wantEstimateCount:   28, // Should estimate 28 days
		},
		{
			name:                "OpenEndedDefaultAvg",
			cycleDays:           0,
			cycleEnd:            "",
			avgCycleLen:         0, // Should default to 28
			completedCycleCount: 1,
			wantEstimateCount:   28,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycleEnd := tt.cycleEnd
			if tt.cycleDays > 0 && cycleEnd == "" {
				// Calculate end date
				cycleEnd = "2026-01-" + padInt(tt.cycleDays, 2)
			}

			cycle := makeCycle("cy1", "u1", "2026-01-01", cycleEnd)
			ests := rules.EstimatePhases(cycle, regularProfile(), tt.avgCycleLen, tt.completedCycleCount)

			if len(ests) != tt.wantEstimateCount {
				t.Errorf("estimate count = %d, want %d", len(ests), tt.wantEstimateCount)
			}

			if tt.wantMenDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
				if got != tt.wantMenDays {
					t.Errorf("menstruation days = %d, want %d", got, tt.wantMenDays)
				}
			}
			if tt.wantFolDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
				if got != tt.wantFolDays {
					t.Errorf("follicular days = %d, want %d", got, tt.wantFolDays)
				}
			}
			if tt.wantOvlDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
				if got != tt.wantOvlDays {
					t.Errorf("ovulation window days = %d, want %d", got, tt.wantOvlDays)
				}
			}
			if tt.wantLutDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_LUTEAL)
				if got != tt.wantLutDays {
					t.Errorf("luteal days = %d, want %d", got, tt.wantLutDays)
				}
			}
		})
	}
}

// ============================================================================
// TestEstimatePhases_BiologicalModels — Different biological cycle models
// ============================================================================

type biologicalModelTestCase struct {
	name                 string
	profile              *v1.UserProfile
	cycleDays            int
	wantOvulationDays    int
	wantMenstruationDays int
	wantMenstruationMin  int
	wantFollicularDays   int
	wantLutealDays       int
}

func TestEstimatePhases_BiologicalModels(t *testing.T) {
	tests := []biologicalModelTestCase{
		{
			name:                 "RegularOvulatory",
			profile:              regularProfile(),
			cycleDays:            28,
			wantMenstruationDays: 5,
			wantFollicularDays:   7,
			wantOvulationDays:    3,
			wantLutealDays:       13,
		},
		{
			name: "HormonallySuppressed",
			profile: &v1.UserProfile{
				Name:            "u1",
				BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED,
				CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
			},
			cycleDays:            28,
			wantMenstruationDays: 5,
			wantFollicularDays:   23,
			wantOvulationDays:    0, // Should have no ovulation window
			wantLutealDays:       0, // Remaining goes to follicular
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-"+padInt(tt.cycleDays, 2))
			ests := rules.EstimatePhases(cycle, tt.profile, tt.cycleDays, 5)

			if len(ests) != tt.cycleDays {
				t.Errorf("estimate count = %d, want %d", len(ests), tt.cycleDays)
			}

			if tt.wantMenstruationDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
				if got != tt.wantMenstruationDays {
					t.Errorf("menstruation days = %d, want %d", got, tt.wantMenstruationDays)
				}
			}
			if tt.wantFollicularDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
				if got != tt.wantFollicularDays {
					t.Errorf("follicular days = %d, want %d", got, tt.wantFollicularDays)
				}
			}
			if tt.wantOvulationDays > 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
				if got != tt.wantOvulationDays {
					t.Errorf("ovulation window days = %d, want %d", got, tt.wantOvulationDays)
				}
			} else if tt.wantOvulationDays == 0 {
				got := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
				if got != 0 {
					t.Errorf("expected 0 ovulation window days, got %d", got)
				}
			}
		})
	}
}

// ============================================================================
// TestEstimatePhases_Confidence — Confidence level assignments
// ============================================================================

type confidenceTestCase struct {
	name             string
	profile          *v1.UserProfile
	completedCycles  int
	wantConfidence   v1.ConfidenceLevel
	wantOvulationCap bool // Some models cap ovulation confidence specifically
}

func TestEstimatePhases_Confidence(t *testing.T) {
	tests := []confidenceTestCase{
		{
			name:            "LowWhenFewCycles",
			profile:         regularProfile(),
			completedCycles: 1,
			wantConfidence:  v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW,
		},
		{
			name:            "MediumFor3Cycles",
			profile:         regularProfile(),
			completedCycles: 3,
			wantConfidence:  v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
		},
		{
			name:            "HighFor5Cycles",
			profile:         regularProfile(),
			completedCycles: 5,
			wantConfidence:  v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH,
		},
		{
			name: "VeryIrregularCapsAtLow",
			profile: &v1.UserProfile{
				Name:            "u1",
				BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
				CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR,
			},
			completedCycles: 10,
			wantConfidence:  v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW,
		},
		{
			name: "SomewhatIrregularCapsAtMedium",
			profile: &v1.UserProfile{
				Name:            "u1",
				BiologicalCycle: v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
				CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR,
			},
			completedCycles: 10,
			wantConfidence:  v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
		},
		{
			name:             "IrregularModelCapsNonOvulationAtMedium",
			profile:          irregularProfile(),
			completedCycles:  10,
			wantConfidence:   v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
			wantOvulationCap: true, // Ovulation window specifically capped at LOW
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-28")
			ests := rules.EstimatePhases(cycle, tt.profile, 28, tt.completedCycles)

			if tt.wantOvulationCap {
				// Check irregular model: ovulation window at LOW, others at MEDIUM
				for _, e := range ests {
					if e.GetPhase() == v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW {
						if e.GetConfidence() != v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW {
							t.Errorf("ovulation window: got %v, want LOW", e.GetConfidence())
						}
					} else {
						if e.GetConfidence() != v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM {
							t.Errorf("non-ovulation phase %v: got %v, want MEDIUM", e.GetPhase(), e.GetConfidence())
						}
					}
				}
			} else {
				// Check all estimates have expected confidence
				if !allConfidence(ests, tt.wantConfidence) {
					t.Errorf("not all estimates have confidence %v", tt.wantConfidence)
				}
			}
		})
	}
}

// ============================================================================
// TestEstimatePhases_IrregularModel — Irregular model specific phase widening
// ============================================================================

type irregularModelTestCase struct {
	name        string
	cycleDays   int
	wantMenDays int
	wantFolDays int
	wantOvlDays int
	wantLutDays int
}

func TestEstimatePhases_IrregularModel(t *testing.T) {
	tests := []irregularModelTestCase{
		{
			name:        "28DayWidened",
			cycleDays:   28,
			wantMenDays: 8,  // Widened from 5 (5+3)
			wantFolDays: 1,  // Compressed (O-5=9, only day 9)
			wantOvlDays: 9,  // Widened ±3 from 3
			wantLutDays: 10, // O+5=19
		},
		{
			name:        "30DayWidened",
			cycleDays:   30,
			wantMenDays: 8,  // Widened to 8
			wantFolDays: 3,  // O-5=11, days 9-11
			wantOvlDays: 9,  // Widened ±3
			wantLutDays: 10, // O+5=21
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycle := makeCycle("cy1", "u1", "2026-01-01", "2026-01-"+padInt(tt.cycleDays, 2))
			ests := rules.EstimatePhases(cycle, irregularProfile(), tt.cycleDays, 5)

			if len(ests) != tt.cycleDays {
				t.Errorf("estimate count = %d, want %d", len(ests), tt.cycleDays)
			}

			gotMen := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_MENSTRUATION)
			if gotMen != tt.wantMenDays {
				t.Errorf("menstruation days = %d, want %d", gotMen, tt.wantMenDays)
			}

			gotFol := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_FOLLICULAR)
			if gotFol != tt.wantFolDays {
				t.Errorf("follicular days = %d, want %d", gotFol, tt.wantFolDays)
			}

			gotOvl := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_OVULATION_WINDOW)
			if gotOvl != tt.wantOvlDays {
				t.Errorf("ovulation window days = %d, want %d", gotOvl, tt.wantOvlDays)
			}

			gotLut := countPhase(ests, v1.CyclePhase_CYCLE_PHASE_LUTEAL)
			if gotLut != tt.wantLutDays {
				t.Errorf("luteal days = %d, want %d", gotLut, tt.wantLutDays)
			}
		})
	}
}

// ============================================================================
// Additional validation tests — Irregular model widening comparison, dates, etc.
// ============================================================================

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

// TestPhases_DatesSequential validates that date fields are populated correctly.
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

// TestPhases_MissingStartDate_Empty validates behavior with no start date.
func TestPhases_MissingStartDate_Empty(t *testing.T) {
	cycle := &v1.Cycle{Name: "cy1", UserId: "u1"}
	ests := rules.EstimatePhases(cycle, regularProfile(), 28, 5)
	if len(ests) != 0 {
		t.Errorf("expected 0 estimates for cycle with no start_date, got %d", len(ests))
	}
}

// ============================================================================
// Utility helpers
// ============================================================================

// padInt formats an integer to a string with zero-padding.
func padInt(n, width int) string {
	return fmt.Sprintf("%0*d", width, n)
}
