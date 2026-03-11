package timeline_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/memory"
	"github.com/2ajoyce/openmenses/engine/internal/timeline"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

var ctx = context.Background()

// ─── helpers ──────────────────────────────────────────────────────────────────

func bleeding(id, userID, date string) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
}

func symptom(id, userID, date string) *v1.SymptomObservation {
	return &v1.SymptomObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
	}
}

func mood(id, userID, date string) *v1.MoodObservation {
	return &v1.MoodObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
	}
}

func medication(id, userID string) *v1.Medication {
	return &v1.Medication{
		Id:       id,
		UserId:   userID,
		Name:     "Ibuprofen",
		Category: v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:   true,
	}
}

func medEvent(id, userID, medID, date string) *v1.MedicationEvent {
	return &v1.MedicationEvent{
		Id:           id,
		UserId:       userID,
		MedicationId: medID,
		Timestamp:    &v1.DateTime{Value: date + "T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
}

func cycle(id, userID, startDate, endDate string) *v1.Cycle {
	return &v1.Cycle{
		Id:        id,
		UserId:    userID,
		StartDate: &v1.LocalDate{Value: startDate},
		EndDate:   &v1.LocalDate{Value: endDate},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
}

func phaseEstimate(id, userID, date string) *v1.PhaseEstimate {
	return &v1.PhaseEstimate{
		Id:         id,
		UserId:     userID,
		Date:       &v1.LocalDate{Value: date},
		Phase:      v1.CyclePhase_CYCLE_PHASE_MENSTRUATION,
		Confidence: v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
	}
}

// recordTimestamp returns the full timestamp or date string from a
// TimelineRecord for use in intra-day ordering assertions.
func recordTimestamp(r *v1.TimelineRecord) string {
	switch x := r.GetRecord().(type) {
	case *v1.TimelineRecord_BleedingObservation:
		return x.BleedingObservation.GetTimestamp().GetValue()
	case *v1.TimelineRecord_SymptomObservation:
		return x.SymptomObservation.GetTimestamp().GetValue()
	case *v1.TimelineRecord_MoodObservation:
		return x.MoodObservation.GetTimestamp().GetValue()
	case *v1.TimelineRecord_MedicationEvent:
		return x.MedicationEvent.GetTimestamp().GetValue()
	case *v1.TimelineRecord_Cycle:
		return x.Cycle.GetStartDate().GetValue()
	case *v1.TimelineRecord_PhaseEstimate:
		return x.PhaseEstimate.GetDate().GetValue()
	}
	return ""
}

// recordDate extracts the YYYY-MM-DD date prefix from a TimelineRecord for
// use in ordering assertions.
func recordDate(r *v1.TimelineRecord) string {
	switch x := r.GetRecord().(type) {
	case *v1.TimelineRecord_BleedingObservation:
		v := x.BleedingObservation.GetTimestamp().GetValue()
		if len(v) >= 10 {
			return v[:10]
		}
	case *v1.TimelineRecord_SymptomObservation:
		v := x.SymptomObservation.GetTimestamp().GetValue()
		if len(v) >= 10 {
			return v[:10]
		}
	case *v1.TimelineRecord_MoodObservation:
		v := x.MoodObservation.GetTimestamp().GetValue()
		if len(v) >= 10 {
			return v[:10]
		}
	case *v1.TimelineRecord_MedicationEvent:
		v := x.MedicationEvent.GetTimestamp().GetValue()
		if len(v) >= 10 {
			return v[:10]
		}
	case *v1.TimelineRecord_Cycle:
		return x.Cycle.GetStartDate().GetValue()
	case *v1.TimelineRecord_PhaseEstimate:
		return x.PhaseEstimate.GetDate().GetValue()
	}
	return ""
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestBuildTimeline_Empty verifies that an empty store returns empty results
// with no next-page token.
func TestBuildTimeline_Empty(t *testing.T) {
	store := memory.New()
	records, next, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("want 0 records, got %d", len(records))
	}
	if next != "" {
		t.Errorf("want empty next token, got %q", next)
	}
}

// TestBuildTimeline_MixedRecords_SortedDescending verifies that records of
// different types are all returned and sorted most-recent-first.
func TestBuildTimeline_MixedRecords_SortedDescending(t *testing.T) {
	store := memory.New()

	if err := store.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2024-01-01")); err != nil {
		t.Fatal(err)
	}
	if err := store.SymptomObservations().Create(ctx, symptom("s1", "u1", "2024-01-03")); err != nil {
		t.Fatal(err)
	}
	if err := store.MoodObservations().Create(ctx, mood("m1", "u1", "2024-01-02")); err != nil {
		t.Fatal(err)
	}

	records, _, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d", len(records))
	}

	// Verify descending order: 2024-01-03 > 2024-01-02 > 2024-01-01
	dates := []string{"2024-01-03", "2024-01-02", "2024-01-01"}
	for i, want := range dates {
		got := recordDate(records[i])
		if got != want {
			t.Errorf("records[%d]: want date %q, got %q", i, want, got)
		}
	}
}

// TestBuildTimeline_AllRecordTypes verifies that all six record types
// (bleeding obs, symptom obs, mood obs, medication event, cycle, phase
// estimate) are included in the timeline.
func TestBuildTimeline_AllRecordTypes(t *testing.T) {
	store := memory.New()

	if err := store.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2024-02-01")); err != nil {
		t.Fatal(err)
	}
	if err := store.SymptomObservations().Create(ctx, symptom("s1", "u1", "2024-02-02")); err != nil {
		t.Fatal(err)
	}
	if err := store.MoodObservations().Create(ctx, mood("m1", "u1", "2024-02-03")); err != nil {
		t.Fatal(err)
	}
	if err := store.Medications().Create(ctx, medication("med1", "u1")); err != nil {
		t.Fatal(err)
	}
	if err := store.MedicationEvents().Create(ctx, medEvent("me1", "u1", "med1", "2024-02-04")); err != nil {
		t.Fatal(err)
	}
	if err := store.Cycles().Create(ctx, cycle("c1", "u1", "2024-02-05", "2024-03-05")); err != nil {
		t.Fatal(err)
	}
	if err := store.PhaseEstimates().Create(ctx, phaseEstimate("pe1", "u1", "2024-02-06")); err != nil {
		t.Fatal(err)
	}

	records, _, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 6 {
		t.Fatalf("want 6 records (one per type), got %d", len(records))
	}

	// Count distinct concrete types.
	types := map[string]int{}
	for _, r := range records {
		switch r.GetRecord().(type) {
		case *v1.TimelineRecord_BleedingObservation:
			types["bleeding"]++
		case *v1.TimelineRecord_SymptomObservation:
			types["symptom"]++
		case *v1.TimelineRecord_MoodObservation:
			types["mood"]++
		case *v1.TimelineRecord_MedicationEvent:
			types["medEvent"]++
		case *v1.TimelineRecord_Cycle:
			types["cycle"]++
		case *v1.TimelineRecord_PhaseEstimate:
			types["phaseEstimate"]++
		}
	}
	for _, name := range []string{"bleeding", "symptom", "mood", "medEvent", "cycle", "phaseEstimate"} {
		if types[name] != 1 {
			t.Errorf("expected 1 %s record, got %d", name, types[name])
		}
	}
}

// TestBuildTimeline_DateRangeFilter verifies that records outside the
// requested date range are excluded.
func TestBuildTimeline_DateRangeFilter(t *testing.T) {
	store := memory.New()

	// Inside range
	if err := store.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2024-03-10")); err != nil {
		t.Fatal(err)
	}
	// Outside range (too early)
	if err := store.BleedingObservations().Create(ctx, bleeding("b2", "u1", "2024-01-01")); err != nil {
		t.Fatal(err)
	}
	// Outside range (too late)
	if err := store.BleedingObservations().Create(ctx, bleeding("b3", "u1", "2024-12-01")); err != nil {
		t.Fatal(err)
	}

	records, _, err := timeline.BuildTimeline(ctx, store, "u1", "2024-03-01", "2024-06-30", storage.PageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("want 1 record in range, got %d", len(records))
	}
}

// TestBuildTimeline_OnlyOwnUserRecords verifies that records belonging to a
// different user are not included.
func TestBuildTimeline_OnlyOwnUserRecords(t *testing.T) {
	store := memory.New()

	if err := store.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2024-04-01")); err != nil {
		t.Fatal(err)
	}
	if err := store.BleedingObservations().Create(ctx, bleeding("b2", "u2", "2024-04-02")); err != nil {
		t.Fatal(err)
	}

	records, _, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("want 1 record for u1, got %d", len(records))
	}
}

// TestBuildTimeline_Pagination verifies that results are split across pages
// correctly and that the combination of all pages equals the full result set.
func TestBuildTimeline_Pagination(t *testing.T) {
	store := memory.New()

	// Insert 5 bleeding observations on consecutive days.
	dates := []string{"2024-05-01", "2024-05-02", "2024-05-03", "2024-05-04", "2024-05-05"}
	for i, d := range dates {
		id := string(rune('a' + i))
		if err := store.BleedingObservations().Create(ctx, bleeding(id, "u1", d)); err != nil {
			t.Fatal(err)
		}
	}

	// First page: 3 records.
	page1, next1, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{PageSize: 3})
	if err != nil {
		t.Fatalf("page 1 error: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("page 1: want 3 records, got %d", len(page1))
	}
	if next1 == "" {
		t.Fatal("page 1: expected a non-empty next token")
	}

	// Second page: remaining 2 records.
	page2, next2, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{PageSize: 3, PageToken: next1})
	if err != nil {
		t.Fatalf("page 2 error: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page 2: want 2 records, got %d", len(page2))
	}
	if next2 != "" {
		t.Errorf("page 2: expected empty next token, got %q", next2)
	}

	// Combined pages should cover all 5 records with no duplicates.
	all := append(page1, page2...)
	if len(all) != 5 {
		t.Errorf("combined: want 5 records, got %d", len(all))
	}

	// Verify descending order across both pages.
	for i := 1; i < len(all); i++ {
		prev := recordDate(all[i-1])
		curr := recordDate(all[i])
		if prev < curr {
			t.Errorf("out of order at index %d: %q < %q", i, prev, curr)
		}
	}
}

// TestBuildTimeline_PaginationBeyondEnd verifies that requesting a page
// starting beyond the last record returns empty results with no token.
func TestBuildTimeline_PaginationBeyondEnd(t *testing.T) {
	store := memory.New()

	if err := store.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2024-06-01")); err != nil {
		t.Fatal(err)
	}

	// Use a token offset beyond the total count.
	records, next, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{PageSize: 10, PageToken: "999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("want 0 records, got %d", len(records))
	}
	if next != "" {
		t.Errorf("want empty next token, got %q", next)
	}
}

// TestBuildTimeline_DefaultPageSize verifies that the default page size (50)
// is applied when PageSize is 0.
func TestBuildTimeline_DefaultPageSize(t *testing.T) {
	store := memory.New()

	// Insert 60 observations so they exceed the default page size.
	for i := 0; i < 60; i++ {
		year := 2024
		month := (i / 28) + 1
		day := (i % 28) + 1
		date := formatDate(year, month, day)
		id := formatID(i)
		if err := store.BleedingObservations().Create(ctx, bleeding(id, "u1", date)); err != nil {
			t.Fatal(err)
		}
	}

	// PageSize 0 should default to 50.
	records, next, err := timeline.BuildTimeline(ctx, store, "u1", "0001-01-01", "9999-12-31", storage.PageRequest{PageSize: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 50 {
		t.Errorf("want 50 records (default page size), got %d", len(records))
	}
	if next == "" {
		t.Error("want non-empty next token for partial page")
	}
}

// TestBuildTimeline_IntraDayChronologicalOrdering verifies that records on the
// same day are sorted by their full timestamp (most-recent-first) rather than
// by entity-type insertion order.
func TestBuildTimeline_IntraDayChronologicalOrdering(t *testing.T) {
	store := memory.New()

	// All three records share the same calendar day but have different times.
	// Bleeding at 14:00, mood at 11:00, symptom at 08:00.
	bleedingRec := &v1.BleedingObservation{
		Id:        "b1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2024-07-15T14:00:00Z"},
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
	moodRec := &v1.MoodObservation{
		Id:        "m1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2024-07-15T11:00:00Z"},
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
	}
	symptomRec := &v1.SymptomObservation{
		Id:        "s1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2024-07-15T08:00:00Z"},
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
	}

	if err := store.BleedingObservations().Create(ctx, bleedingRec); err != nil {
		t.Fatal(err)
	}
	if err := store.MoodObservations().Create(ctx, moodRec); err != nil {
		t.Fatal(err)
	}
	if err := store.SymptomObservations().Create(ctx, symptomRec); err != nil {
		t.Fatal(err)
	}

	records, _, err := timeline.BuildTimeline(ctx, store, "u1", "2024-07-15", "2024-07-15", storage.PageRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("want 3 records, got %d", len(records))
	}

	// Expected descending (most-recent-first) order: 14:00, 11:00, 08:00.
	wantTimestamps := []string{
		"2024-07-15T14:00:00Z",
		"2024-07-15T11:00:00Z",
		"2024-07-15T08:00:00Z",
	}
	for i, want := range wantTimestamps {
		got := recordTimestamp(records[i])
		if got != want {
			t.Errorf("records[%d]: want timestamp %q, got %q", i, want, got)
		}
	}
}

// ─── formatting helpers ───────────────────────────────────────────────────────

func formatDate(year, month, day int) string {
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func formatID(n int) string {
	return fmt.Sprintf("id-%04d", n)
}
