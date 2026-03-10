package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/sqlite"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()
	s, err := sqlite.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

var ctx = context.Background()

func localDate(v string) *v1.LocalDate { return &v1.LocalDate{Value: v} }
func dateTime(v string) *v1.DateTime   { return &v1.DateTime{Value: v} }

// ---- UserProfile ------------------------------------------------------------

func TestUserProfile_UpsertAndGet(t *testing.T) {
	s := newStore(t)
	p := &v1.UserProfile{Id: "u1"}
	if err := s.UserProfiles().Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := s.UserProfiles().GetByID(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetId() != "u1" {
		t.Fatalf("got %q", got.GetId())
	}
}

func TestUserProfile_GetNotFound(t *testing.T) {
	s := newStore(t)
	_, err := s.UserProfiles().GetByID(ctx, "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestUserProfile_UpsertOverwrite(t *testing.T) {
	s := newStore(t)
	s.UserProfiles().Upsert(ctx, &v1.UserProfile{Id: "u1", CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR})        //nolint:errcheck
	s.UserProfiles().Upsert(ctx, &v1.UserProfile{Id: "u1", CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR}) //nolint:errcheck
	got, _ := s.UserProfiles().GetByID(ctx, "u1")
	if got.GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR {
		t.Fatal("upsert did not overwrite")
	}
}

// ---- BleedingObservation ----------------------------------------------------

func bleeding(id, userID, ts string) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Id:        id,
		UserId:    userID,
		Timestamp: dateTime(ts),
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
}

func TestBleeding_CreateAndGet(t *testing.T) {
	s := newStore(t)
	if err := s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T08:00:00Z")); err != nil {
		t.Fatal(err)
	}
	got, err := s.BleedingObservations().GetByID(ctx, "b1")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetId() != "b1" {
		t.Fatalf("got %q", got.GetId())
	}
}

func TestBleeding_DuplicateIDRejected(t *testing.T) {
	s := newStore(t)
	obs := bleeding("b1", "u1", "2026-01-01T08:00:00Z")
	s.BleedingObservations().Create(ctx, obs) //nolint:errcheck
	err := s.BleedingObservations().Create(ctx, obs)
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestBleeding_GetNotFound(t *testing.T) {
	s := newStore(t)
	_, err := s.BleedingObservations().GetByID(ctx, "nope")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBleeding_Delete(t *testing.T) {
	s := newStore(t)
	s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T08:00:00Z")) //nolint:errcheck
	if err := s.BleedingObservations().DeleteByID(ctx, "b1"); err != nil {
		t.Fatal(err)
	}
	_, err := s.BleedingObservations().GetByID(ctx, "b1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("expected not found after delete")
	}
}

func TestBleeding_DeleteNotFound(t *testing.T) {
	s := newStore(t)
	err := s.BleedingObservations().DeleteByID(ctx, "nope")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBleeding_ListByUserAndDateRange(t *testing.T) {
	s := newStore(t)
	s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-05T08:00:00Z")) //nolint:errcheck
	s.BleedingObservations().Create(ctx, bleeding("b2", "u1", "2026-01-10T08:00:00Z")) //nolint:errcheck
	s.BleedingObservations().Create(ctx, bleeding("b3", "u1", "2026-02-01T08:00:00Z")) //nolint:errcheck
	s.BleedingObservations().Create(ctx, bleeding("b4", "u2", "2026-01-07T08:00:00Z")) //nolint:errcheck

	page, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

func TestBleeding_ListEmpty(t *testing.T) {
	s := newStore(t)
	page, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatal("expected empty")
	}
}

func TestBleeding_Pagination(t *testing.T) {
	s := newStore(t)
	for i := 0; i < 5; i++ {
		ts := fmt.Sprintf("2026-01-%02dT08:00:00Z", i+1)
		s.BleedingObservations().Create(ctx, bleeding(fmt.Sprintf("b%d", i), "u1", ts)) //nolint:errcheck
	}

	page1, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.NextPageToken == "" {
		t.Fatalf("page1: got %d items, token=%q", len(page1.Items), page1.NextPageToken)
	}

	page2, _ := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page1.NextPageToken})
	if len(page2.Items) != 2 {
		t.Fatalf("page2: want 2, got %d", len(page2.Items))
	}

	page3, _ := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page2.NextPageToken})
	if len(page3.Items) != 1 || page3.NextPageToken != "" {
		t.Fatalf("page3: got %d items, token=%q", len(page3.Items), page3.NextPageToken)
	}
}

func TestBleeding_DateRangeBoundaries(t *testing.T) {
	s := newStore(t)
	s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T00:00:00Z")) //nolint:errcheck
	s.BleedingObservations().Create(ctx, bleeding("b2", "u1", "2026-01-31T23:59:59Z")) //nolint:errcheck
	s.BleedingObservations().Create(ctx, bleeding("b3", "u1", "2026-02-01T00:00:00Z")) //nolint:errcheck

	page, _ := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

// ---- Cycle ------------------------------------------------------------------

func cycleRecord(id, userID, start, end string) *v1.Cycle {
	c := &v1.Cycle{
		Id:        id,
		UserId:    userID,
		StartDate: localDate(start),
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if end != "" {
		c.EndDate = localDate(end)
	}
	return c
}

func TestCycle_CreateGetDelete(t *testing.T) {
	s := newStore(t)
	c := cycleRecord("cy1", "u1", "2026-01-01", "2026-01-28")
	if err := s.Cycles().Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Cycles().GetByID(ctx, "cy1")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetStartDate().GetValue() != "2026-01-01" {
		t.Fatal("start date mismatch")
	}
	if err = s.Cycles().DeleteByID(ctx, "cy1"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Cycles().GetByID(ctx, "cy1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("expected not found after delete")
	}
}

func TestCycle_DuplicateRejected(t *testing.T) {
	s := newStore(t)
	c := cycleRecord("cy1", "u1", "2026-01-01", "2026-01-28")
	s.Cycles().Create(ctx, c) //nolint:errcheck
	if err := s.Cycles().Create(ctx, c); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCycle_Update(t *testing.T) {
	s := newStore(t)
	c := cycleRecord("cy1", "u1", "2026-01-01", "2026-01-28")
	s.Cycles().Create(ctx, c) //nolint:errcheck
	c.EndDate = localDate("2026-01-30")
	if err := s.Cycles().Update(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Cycles().GetByID(ctx, "cy1")
	if got.GetEndDate().GetValue() != "2026-01-30" {
		t.Fatal("update did not persist")
	}
}

func TestCycle_UpdateNotFound(t *testing.T) {
	s := newStore(t)
	err := s.Cycles().Update(ctx, cycleRecord("cy99", "u1", "2026-01-01", ""))
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCycle_ListByUserAndDateRange_Overlap(t *testing.T) {
	s := newStore(t)
	s.Cycles().Create(ctx, cycleRecord("cy1", "u1", "2026-01-01", "2026-01-28")) //nolint:errcheck
	s.Cycles().Create(ctx, cycleRecord("cy2", "u1", "2026-01-29", "2026-02-25")) //nolint:errcheck
	s.Cycles().Create(ctx, cycleRecord("cy3", "u1", "2026-02-26", "2026-03-25")) //nolint:errcheck

	page, _ := s.Cycles().ListByUserAndDateRange(ctx, "u1", "2026-01-15", "2026-02-10", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

// ---- Medication -------------------------------------------------------------

func med(id, userID string) *v1.Medication {
	return &v1.Medication{
		Id:       id,
		UserId:   userID,
		Name:     "Ibuprofen",
		Category: v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:   true,
	}
}

func TestMedication_CRUD(t *testing.T) {
	s := newStore(t)
	if err := s.Medications().Create(ctx, med("m1", "u1")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Medications().GetByID(ctx, "m1")
	if err != nil {
		t.Fatal(err)
	}
	got.Active = false
	if err = s.Medications().Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	updated, _ := s.Medications().GetByID(ctx, "m1")
	if updated.GetActive() {
		t.Fatal("active should be false after update")
	}
	if err = s.Medications().DeleteByID(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Medications().GetByID(ctx, "m1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("expected not found")
	}
}

func TestMedication_ListByUser(t *testing.T) {
	s := newStore(t)
	s.Medications().Create(ctx, med("m1", "u1")) //nolint:errcheck
	s.Medications().Create(ctx, med("m2", "u1")) //nolint:errcheck
	s.Medications().Create(ctx, med("m3", "u2")) //nolint:errcheck
	page, _ := s.Medications().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

// ---- Prediction-------------------------------------------------------------

func prediction(id, userID, start string) *v1.Prediction {
	return &v1.Prediction{
		Id:                 id,
		UserId:             userID,
		Kind:               v1.PredictionType_PREDICTION_TYPE_NEXT_BLEED,
		PredictedStartDate: localDate(start),
		Confidence:         v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
	}
}

func TestPrediction_CreateListDeleteByUser(t *testing.T) {
	s := newStore(t)
	s.Predictions().Create(ctx, prediction("p1", "u1", "2026-02-01")) //nolint:errcheck
	s.Predictions().Create(ctx, prediction("p2", "u1", "2026-03-01")) //nolint:errcheck
	s.Predictions().Create(ctx, prediction("p3", "u2", "2026-02-01")) //nolint:errcheck

	page, _ := s.Predictions().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}

	if err := s.Predictions().DeleteByUser(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	page2, _ := s.Predictions().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page2.Items) != 0 {
		t.Fatal("expected empty after DeleteByUser")
	}
	page3, _ := s.Predictions().ListByUser(ctx, "u2", storage.PageRequest{})
	if len(page3.Items) != 1 {
		t.Fatal("u2 predictions should be intact")
	}
}

// ---- Insight ----------------------------------------------------------------

func insightRecord(id, userID string) *v1.Insight {
	return &v1.Insight{
		Id:         id,
		UserId:     userID,
		Kind:       v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN,
		Summary:    "Cycles are regular.",
		Confidence: v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH,
	}
}

func TestInsight_CreateListDeleteByUser(t *testing.T) {
	s := newStore(t)
	s.Insights().Create(ctx, insightRecord("i1", "u1")) //nolint:errcheck
	s.Insights().Create(ctx, insightRecord("i2", "u1")) //nolint:errcheck

	page, _ := s.Insights().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}

	s.Insights().DeleteByUser(ctx, "u1") //nolint:errcheck
	page2, _ := s.Insights().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page2.Items) != 0 {
		t.Fatal("expected empty after delete")
	}
}
