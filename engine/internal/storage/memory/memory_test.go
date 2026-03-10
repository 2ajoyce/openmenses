package memory_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/2ajoyce/opencycle/engine/internal/storage"
	"github.com/2ajoyce/opencycle/engine/internal/storage/memory"
	v1 "github.com/2ajoyce/opencycle/gen/go/opencycle/v1"
)

func newStore() *memory.Store { return memory.New() }

var ctx = context.Background()

// ---- helpers ----------------------------------------------------------------

func localDate(s string) *v1.LocalDate { return &v1.LocalDate{Value: s} }
func dateTime(s string) *v1.DateTime   { return &v1.DateTime{Value: s} }

// ---- UserProfile ------------------------------------------------------------

func TestUserProfile_UpsertAndGet(t *testing.T) {
	s := newStore()
	p := &v1.UserProfile{Id: "u1"}
	if err := s.UserProfiles().Upsert(ctx, p); err != nil {
		t.Fatal(err)
	}
	got, err := s.UserProfiles().GetByID(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetId() != "u1" {
		t.Fatalf("got id %q, want %q", got.GetId(), "u1")
	}
}

func TestUserProfile_GetNotFound(t *testing.T) {
	s := newStore()
	_, err := s.UserProfiles().GetByID(ctx, "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserProfile_UpsertOverwrite(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.UserProfiles().Upsert(ctx, &v1.UserProfile{Id: "u1", CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR}))
	mustNoErr(t, s.UserProfiles().Upsert(ctx, &v1.UserProfile{Id: "u1", CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR}))
	got, _ := s.UserProfiles().GetByID(ctx, "u1")
	if got.GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR {
		t.Fatal("upsert did not overwrite")
	}
}

func TestUserProfile_EmptyIDRejected(t *testing.T) {
	s := newStore()
	err := s.UserProfiles().Upsert(ctx, &v1.UserProfile{})
	if !errors.Is(err, storage.ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
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
	s := newStore()
	obs := bleeding("b1", "u1", "2026-01-01T08:00:00Z")
	if err := s.BleedingObservations().Create(ctx, obs); err != nil {
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
	s := newStore()
	obs := bleeding("b1", "u1", "2026-01-01T08:00:00Z")
	mustNoErr(t, s.BleedingObservations().Create(ctx, obs))
	err := s.BleedingObservations().Create(ctx, obs)
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestBleeding_GetNotFound(t *testing.T) {
	s := newStore()
	_, err := s.BleedingObservations().GetByID(ctx, "nope")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBleeding_Delete(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T08:00:00Z")))
	if err := s.BleedingObservations().DeleteByID(ctx, "b1"); err != nil {
		t.Fatal(err)
	}
	_, err := s.BleedingObservations().GetByID(ctx, "b1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatal("expected not found after delete")
	}
}

func TestBleeding_DeleteNotFound(t *testing.T) {
	s := newStore()
	err := s.BleedingObservations().DeleteByID(ctx, "nope")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestBleeding_ListByUserAndDateRange(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-05T08:00:00Z")))
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b2", "u1", "2026-01-10T08:00:00Z")))
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b3", "u1", "2026-02-01T08:00:00Z")))
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b4", "u2", "2026-01-07T08:00:00Z"))) // different user

	page, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
	if page.Items[0].GetId() != "b1" || page.Items[1].GetId() != "b2" {
		t.Fatal("wrong order or items")
	}
}

func TestBleeding_ListEmpty(t *testing.T) {
	s := newStore()
	page, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatal("expected empty")
	}
}

func TestBleeding_Pagination(t *testing.T) {
	s := newStore()
	for i := 0; i < 5; i++ {
		ts := fmt.Sprintf("2026-01-%02dT08:00:00Z", i+1)
		mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding(fmt.Sprintf("b%d", i), "u1", ts)))
	}

	page1, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page1: want 2, got %d", len(page1.Items))
	}
	if page1.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	page2, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page1.NextPageToken})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 2 {
		t.Fatalf("page2: want 2, got %d", len(page2.Items))
	}

	page3, err := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page2.NextPageToken})
	if err != nil {
		t.Fatal(err)
	}
	if len(page3.Items) != 1 {
		t.Fatalf("page3: want 1, got %d", len(page3.Items))
	}
	if page3.NextPageToken != "" {
		t.Fatal("expected no next page on last page")
	}
}

func TestBleeding_DateRangeBoundaries(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T00:00:00Z")))
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b2", "u1", "2026-01-31T23:59:59Z")))
	mustNoErr(t, s.BleedingObservations().Create(ctx, bleeding("b3", "u1", "2026-02-01T00:00:00Z")))

	page, _ := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

func TestBleeding_ConcurrentAccess(t *testing.T) {
	s := newStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ts := fmt.Sprintf("2026-01-%02dT08:00:00Z", (n%28)+1)
			_ = s.BleedingObservations().Create(ctx, bleeding(fmt.Sprintf("c%d", n), "u1", ts))
		}(i)
	}
	wg.Wait()
	page, _ := s.BleedingObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 100})
	if len(page.Items) != 50 {
		t.Fatalf("want 50 concurrent items, got %d", len(page.Items))
	}
}

// ---- Cycle ------------------------------------------------------------------

func cycle(id, userID, start, end string) *v1.Cycle {
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
	s := newStore()
	c := cycle("cy1", "u1", "2026-01-01", "2026-01-28")
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
	s := newStore()
	c := cycle("cy1", "u1", "2026-01-01", "2026-01-28")
	mustNoErr(t, s.Cycles().Create(ctx, c))
	if err := s.Cycles().Create(ctx, c); !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCycle_Update(t *testing.T) {
	s := newStore()
	c := cycle("cy1", "u1", "2026-01-01", "2026-01-28")
	mustNoErr(t, s.Cycles().Create(ctx, c))
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
	s := newStore()
	err := s.Cycles().Update(ctx, cycle("cy99", "u1", "2026-01-01", ""))
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCycle_ListByUserAndDateRange_Overlap(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.Cycles().Create(ctx, cycle("cy1", "u1", "2026-01-01", "2026-01-28")))
	mustNoErr(t, s.Cycles().Create(ctx, cycle("cy2", "u1", "2026-01-29", "2026-02-25")))
	mustNoErr(t, s.Cycles().Create(ctx, cycle("cy3", "u1", "2026-02-26", "2026-03-25")))

	page, _ := s.Cycles().ListByUserAndDateRange(ctx, "u1", "2026-01-15", "2026-02-10", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

func TestCycle_ListByUser(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.Cycles().Create(ctx, cycle("cy1", "u1", "2026-01-01", "2026-01-28")))
	mustNoErr(t, s.Cycles().Create(ctx, cycle("cy2", "u2", "2026-01-01", "2026-01-28")))
	page, _ := s.Cycles().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page.Items) != 1 {
		t.Fatalf("want 1, got %d", len(page.Items))
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
	s := newStore()
	m := med("m1", "u1")
	if err := s.Medications().Create(ctx, m); err != nil {
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
	s := newStore()
	mustNoErr(t, s.Medications().Create(ctx, med("m1", "u1")))
	mustNoErr(t, s.Medications().Create(ctx, med("m2", "u1")))
	mustNoErr(t, s.Medications().Create(ctx, med("m3", "u2")))
	page, _ := s.Medications().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

// ---- MedicationEvent --------------------------------------------------------

func medEvent(id, userID, medID, ts string) *v1.MedicationEvent {
	return &v1.MedicationEvent{
		Id:           id,
		UserId:       userID,
		MedicationId: medID,
		Timestamp:    dateTime(ts),
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
}

func TestMedicationEvent_ListByMedicationID(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e1", "u1", "m1", "2026-01-01T08:00:00Z")))
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e2", "u1", "m1", "2026-01-02T08:00:00Z")))
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e3", "u1", "m2", "2026-01-02T08:00:00Z")))
	page, _ := s.MedicationEvents().ListByMedicationID(ctx, "m1", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

// ---- Prediction -------------------------------------------------------------

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
	s := newStore()
	mustNoErr(t, s.Predictions().Create(ctx, prediction("p1", "u1", "2026-02-01")))
	mustNoErr(t, s.Predictions().Create(ctx, prediction("p2", "u1", "2026-03-01")))
	mustNoErr(t, s.Predictions().Create(ctx, prediction("p3", "u2", "2026-02-01")))

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
	// u2 unaffected
	page3, _ := s.Predictions().ListByUser(ctx, "u2", storage.PageRequest{})
	if len(page3.Items) != 1 {
		t.Fatal("u2 predictions should be intact")
	}
}

// ---- Insight ----------------------------------------------------------------

func insight(id, userID string) *v1.Insight {
	return &v1.Insight{
		Id:         id,
		UserId:     userID,
		Kind:       v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN,
		Summary:    "Your cycles are regular.",
		Confidence: v1.ConfidenceLevel_CONFIDENCE_LEVEL_HIGH,
	}
}

func TestInsight_CreateListDeleteByUser(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.Insights().Create(ctx, insight("i1", "u1")))
	mustNoErr(t, s.Insights().Create(ctx, insight("i2", "u1")))

	page, _ := s.Insights().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}

	mustNoErr(t, s.Insights().DeleteByUser(ctx, "u1"))
	page2, _ := s.Insights().ListByUser(ctx, "u1", storage.PageRequest{})
	if len(page2.Items) != 0 {
		t.Fatal("expected empty after delete")
	}
}
