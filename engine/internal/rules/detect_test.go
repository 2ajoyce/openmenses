package rules_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	"github.com/2ajoyce/openmenses/engine/internal/storage/memory"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

var ctx = context.Background()

// makeObs constructs a BleedingObservation with the given calendar date
// (YYYY-MM-DD) and flow.
func makeObs(id, uid, date string, flow v1.BleedingFlow) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Name:      id,
		UserId:    uid,
		Timestamp: &v1.DateTime{Value: date + "T00:00:00Z"},
		Flow:      flow,
	}
}

func mustCreate[T any](t *testing.T, fn func(context.Context, T) error, val T) {
	t.Helper()
	if err := fn(ctx, val); err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

// ============================================================================
// TestDetectCycles — Table-driven tests for cycle detection
// ============================================================================

type detectCyclesTestCase struct {
	name           string
	userID         string
	observations   [][2]interface{} // [0]=date string, [1]=BleedingFlow
	confirmedCycle *v1.Cycle
	multiUserObs   map[string][][2]interface{}

	wantCycleCount   int
	wantFirstName    string         // Expected name of first cycle
	wantFirstStart   string         // Expected start of first cycle
	wantFirstEnd     string         // Expected end of first cycle
	wantFirstSource  v1.CycleSource // Expected source of first cycle
	wantSecondStart  string         // Expected start of second cycle
	wantSecondEnd    string         // Expected end of second cycle
	wantSecondSource v1.CycleSource // Expected source of second cycle
	wantThirdEnd     string         // Expected end of third cycle
	wantOtherUserID  string         // For isolation test
	wantOtherCount   int            // For isolation test
}

func TestDetectCycles(t *testing.T) {
	tests := []detectCyclesTestCase{
		{
			name:           "NoObs_NoUserCycles",
			userID:         "u1",
			observations:   nil,
			wantCycleCount: 0,
		},
		{
			name:            "SingleObs_OpenEndedCycle",
			userID:          "u1",
			observations:    [][2]interface{}{{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM}},
			wantCycleCount:  1,
			wantFirstStart:  "2026-01-01",
			wantFirstEnd:    "",
			wantFirstSource: v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
		},
		{
			name:   "TwoRegularCycles",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-05", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			wantCycleCount:  2,
			wantFirstStart:  "2026-01-01",
			wantFirstEnd:    "2026-01-31",
			wantSecondStart: "2026-02-01",
			wantSecondEnd:   "",
		},
		{
			name:   "Gap4Days_NewEpisode",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-05", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			wantCycleCount: 2,
		},
		{
			name:   "Gap3Days_SameEpisode",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			wantCycleCount: 1,
		},
		{
			name:   "SpottingFollowedByHeavy_ValidCycleStart",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING},
				{"2026-02-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			wantCycleCount:  2,
			wantSecondStart: "2026-02-01",
		},
		{
			name:   "SpottingAlone_MidCycle",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING},
			},
			wantCycleCount: 1,
		},
		{
			name:   "ConfirmedCycle_NotOverridden",
			userID: "u1",
			confirmedCycle: &v1.Cycle{
				Name:      "cy-confirmed",
				UserId:    "u1",
				StartDate: &v1.LocalDate{Value: "2026-01-01"},
				EndDate:   &v1.LocalDate{Value: "2026-01-28"},
				Source:    v1.CycleSource_CYCLE_SOURCE_USER_CONFIRMED,
			},
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			wantCycleCount: 1,
			wantFirstName:  "cy-confirmed",
		},
		{
			name:   "DerivedAfterConfirmed_BothPresent",
			userID: "u1",
			confirmedCycle: &v1.Cycle{
				Name:      "cy-confirmed",
				UserId:    "u1",
				StartDate: &v1.LocalDate{Value: "2026-01-01"},
				EndDate:   &v1.LocalDate{Value: "2026-01-28"},
				Source:    v1.CycleSource_CYCLE_SOURCE_USER_CONFIRMED,
			},
			observations: [][2]interface{}{
				{"2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			wantCycleCount:   2,
			wantFirstName:    "cy-confirmed",
			wantSecondSource: v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
		},
		{
			name:   "Isolation_DifferentUsers",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
			},
			multiUserObs: map[string][][2]interface{}{
				"u2": {{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM}},
			},
			wantCycleCount:  1,
			wantOtherUserID: "u2",
			wantOtherCount:  1,
		},
		{
			name:   "AllSpotting_NoCycles",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING},
				{"2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING},
			},
			wantCycleCount: 0,
		},
		{
			name:   "MultipleEpisodes_CorrectBoundaries",
			userID: "u1",
			observations: [][2]interface{}{
				{"2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-01-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-02-03", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM},
				{"2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_LIGHT},
			},
			wantCycleCount: 3,
			wantFirstEnd:   "2026-01-31",
			wantSecondEnd:  "2026-02-28",
			wantThirdEnd:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()

			// Setup confirmed cycle if provided
			if tt.confirmedCycle != nil {
				mustCreate(t, store.Cycles().Create, tt.confirmedCycle)
			}

			// Setup observations for main user
			for i, obs := range tt.observations {
				date := obs[0].(string)
				flow := obs[1].(v1.BleedingFlow)
				id := fmt.Sprintf("b%s-%d", tt.userID, i)
				mustCreate(t, store.BleedingObservations().Create,
					makeObs(id, tt.userID, date, flow))
			}

			// Setup observations for other users (isolation test)
			for uid, obs := range tt.multiUserObs {
				for j, o := range obs {
					date := o[0].(string)
					flow := o[1].(v1.BleedingFlow)
					id := fmt.Sprintf("b%s-%d", uid, j)
					mustCreate(t, store.BleedingObservations().Create,
						makeObs(id, uid, date, flow))
				}
			}

			cycles, err := rules.DetectCycles(ctx, tt.userID, store)
			if err != nil {
				t.Fatalf("DetectCycles failed: %v", err)
			}

			if len(cycles) != tt.wantCycleCount {
				t.Errorf("cycle count = %d, want %d", len(cycles), tt.wantCycleCount)
			}

			if tt.wantCycleCount > 0 {
				// Check first cycle properties
				if tt.wantFirstName != "" && cycles[0].GetName() != tt.wantFirstName {
					t.Errorf("cycle[0].name = %q, want %q", cycles[0].GetName(), tt.wantFirstName)
				}
				if tt.wantFirstStart != "" && cycles[0].GetStartDate().GetValue() != tt.wantFirstStart {
					t.Errorf("cycle[0].start = %q, want %q", cycles[0].GetStartDate().GetValue(), tt.wantFirstStart)
				}
				if tt.wantFirstEnd != "" && cycles[0].GetEndDate().GetValue() != tt.wantFirstEnd {
					t.Errorf("cycle[0].end = %q, want %q", cycles[0].GetEndDate().GetValue(), tt.wantFirstEnd)
				}
				if tt.wantFirstSource != 0 && cycles[0].GetSource() != tt.wantFirstSource {
					t.Errorf("cycle[0].source = %v, want %v", cycles[0].GetSource(), tt.wantFirstSource)
				}
			}

			if tt.wantCycleCount > 1 {
				// Check second cycle properties
				if tt.wantSecondStart != "" && cycles[1].GetStartDate().GetValue() != tt.wantSecondStart {
					t.Errorf("cycle[1].start = %q, want %q", cycles[1].GetStartDate().GetValue(), tt.wantSecondStart)
				}
				if tt.wantSecondEnd != "" && cycles[1].GetEndDate().GetValue() != tt.wantSecondEnd {
					t.Errorf("cycle[1].end = %q, want %q", cycles[1].GetEndDate().GetValue(), tt.wantSecondEnd)
				}
				if tt.wantSecondSource != 0 && cycles[1].GetSource() != tt.wantSecondSource {
					t.Errorf("cycle[1].source = %v, want %v", cycles[1].GetSource(), tt.wantSecondSource)
				}
			}

			if tt.wantCycleCount > 2 {
				// Check third cycle properties
				if tt.wantThirdEnd != "" && cycles[2].GetEndDate().GetValue() != tt.wantThirdEnd {
					t.Errorf("cycle[2].end = %q, want %q", cycles[2].GetEndDate().GetValue(), tt.wantThirdEnd)
				}
			}

			// Check isolation (different users should have independent cycles)
			if tt.wantOtherUserID != "" {
				otherCycles, err := rules.DetectCycles(ctx, tt.wantOtherUserID, store)
				if err != nil {
					t.Fatalf("DetectCycles for other user failed: %v", err)
				}
				if len(otherCycles) != tt.wantOtherCount {
					t.Errorf("other user cycle count = %d, want %d", len(otherCycles), tt.wantOtherCount)
				}
				if tt.wantOtherCount > 0 && otherCycles[0].GetUserId() != tt.wantOtherUserID {
					t.Errorf("other user cycle attributed to wrong user: %q", otherCycles[0].GetUserId())
				}
			}
		})
	}
}

// ============================================================================
// TestIsOutlierLength — Table-driven tests for outlier detection
// ============================================================================

type outlierLengthTestCase struct {
	name        string
	startDate   string
	endDate     string
	wantOutlier bool
}

func TestIsOutlierLength(t *testing.T) {
	tests := []outlierLengthTestCase{
		{
			name:        "NormalCycle",
			startDate:   "2026-01-01",
			endDate:     "2026-01-28", // 28 days
			wantOutlier: false,
		},
		{
			name:        "TooShort",
			startDate:   "2026-01-01",
			endDate:     "2026-01-10", // 10 days
			wantOutlier: true,
		},
		{
			name:        "TooLong",
			startDate:   "2026-01-01",
			endDate:     "2026-05-01", // 121 days
			wantOutlier: true,
		},
		{
			name:        "AtMinBound",
			startDate:   "2026-01-01",
			endDate:     "2026-01-15", // 15 days (= minimum)
			wantOutlier: false,
		},
		{
			name:        "AtMaxBound",
			startDate:   "2026-01-01",
			endDate:     "2026-03-31", // 90 days (= maximum)
			wantOutlier: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycle := &v1.Cycle{
				Name:      "c1",
				UserId:    "u1",
				StartDate: &v1.LocalDate{Value: tt.startDate},
				EndDate:   &v1.LocalDate{Value: tt.endDate},
				Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
			}

			got := rules.IsOutlierLength(cycle)
			if got != tt.wantOutlier {
				t.Errorf("IsOutlierLength() = %v, want %v", got, tt.wantOutlier)
			}
		})
	}
}

// ---- Irregular cycle intervals -------------------------------------------- //

// TestDetect_IrregularCycles verifies that cycle detection correctly identifies
// cycles with highly variable interval lengths (22, 35, 28 days), demonstrating
// that the algorithm does not rely on regularity assumptions.
func TestDetect_IrregularCycles(t *testing.T) {
	store := memory.New()

	// Episode 1: Jan 1 — cycle interval to next: 22 days.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	// Episode 2: Jan 23 — 22 days after Jan 1; interval to next: 35 days.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-01-23", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	// Episode 3: Feb 27 — 35 days after Jan 23; interval to next: 28 days.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b3", "u1", "2026-02-27", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	// Episode 4: Mar 27 — 28 days after Feb 27; open-ended.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b4", "u1", "2026-03-27", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 4 {
		t.Fatalf("expected 4 cycles with irregular intervals, got %d", len(cycles))
	}
	// Verify start dates.
	wantStarts := []string{"2026-01-01", "2026-01-23", "2026-02-27", "2026-03-27"}
	for i, want := range wantStarts {
		if got := cycles[i].GetStartDate().GetValue(); got != want {
			t.Errorf("cycle[%d] start = %q, want %q", i, got, want)
		}
	}
	// Cycles 1–3 should be closed; cycle 4 is open-ended.
	wantEnds := []string{"2026-01-22", "2026-02-26", "2026-03-26", ""}
	for i, want := range wantEnds {
		if got := cycles[i].GetEndDate().GetValue(); got != want {
			t.Errorf("cycle[%d] end = %q, want %q", i, got, want)
		}
	}
}

// ---- Re-detection after adding a new observation -------------------------- //

// TestDetect_RedetectionAfterNewObservation verifies that calling DetectCycles
// again after adding a new observation produces correctly updated cycle
// boundaries (previously open-ended cycle gains an end_date).
func TestDetect_RedetectionAfterNewObservation(t *testing.T) {
	store := memory.New()

	// Step 1: two initial observations → two cycles.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 2 {
		t.Fatalf("initial: expected 2 cycles, got %d", len(cycles))
	}
	if got := cycles[0].GetEndDate().GetValue(); got != "2026-01-31" {
		t.Errorf("initial: cycle[0] end = %q, want 2026-01-31", got)
	}
	if got := cycles[1].GetEndDate().GetValue(); got != "" {
		t.Errorf("initial: cycle[1] should be open-ended, got %q", got)
	}

	// Step 2: add a third observation that begins a new cycle.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b3", "u1", "2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err = rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 3 {
		t.Fatalf("after re-detection: expected 3 cycles, got %d", len(cycles))
	}
	// Previously open-ended cycle[1] should now be closed.
	if got := cycles[1].GetEndDate().GetValue(); got != "2026-02-28" {
		t.Errorf("after re-detection: cycle[1] end = %q, want 2026-02-28", got)
	}
	if got := cycles[2].GetStartDate().GetValue(); got != "2026-03-01" {
		t.Errorf("after re-detection: cycle[2] start = %q, want 2026-03-01", got)
	}
	if got := cycles[2].GetEndDate().GetValue(); got != "" {
		t.Errorf("after re-detection: cycle[2] should be open-ended, got %q", got)
	}
}

func TestIsOutlierLength_OpenEnded(t *testing.T) {
	c := &v1.Cycle{
		Name:      "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if rules.IsOutlierLength(c) {
		t.Error("open-ended cycle should not be an outlier")
	}
}
