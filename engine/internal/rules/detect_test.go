package rules_test

import (
	"context"
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
		Id:        id,
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

// ---- No observations ------------------------------------------------------ //

func TestDetect_NoObs_NoUserCycles(t *testing.T) {
	store := memory.New()
	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Fatalf("expected 0 cycles, got %d", len(cycles))
	}
}

// ---- Single observation --------------------------------------------------- //

func TestDetect_SingleObs_OpenEndedCycle(t *testing.T) {
	store := memory.New()
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d", len(cycles))
	}
	if got := cycles[0].GetStartDate().GetValue(); got != "2026-01-01" {
		t.Errorf("start_date = %q, want 2026-01-01", got)
	}
	if got := cycles[0].GetEndDate().GetValue(); got != "" {
		t.Errorf("expected open-ended cycle, got end_date=%q", got)
	}
	if cycles[0].GetSource() != v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING {
		t.Error("expected DERIVED_FROM_BLEEDING source")
	}
}

// ---- Two regular cycles --------------------------------------------------- //

func TestDetect_TwoRegularCycles(t *testing.T) {
	store := memory.New()
	// Cycle 1: Jan 1-5. Cycle 2: Feb 1-5. Gap = 27 days (>3).
	for i, date := range []string{
		"2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04", "2026-01-05",
		"2026-02-01", "2026-02-02", "2026-02-03",
	} {
		mustCreate(t, store.BleedingObservations().Create,
			makeObs(date, "u1", date, v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
		_ = i
	}

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 2 {
		t.Fatalf("expected 2 cycles, got %d", len(cycles))
	}
	// First cycle ends the day before the second starts.
	if got := cycles[0].GetStartDate().GetValue(); got != "2026-01-01" {
		t.Errorf("cycle[0] start = %q, want 2026-01-01", got)
	}
	if got := cycles[0].GetEndDate().GetValue(); got != "2026-01-31" {
		t.Errorf("cycle[0] end = %q, want 2026-01-31", got)
	}
	if got := cycles[1].GetStartDate().GetValue(); got != "2026-02-01" {
		t.Errorf("cycle[1] start = %q, want 2026-02-01", got)
	}
	if got := cycles[1].GetEndDate().GetValue(); got != "" {
		t.Errorf("cycle[1] should be open-ended, got end_date=%q", got)
	}
}

// ---- Gap exactly at boundary ---------------------------------------------- //

// Resume 4 days after last bleed = 3 non-bleeding days = new episode.
func TestDetect_Gap4Days_NewEpisode(t *testing.T) {
	store := memory.New()
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	// 4-day gap: Jan 2, 3, 4 are non-bleeding → Jan 5 starts new episode.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-01-05", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 2 {
		t.Fatalf("expected 2 cycles (gap = 4 days), got %d", len(cycles))
	}
}

// Resume 3 days after last bleed = 2 non-bleeding days = same episode.
func TestDetect_Gap3Days_SameEpisode(t *testing.T) {
	store := memory.New()
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	// 3-day gap: Jan 2, 3 are non-bleeding → Jan 4 continues same episode.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-01-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle (gap = 3 days, same episode), got %d", len(cycles))
	}
}

// ---- Spotting disambiguation ---------------------------------------------- //

func TestDetect_SpottingFollowedByHeavy_ValidCycleStart(t *testing.T) {
	store := memory.New()
	// Episode 1: heavier bleed in Jan.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	// After sufficient gap, only spotting on Feb 1 but medium on Feb 2
	// → spotting is valid cycle start.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING))
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b3", "u1", "2026-02-02", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 2 {
		t.Fatalf("expected 2 cycles (spotting + heavy = valid start), got %d", len(cycles))
	}
	if got := cycles[1].GetStartDate().GetValue(); got != "2026-02-01" {
		t.Errorf("cycle[1] start = %q, want 2026-02-01 (spotting day)", got)
	}
}

func TestDetect_SpottingAlone_MidCycle(t *testing.T) {
	store := memory.New()
	// Episode 1: Jan 1.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	// Spotting only on Feb 1 with no heavier flow within 2 days: mid-cycle spotting.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	// Only 1 cycle: the spotting does not start a new one.
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle (spotting = mid-cycle), got %d", len(cycles))
	}
}

// ---- USER_CONFIRMED cycle preservation ------------------------------------ //

func TestDetect_ConfirmedCycle_NotOverridden(t *testing.T) {
	store := memory.New()
	// Store a user-confirmed cycle.
	confirmed := &v1.Cycle{
		Id:        "cy-confirmed",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_USER_CONFIRMED,
	}
	mustCreate(t, store.Cycles().Create, confirmed)

	// Add bleeding observations in the same range — engine should keep confirmed.
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-01-04", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	// Only the confirmed cycle should be present (derived overlaps with it).
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle (confirmed preserved), got %d", len(cycles))
	}
	if cycles[0].GetId() != "cy-confirmed" {
		t.Errorf("expected confirmed cycle id, got %q", cycles[0].GetId())
	}
}

func TestDetect_DerivedAfterConfirmed_BothPresent(t *testing.T) {
	store := memory.New()
	// Confirmed cycle in January.
	mustCreate(t, store.Cycles().Create, &v1.Cycle{
		Id:        "cy-confirmed",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_USER_CONFIRMED,
	})
	// Bleeding observations in February (no overlap with confirmed cycle).
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 2 {
		t.Fatalf("expected 2 cycles, got %d", len(cycles))
	}
	// Sorted by start_date: confirmed first, then derived.
	if cycles[0].GetId() != "cy-confirmed" {
		t.Errorf("expected confirmed cycle first, got id=%q", cycles[0].GetId())
	}
	if cycles[1].GetSource() != v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING {
		t.Error("expected second cycle to be derived")
	}
}

// ---- Isolation between users --------------------------------------------- //

func TestDetect_Isolation_DifferentUsers(t *testing.T) {
	store := memory.New()
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u2", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))

	cyclesU1, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	cyclesU2, err := rules.DetectCycles(ctx, "u2", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cyclesU1) != 1 || len(cyclesU2) != 1 {
		t.Fatalf("each user should have 1 cycle; u1=%d u2=%d", len(cyclesU1), len(cyclesU2))
	}
	if cyclesU1[0].GetUserId() != "u1" || cyclesU2[0].GetUserId() != "u2" {
		t.Error("cycles attributed to wrong user")
	}
}

// ---- OnlySpotting, no heavier follow-up ----------------------------------- //

func TestDetect_AllSpotting_NoCycles(t *testing.T) {
	store := memory.New()
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b1", "u1", "2026-01-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING))
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("b2", "u1", "2026-02-01", v1.BleedingFlow_BLEEDING_FLOW_SPOTTING))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 0 {
		t.Fatalf("expected 0 cycles for all-spotting observations, got %d", len(cycles))
	}
}

// ---- Multiple days same episode, correct end_date ------------------------- //

func TestDetect_MultipleEpisodes_CorrectBoundaries(t *testing.T) {
	store := memory.New()
	// Episode 1: Jan 1-3.
	for _, d := range []string{"2026-01-01", "2026-01-02", "2026-01-03"} {
		mustCreate(t, store.BleedingObservations().Create,
			makeObs(d, "u1", d, v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	}
	// Episode 2: Feb 1-3 (gap > 3 days).
	for _, d := range []string{"2026-02-01", "2026-02-02", "2026-02-03"} {
		mustCreate(t, store.BleedingObservations().Create,
			makeObs(d, "u1", d, v1.BleedingFlow_BLEEDING_FLOW_MEDIUM))
	}
	// Episode 3: Mar 1 (open-ended).
	mustCreate(t, store.BleedingObservations().Create,
		makeObs("m1", "u1", "2026-03-01", v1.BleedingFlow_BLEEDING_FLOW_LIGHT))

	cycles, err := rules.DetectCycles(ctx, "u1", store)
	if err != nil {
		t.Fatal(err)
	}
	if len(cycles) != 3 {
		t.Fatalf("expected 3 cycles, got %d", len(cycles))
	}
	// Cycle 1 ends the day before cycle 2 starts.
	if got := cycles[0].GetEndDate().GetValue(); got != "2026-01-31" {
		t.Errorf("cycle[0] end = %q, want 2026-01-31", got)
	}
	// Cycle 2 ends the day before cycle 3 starts.
	if got := cycles[1].GetEndDate().GetValue(); got != "2026-02-28" {
		t.Errorf("cycle[1] end = %q, want 2026-02-28", got)
	}
	// Cycle 3 is open-ended.
	if got := cycles[2].GetEndDate().GetValue(); got != "" {
		t.Errorf("cycle[2] should be open-ended, got end_date=%q", got)
	}
}

// ---- IsOutlierLength ------------------------------------------------------ //

func TestIsOutlierLength_NormalCycle(t *testing.T) {
	c := &v1.Cycle{
		Id:        "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"}, // 28 days
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if rules.IsOutlierLength(c) {
		t.Error("28-day cycle should not be an outlier")
	}
}

func TestIsOutlierLength_TooShort(t *testing.T) {
	c := &v1.Cycle{
		Id:        "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-10"}, // 10 days
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if !rules.IsOutlierLength(c) {
		t.Error("10-day cycle should be an outlier (< 15)")
	}
}

func TestIsOutlierLength_TooLong(t *testing.T) {
	c := &v1.Cycle{
		Id:        "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-05-01"}, // 121 days
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if !rules.IsOutlierLength(c) {
		t.Error("121-day cycle should be an outlier (> 90)")
	}
}

func TestIsOutlierLength_AtMinBound(t *testing.T) {
	c := &v1.Cycle{
		Id:        "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-15"}, // 15 days
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if rules.IsOutlierLength(c) {
		t.Error("15-day cycle should not be an outlier (= min)")
	}
}

func TestIsOutlierLength_AtMaxBound(t *testing.T) {
	c := &v1.Cycle{
		Id:        "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-03-31"}, // 90 days
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if rules.IsOutlierLength(c) {
		t.Error("90-day cycle should not be an outlier (= max)")
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
		Id:        "c1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if rules.IsOutlierLength(c) {
		t.Error("open-ended cycle should not be an outlier")
	}
}
