package predictions_test

import (
	"testing"
	"time"

	"github.com/2ajoyce/openmenses/engine/internal/predictions"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// makeCycle creates a Cycle with YYYY-MM-DD start and end dates.
// Pass empty string for end to create an open-ended cycle.
func makeCycle(id, uid, start, end string) *v1.Cycle {
	c := &v1.Cycle{
		Name:      id,
		UserId:    uid,
		StartDate: &v1.LocalDate{Value: start},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if end != "" {
		c.EndDate = &v1.LocalDate{Value: end}
	}
	return c
}

// makeProfile creates a UserProfile with given biological cycle and regularity.
func makeProfile(bio v1.BiologicalCycleModel, regularity v1.CycleRegularity) *v1.UserProfile {
	return &v1.UserProfile{
		BiologicalCycle: bio,
		CycleRegularity: regularity,
	}
}

// makeSymptomObs creates a SymptomObservation on a given date.
func makeSymptomObs(dateStr string, symptom v1.SymptomType) *v1.SymptomObservation {
	ts := dateStr + "T12:00:00Z"
	return &v1.SymptomObservation{
		Symptom: symptom,
		Timestamp: &v1.DateTime{
			Value: ts,
		},
	}
}

// findPredictionByKind returns the first prediction of a given kind, or nil.
func findPredictionByKind(preds []*v1.Prediction, kind v1.PredictionType) *v1.Prediction {
	for _, p := range preds {
		if p.GetKind() == kind {
			return p
		}
	}
	return nil
}

// ============================================================================
// Table-driven tests for Generate
// ============================================================================

type generateTestCase struct {
	name      string
	cycles    []*v1.Cycle
	symptoms  []*v1.SymptomObservation
	profile   *v1.UserProfile
	wantCount int
	wantKinds []v1.PredictionType
	wantDates map[v1.PredictionType][2]string // [startDate, endDate]
}

func TestGenerate(t *testing.T) {
	userID := "user-1"

	tests := []generateTestCase{
		{
			name:      "no cycles",
			cycles:    nil,
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 0,
			wantKinds: nil,
		},

		{
			name: "one completed cycle",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 0,
			wantKinds: nil,
		},

		{
			name: "two completed cycles, ovulatory",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 2,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
			},
		},

		{
			name: "three completed cycles, regular, ovulatory",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 3,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
				v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW,
			},
		},

		{
			name: "three cycles, hormonally suppressed",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms: nil,
			profile: makeProfile(
				v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED,
				v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
			),
			wantCount: 2,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
			},
		},

		{
			name: "three cycles, irregular biological model",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms: nil,
			profile: makeProfile(
				v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR,
				v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
			),
			wantCount: 2,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
			},
		},

		{
			name: "three cycles, very irregular regularity",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms: nil,
			profile: makeProfile(
				v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
				v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR,
			),
			wantCount: 2,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
			},
		},

		{
			name: "three cycles with matching cramp observations",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms: []*v1.SymptomObservation{
				// 3 cramp observations on cycle day 5 across 3 cycles
				makeSymptomObs("2026-01-05", v1.SymptomType_SYMPTOM_TYPE_CRAMPS),
				makeSymptomObs("2026-02-02", v1.SymptomType_SYMPTOM_TYPE_CRAMPS),
				makeSymptomObs("2026-03-02", v1.SymptomType_SYMPTOM_TYPE_CRAMPS),
			},
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 4, // NEXT_BLEED, PMS_WINDOW, OVULATION_WINDOW, SYMPTOM_WINDOW
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
				v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW,
				v1.PredictionType_PREDICTION_TYPE_SYMPTOM_WINDOW,
			},
		},

		{
			name: "three cycles with insufficient symptom observations",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms: []*v1.SymptomObservation{
				// Only 2 cramp observations, not enough for prediction
				makeSymptomObs("2026-01-05", v1.SymptomType_SYMPTOM_TYPE_CRAMPS),
				makeSymptomObs("2026-02-02", v1.SymptomType_SYMPTOM_TYPE_CRAMPS),
			},
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 3, // No SYMPTOM_WINDOW
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
				v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW,
			},
		},

		{
			name: "three completed cycles, confidence levels correct",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 3,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
				v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW,
			},
		},

		{
			name: "all outlier cycles",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-04-02"), // 92 days: outlier (>90)
				makeCycle("c2", userID, "2026-04-03", "2026-07-03"), // 92 days: outlier (>90)
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 0,
			wantKinds: nil,
		},

		{
			name: "two completed cycles where one is outlier",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"), // 28 days: valid
				makeCycle("c2", userID, "2026-01-29", "2026-05-02"), // 94 days: outlier (>90)
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 0, // Only 1 non-outlier; need ≥2
			wantKinds: nil,
		},

		{
			name: "next bleed end date is start + 5 days",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 2,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
			},
			wantDates: map[v1.PredictionType][2]string{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED: {"2026-02-26", "2026-03-03"},
			},
		},

		{
			name: "two completed cycles with open cycle as anchor",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("open", userID, "2026-02-26", ""), // open-ended
			},
			symptoms:  nil,
			profile:   makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantCount: 2,
			wantKinds: []v1.PredictionType{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preds := predictions.Generate(userID, tt.cycles, tt.symptoms, tt.profile)

			if len(preds) != tt.wantCount {
				t.Errorf("got %d predictions, want %d", len(preds), tt.wantCount)
			}

			// Check that all wanted kinds are present.
			for _, kind := range tt.wantKinds {
				if findPredictionByKind(preds, kind) == nil {
					t.Errorf("expected prediction of kind %v not found", kind)
				}
			}

			// Check unwanted kinds are absent.
			if tt.wantKinds == nil {
				for _, pred := range preds {
					t.Errorf("unexpected prediction of kind %v", pred.GetKind())
				}
			}

			// Check specific dates if provided.
			for kind, dates := range tt.wantDates {
				pred := findPredictionByKind(preds, kind)
				if pred == nil {
					t.Errorf("prediction of kind %v not found for date check", kind)
					continue
				}
				gotStart := pred.GetPredictedStartDate().GetValue()
				gotEnd := pred.GetPredictedEndDate().GetValue()
				if gotStart != dates[0] || gotEnd != dates[1] {
					t.Errorf("kind %v: got dates [%s, %s], want [%s, %s]",
						kind, gotStart, gotEnd, dates[0], dates[1])
				}
			}

			// Check all predictions have name and user_id.
			for _, pred := range preds {
				if pred.GetName() == "" {
					t.Error("prediction missing name")
				}
				if pred.GetUserId() != userID {
					t.Errorf("prediction has user_id %q, want %q", pred.GetUserId(), userID)
				}
			}
		})
	}
}

func TestPredictedDates(t *testing.T) {
	userID := "user-1"

	// Test that predicted dates scale with average cycle length.
	tests := []struct {
		name     string
		cycles   []*v1.Cycle
		avgLen   int
		dayCount map[v1.PredictionType]int // [kind] = expected day offset from nextBleedStart
	}{
		{
			name: "28-day cycle with 2 cycles",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
			},
			avgLen: 28,
			dayCount: map[v1.PredictionType]int{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED: 0,   // starts on nextBleedStart
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW: -10, // 10 days before
			},
		},
		{
			name: "28-day cycle with 3 cycles (includes ovulation)",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-24"),
			},
			avgLen: 28,
			dayCount: map[v1.PredictionType]int{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED:       0,   // starts on nextBleedStart
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW:       -10, // 10 days before
				v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW: 12,  // O = 28-14 = 14, so starts at day 12 (14-2)
			},
		},
		{
			name: "35-day cycle with 3 cycles",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-02-04"),
				makeCycle("c2", userID, "2026-02-05", "2026-03-11"),
				makeCycle("c3", userID, "2026-03-12", "2026-04-15"),
			},
			avgLen: 35,
			dayCount: map[v1.PredictionType]int{
				v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED:       0,   // starts on nextBleedStart
				v1.PredictionType_PREDICTION_TYPE_PMS_WINDOW:       -10, // 10 days before
				v1.PredictionType_PREDICTION_TYPE_OVULATION_WINDOW: 19,  // O = 35-14 = 21, so starts at day 19 (21-2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR)
			preds := predictions.Generate(userID, tt.cycles, nil, profile)

			if len(preds) < len(tt.dayCount) {
				t.Fatalf("got %d predictions, want at least %d", len(preds), len(tt.dayCount))
			}

			// Get nextBleedStart from the last cycle.
			lastCycle := tt.cycles[len(tt.cycles)-1]
			lastStart := lastCycle.GetStartDate().GetValue()
			lastT, _ := time.Parse("2006-01-02", lastStart)
			nextBleedStart := lastT.AddDate(0, 0, tt.avgLen)

			for kind, expectedDayOffset := range tt.dayCount {
				pred := findPredictionByKind(preds, kind)
				if pred == nil {
					t.Errorf("prediction kind %v not found", kind)
					continue
				}

				startStr := pred.GetPredictedStartDate().GetValue()
				startT, err := time.Parse("2006-01-02", startStr)
				if err != nil {
					t.Errorf("could not parse start date %q: %v", startStr, err)
					continue
				}

				dayOffset := int(startT.Sub(nextBleedStart).Hours() / 24)
				if dayOffset != expectedDayOffset {
					t.Errorf("kind %v: predicted start day offset %d, want %d",
						kind, dayOffset, expectedDayOffset)
				}
			}
		})
	}
}

func TestConfidenceLevels(t *testing.T) {
	userID := "user-1"

	tests := []struct {
		name     string
		cycles   []*v1.Cycle
		profile  *v1.UserProfile
		wantConf v1.ConfidenceLevel
	}{
		{
			name: "two cycles, regular: MEDIUM",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
			},
			profile:  makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantConf: v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
		},
		{
			name: "two cycles, very irregular: LOW",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
			},
			profile:  makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR),
			wantConf: v1.ConfidenceLevel_CONFIDENCE_LEVEL_LOW,
		},
		{
			name: "five cycles, regular: HIGH",
			cycles: []*v1.Cycle{
				makeCycle("c1", userID, "2026-01-01", "2026-01-28"),
				makeCycle("c2", userID, "2026-01-29", "2026-02-25"),
				makeCycle("c3", userID, "2026-02-26", "2026-03-25"),
				makeCycle("c4", userID, "2026-03-26", "2026-04-22"),
				makeCycle("c5", userID, "2026-04-23", "2026-05-20"),
			},
			profile:  makeProfile(v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY, v1.CycleRegularity_CYCLE_REGULARITY_REGULAR),
			wantConf: v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preds := predictions.Generate(userID, tt.cycles, nil, tt.profile)

			if len(preds) == 0 {
				t.Fatal("no predictions generated")
			}

			// All predictions should have the same confidence.
			for _, pred := range preds {
				if pred.GetConfidence() != tt.wantConf {
					t.Errorf("got confidence %v, want %v",
						pred.GetConfidence(), tt.wantConf)
				}
			}
		})
	}
}
