package memory_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/memory"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

func newStore() *memory.Store { return memory.New() }

var ctx = context.Background()

// ---- helpers ----------------------------------------------------------------

func localDate(s string) *v1.LocalDate { return &v1.LocalDate{Value: s} }
func dateTime(s string) *v1.DateTime   { return &v1.DateTime{Value: s} }

// ---- UserProfile ------------------------------------------------------------

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestUserProfile_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "upsert and get",
			setup: func(s *memory.Store) error {
				return s.UserProfiles().Upsert(ctx, &v1.UserProfile{Name: "u1"})
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.UserProfiles().GetByID(ctx, "u1")
				mustNoErr(t, err)
				if got.GetName() != "u1" {
					t.Fatalf("got name %q, want %q", got.GetName(), "u1")
				}
			},
		},
		{
			name: "get not found",
			setup: func(s *memory.Store) error {
				return nil // no setup
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.UserProfiles().GetByID(ctx, "missing")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("want ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "upsert overwrites",
			setup: func(s *memory.Store) error {
				if err := s.UserProfiles().Upsert(ctx, &v1.UserProfile{Name: "u1", CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_REGULAR}); err != nil {
					return err
				}
				return s.UserProfiles().Upsert(ctx, &v1.UserProfile{Name: "u1", CycleRegularity: v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR})
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, _ := s.UserProfiles().GetByID(ctx, "u1")
				if got.GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR {
					t.Fatal("upsert did not overwrite")
				}
			},
		},
		{
			name: "empty ID rejected",
			setup: func(s *memory.Store) error {
				return s.UserProfiles().Upsert(ctx, &v1.UserProfile{})
			},
			wantErr: storage.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
	}
}

// ---- BleedingObservation ----------------------------------------------------

func bleeding(id, userID, ts string) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Name:      id,
		UserId:    userID,
		Timestamp: dateTime(ts),
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
}

func TestBleeding_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create and get",
			setup: func(s *memory.Store) error {
				return s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T08:00:00Z"))
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.BleedingObservations().GetByID(ctx, "b1")
				mustNoErr(t, err)
				if got.GetName() != "b1" {
					t.Fatalf("got %q", got.GetName())
				}
			},
		},
		{
			name: "get not found",
			setup: func(s *memory.Store) error {
				return nil
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.BleedingObservations().GetByID(ctx, "nope")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("want ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "duplicate ID rejected",
			setup: func(s *memory.Store) error {
				obs := bleeding("b1", "u1", "2026-01-01T08:00:00Z")
				if err := s.BleedingObservations().Create(ctx, obs); err != nil {
					return err
				}
				return s.BleedingObservations().Create(ctx, obs)
			},
			wantErr: storage.ErrConflict,
		},
		{
			name: "delete",
			setup: func(s *memory.Store) error {
				if err := s.BleedingObservations().Create(ctx, bleeding("b1", "u1", "2026-01-01T08:00:00Z")); err != nil {
					return err
				}
				return s.BleedingObservations().DeleteByID(ctx, "b1")
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.BleedingObservations().GetByID(ctx, "b1")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("expected not found after delete")
				}
			},
		},
		{
			name: "delete not found",
			setup: func(s *memory.Store) error {
				return s.BleedingObservations().DeleteByID(ctx, "nope")
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
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
	if page.Items[0].GetName() != "b1" || page.Items[1].GetName() != "b2" {
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
		Name:      id,
		UserId:    userID,
		StartDate: localDate(start),
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if end != "" {
		c.EndDate = localDate(end)
	}
	return c
}

func TestCycle_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create, get, and delete",
			setup: func(s *memory.Store) error {
				c := cycle("cy1", "u1", "2026-01-01", "2026-01-28")
				if err := s.Cycles().Create(ctx, c); err != nil {
					return err
				}
				got, err := s.Cycles().GetByID(ctx, "cy1")
				if err != nil {
					return err
				}
				if got.GetStartDate().GetValue() != "2026-01-01" {
					return errors.New("start date mismatch")
				}
				return s.Cycles().DeleteByID(ctx, "cy1")
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.Cycles().GetByID(ctx, "cy1")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("expected not found after delete")
				}
			},
		},
		{
			name: "duplicate rejected",
			setup: func(s *memory.Store) error {
				c := cycle("cy1", "u1", "2026-01-01", "2026-01-28")
				if err := s.Cycles().Create(ctx, c); err != nil {
					return err
				}
				return s.Cycles().Create(ctx, c)
			},
			wantErr: storage.ErrConflict,
		},
		{
			name: "update",
			setup: func(s *memory.Store) error {
				c := cycle("cy1", "u1", "2026-01-01", "2026-01-28")
				if err := s.Cycles().Create(ctx, c); err != nil {
					return err
				}
				c.EndDate = localDate("2026-01-30")
				return s.Cycles().Update(ctx, c)
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, _ := s.Cycles().GetByID(ctx, "cy1")
				if got.GetEndDate().GetValue() != "2026-01-30" {
					t.Fatal("update did not persist")
				}
			},
		},
		{
			name: "update not found",
			setup: func(s *memory.Store) error {
				return s.Cycles().Update(ctx, cycle("cy99", "u1", "2026-01-01", ""))
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
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
		Name:        id,
		UserId:      userID,
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      true,
	}
}

func TestMedication_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create and get",
			setup: func(s *memory.Store) error {
				return s.Medications().Create(ctx, med("m1", "u1"))
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.Medications().GetByID(ctx, "m1")
				mustNoErr(t, err)
				if got.GetName() != "m1" {
					t.Fatalf("got %q", got.GetName())
				}
			},
		},
		{
			name: "get not found",
			setup: func(s *memory.Store) error {
				return nil
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.Medications().GetByID(ctx, "nope")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("want ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "duplicate ID rejected",
			setup: func(s *memory.Store) error {
				m := med("m1", "u1")
				if err := s.Medications().Create(ctx, m); err != nil {
					return err
				}
				return s.Medications().Create(ctx, m)
			},
			wantErr: storage.ErrConflict,
		},
		{
			name: "update",
			setup: func(s *memory.Store) error {
				m := med("m1", "u1")
				if err := s.Medications().Create(ctx, m); err != nil {
					return err
				}
				m.Active = false
				return s.Medications().Update(ctx, m)
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.Medications().GetByID(ctx, "m1")
				mustNoErr(t, err)
				if got.GetActive() {
					t.Fatal("expected active to be false after update")
				}
			},
		},
		{
			name: "update not found",
			setup: func(s *memory.Store) error {
				return s.Medications().Update(ctx, med("nope", "u1"))
			},
			wantErr: storage.ErrNotFound,
		},
		{
			name: "delete",
			setup: func(s *memory.Store) error {
				if err := s.Medications().Create(ctx, med("m1", "u1")); err != nil {
					return err
				}
				return s.Medications().DeleteByID(ctx, "m1")
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.Medications().GetByID(ctx, "m1")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("expected not found after delete")
				}
			},
		},
		{
			name: "delete not found",
			setup: func(s *memory.Store) error {
				return s.Medications().DeleteByID(ctx, "nope")
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
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

// ---- SymptomObservation -----------------------------------------------------

func symptom(id, userID, ts string) *v1.SymptomObservation {
	return &v1.SymptomObservation{
		Name:      id,
		UserId:    userID,
		Timestamp: dateTime(ts),
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
		Severity:  v1.SymptomSeverity_SYMPTOM_SEVERITY_MODERATE,
	}
}

func TestSymptom_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create and get",
			setup: func(s *memory.Store) error {
				return s.SymptomObservations().Create(ctx, symptom("s1", "u1", "2026-01-01T08:00:00Z"))
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.SymptomObservations().GetByID(ctx, "s1")
				mustNoErr(t, err)
				if got.GetName() != "s1" {
					t.Fatalf("got %q", got.GetName())
				}
			},
		},
		{
			name: "get not found",
			setup: func(s *memory.Store) error {
				return nil
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.SymptomObservations().GetByID(ctx, "nope")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("want ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "duplicate ID rejected",
			setup: func(s *memory.Store) error {
				obs := symptom("s1", "u1", "2026-01-01T08:00:00Z")
				if err := s.SymptomObservations().Create(ctx, obs); err != nil {
					return err
				}
				return s.SymptomObservations().Create(ctx, obs)
			},
			wantErr: storage.ErrConflict,
		},
		{
			name: "delete",
			setup: func(s *memory.Store) error {
				if err := s.SymptomObservations().Create(ctx, symptom("s1", "u1", "2026-01-01T08:00:00Z")); err != nil {
					return err
				}
				return s.SymptomObservations().DeleteByID(ctx, "s1")
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.SymptomObservations().GetByID(ctx, "s1")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("expected not found after delete")
				}
			},
		},
		{
			name: "delete not found",
			setup: func(s *memory.Store) error {
				return s.SymptomObservations().DeleteByID(ctx, "nope")
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
	}
}

func TestSymptom_ListByUserAndDateRange(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.SymptomObservations().Create(ctx, symptom("s1", "u1", "2026-01-05T08:00:00Z")))
	mustNoErr(t, s.SymptomObservations().Create(ctx, symptom("s2", "u1", "2026-01-10T08:00:00Z")))
	mustNoErr(t, s.SymptomObservations().Create(ctx, symptom("s3", "u1", "2026-02-01T08:00:00Z")))
	mustNoErr(t, s.SymptomObservations().Create(ctx, symptom("s4", "u2", "2026-01-07T08:00:00Z")))

	page, err := s.SymptomObservations().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

func TestSymptom_Pagination(t *testing.T) {
	s := newStore()
	for i := 0; i < 5; i++ {
		ts := fmt.Sprintf("2026-01-%02dT08:00:00Z", i+1)
		mustNoErr(t, s.SymptomObservations().Create(ctx, symptom(fmt.Sprintf("s%d", i), "u1", ts)))
	}

	page1, _ := s.SymptomObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2})
	if len(page1.Items) != 2 || page1.NextPageToken == "" {
		t.Fatalf("page1: got %d items, token=%q", len(page1.Items), page1.NextPageToken)
	}
	page2, _ := s.SymptomObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page1.NextPageToken})
	if len(page2.Items) != 2 {
		t.Fatalf("page2: want 2, got %d", len(page2.Items))
	}
	page3, _ := s.SymptomObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page2.NextPageToken})
	if len(page3.Items) != 1 || page3.NextPageToken != "" {
		t.Fatalf("page3: got %d items, token=%q", len(page3.Items), page3.NextPageToken)
	}
}

// ---- MoodObservation --------------------------------------------------------

func mood(id, userID, ts string) *v1.MoodObservation {
	return &v1.MoodObservation{
		Name:      id,
		UserId:    userID,
		Timestamp: dateTime(ts),
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
		Intensity: v1.MoodIntensity_MOOD_INTENSITY_MEDIUM,
	}
}

func TestMood_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create and get",
			setup: func(s *memory.Store) error {
				return s.MoodObservations().Create(ctx, mood("mo1", "u1", "2026-01-01T08:00:00Z"))
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.MoodObservations().GetByID(ctx, "mo1")
				mustNoErr(t, err)
				if got.GetName() != "mo1" {
					t.Fatalf("got %q", got.GetName())
				}
			},
		},
		{
			name: "get not found",
			setup: func(s *memory.Store) error {
				return nil
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.MoodObservations().GetByID(ctx, "nope")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("want ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "duplicate ID rejected",
			setup: func(s *memory.Store) error {
				obs := mood("mo1", "u1", "2026-01-01T08:00:00Z")
				if err := s.MoodObservations().Create(ctx, obs); err != nil {
					return err
				}
				return s.MoodObservations().Create(ctx, obs)
			},
			wantErr: storage.ErrConflict,
		},
		{
			name: "delete",
			setup: func(s *memory.Store) error {
				if err := s.MoodObservations().Create(ctx, mood("mo1", "u1", "2026-01-01T08:00:00Z")); err != nil {
					return err
				}
				return s.MoodObservations().DeleteByID(ctx, "mo1")
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.MoodObservations().GetByID(ctx, "mo1")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("expected not found after delete")
				}
			},
		},
		{
			name: "delete not found",
			setup: func(s *memory.Store) error {
				return s.MoodObservations().DeleteByID(ctx, "nope")
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
	}
}

func TestMood_ListByUserAndDateRange(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.MoodObservations().Create(ctx, mood("mo1", "u1", "2026-01-05T08:00:00Z")))
	mustNoErr(t, s.MoodObservations().Create(ctx, mood("mo2", "u1", "2026-01-10T08:00:00Z")))
	mustNoErr(t, s.MoodObservations().Create(ctx, mood("mo3", "u1", "2026-02-01T08:00:00Z")))
	mustNoErr(t, s.MoodObservations().Create(ctx, mood("mo4", "u2", "2026-01-07T08:00:00Z")))

	page, err := s.MoodObservations().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
	}
}

func TestMood_Pagination(t *testing.T) {
	s := newStore()
	for i := 0; i < 5; i++ {
		ts := fmt.Sprintf("2026-01-%02dT08:00:00Z", i+1)
		mustNoErr(t, s.MoodObservations().Create(ctx, mood(fmt.Sprintf("mo%d", i), "u1", ts)))
	}

	page1, _ := s.MoodObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2})
	if len(page1.Items) != 2 || page1.NextPageToken == "" {
		t.Fatalf("page1: got %d items, token=%q", len(page1.Items), page1.NextPageToken)
	}
	page2, _ := s.MoodObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page1.NextPageToken})
	if len(page2.Items) != 2 {
		t.Fatalf("page2: want 2, got %d", len(page2.Items))
	}
	page3, _ := s.MoodObservations().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page2.NextPageToken})
	if len(page3.Items) != 1 || page3.NextPageToken != "" {
		t.Fatalf("page3: got %d items, token=%q", len(page3.Items), page3.NextPageToken)
	}
}

// ---- MedicationEvent --------------------------------------------------------

func medEvent(id, userID, medID, ts string) *v1.MedicationEvent {
	return &v1.MedicationEvent{
		Name:         id,
		UserId:       userID,
		MedicationId: medID,
		Timestamp:    dateTime(ts),
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
}

func TestMedicationEvent_CRUD(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create and get",
			setup: func(s *memory.Store) error {
				return s.MedicationEvents().Create(ctx, medEvent("e1", "u1", "m1", "2026-01-01T08:00:00Z"))
			},
			verify: func(t *testing.T, s *memory.Store) {
				got, err := s.MedicationEvents().GetByID(ctx, "e1")
				mustNoErr(t, err)
				if got.GetName() != "e1" {
					t.Fatalf("got %q", got.GetName())
				}
			},
		},
		{
			name: "get not found",
			setup: func(s *memory.Store) error {
				return nil
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.MedicationEvents().GetByID(ctx, "nope")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatalf("want ErrNotFound, got %v", err)
				}
			},
		},
		{
			name: "duplicate ID rejected",
			setup: func(s *memory.Store) error {
				ev := medEvent("e1", "u1", "m1", "2026-01-01T08:00:00Z")
				if err := s.MedicationEvents().Create(ctx, ev); err != nil {
					return err
				}
				return s.MedicationEvents().Create(ctx, ev)
			},
			wantErr: storage.ErrConflict,
		},
		{
			name: "delete",
			setup: func(s *memory.Store) error {
				if err := s.MedicationEvents().Create(ctx, medEvent("e1", "u1", "m1", "2026-01-01T08:00:00Z")); err != nil {
					return err
				}
				return s.MedicationEvents().DeleteByID(ctx, "e1")
			},
			verify: func(t *testing.T, s *memory.Store) {
				_, err := s.MedicationEvents().GetByID(ctx, "e1")
				if !errors.Is(err, storage.ErrNotFound) {
					t.Fatal("expected not found after delete")
				}
			},
		},
		{
			name: "delete not found",
			setup: func(s *memory.Store) error {
				return s.MedicationEvents().DeleteByID(ctx, "nope")
			},
			wantErr: storage.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
	}
}

func TestMedicationEvent_ListByUserAndDateRange(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e1", "u1", "m1", "2026-01-05T08:00:00Z")))
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e2", "u1", "m1", "2026-01-10T08:00:00Z")))
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e3", "u1", "m1", "2026-02-01T08:00:00Z")))
	mustNoErr(t, s.MedicationEvents().Create(ctx, medEvent("e4", "u2", "m2", "2026-01-07T08:00:00Z")))

	page, err := s.MedicationEvents().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2, got %d", len(page.Items))
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

// ---- PhaseEstimate ----------------------------------------------------------

func phaseEstimate(id, userID, date, cycleID string) *v1.PhaseEstimate {
	return &v1.PhaseEstimate{
		Name:       id,
		UserId:     userID,
		Date:       localDate(date),
		Phase:      v1.CyclePhase_CYCLE_PHASE_FOLLICULAR,
		Confidence: v1.ConfidenceLevel_CONFIDENCE_LEVEL_MEDIUM,
		BasedOnRecordRefs: []*v1.RecordRef{
			{Name: cycleID},
		},
	}
}

func TestPhaseEstimate_CreateListDuplicate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*memory.Store) error
		verify  func(*testing.T, *memory.Store)
		wantErr error
	}{
		{
			name: "create and list",
			setup: func(s *memory.Store) error {
				if err := s.PhaseEstimates().Create(ctx, phaseEstimate("pe1", "u1", "2026-01-05", "cy1")); err != nil {
					return err
				}
				if err := s.PhaseEstimates().Create(ctx, phaseEstimate("pe2", "u1", "2026-01-10", "cy1")); err != nil {
					return err
				}
				if err := s.PhaseEstimates().Create(ctx, phaseEstimate("pe3", "u1", "2026-02-01", "cy2")); err != nil {
					return err
				}
				if err := s.PhaseEstimates().Create(ctx, phaseEstimate("pe4", "u2", "2026-01-07", "cy3")); err != nil {
					return err
				}
				return nil
			},
			verify: func(t *testing.T, s *memory.Store) {
				page, err := s.PhaseEstimates().ListByUserAndDateRange(ctx, "u1", "2026-01-01", "2026-01-31", storage.PageRequest{})
				mustNoErr(t, err)
				if len(page.Items) != 2 {
					t.Fatalf("want 2, got %d", len(page.Items))
				}
			},
		},
		{
			name: "duplicate ID rejected",
			setup: func(s *memory.Store) error {
				if err := s.PhaseEstimates().Create(ctx, phaseEstimate("pe1", "u1", "2026-01-05", "cy1")); err != nil {
					return err
				}
				return s.PhaseEstimates().Create(ctx, phaseEstimate("pe1", "u1", "2026-01-05", "cy1"))
			},
			wantErr: storage.ErrConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStore()
			err := tt.setup(s)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("want %v, got %v", tt.wantErr, err)
				}
				return
			}
			mustNoErr(t, err)
			if tt.verify != nil {
				tt.verify(t, s)
			}
		})
	}
}

func TestPhaseEstimate_DeleteByCycleID(t *testing.T) {
	s := newStore()
	mustNoErr(t, s.PhaseEstimates().Create(ctx, phaseEstimate("pe1", "u1", "2026-01-05", "cy1")))
	mustNoErr(t, s.PhaseEstimates().Create(ctx, phaseEstimate("pe2", "u1", "2026-01-06", "cy1")))
	mustNoErr(t, s.PhaseEstimates().Create(ctx, phaseEstimate("pe3", "u1", "2026-02-01", "cy2")))

	mustNoErr(t, s.PhaseEstimates().DeleteByCycleID(ctx, "cy1"))

	page, _ := s.PhaseEstimates().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{})
	if len(page.Items) != 1 {
		t.Fatalf("want 1 remaining, got %d", len(page.Items))
	}
	if page.Items[0].GetName() != "pe3" {
		t.Fatalf("expected pe3 to survive, got %s", page.Items[0].GetName())
	}
}

func TestPhaseEstimate_ListEmpty(t *testing.T) {
	s := newStore()
	page, err := s.PhaseEstimates().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 0 {
		t.Fatal("expected empty")
	}
}

func TestPhaseEstimate_Pagination(t *testing.T) {
	s := newStore()
	for i := 0; i < 5; i++ {
		d := fmt.Sprintf("2026-01-%02d", i+1)
		mustNoErr(t, s.PhaseEstimates().Create(ctx, phaseEstimate(fmt.Sprintf("pe%d", i), "u1", d, "cy1")))
	}

	page1, _ := s.PhaseEstimates().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2})
	if len(page1.Items) != 2 || page1.NextPageToken == "" {
		t.Fatalf("page1: got %d items, token=%q", len(page1.Items), page1.NextPageToken)
	}
	page2, _ := s.PhaseEstimates().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page1.NextPageToken})
	if len(page2.Items) != 2 {
		t.Fatalf("page2: want 2, got %d", len(page2.Items))
	}
	page3, _ := s.PhaseEstimates().ListByUserAndDateRange(ctx, "u1", "", "", storage.PageRequest{PageSize: 2, PageToken: page2.NextPageToken})
	if len(page3.Items) != 1 || page3.NextPageToken != "" {
		t.Fatalf("page3: got %d items, token=%q", len(page3.Items), page3.NextPageToken)
	}
}

// ---- Prediction -------------------------------------------------------------

func prediction(id, userID, start string) *v1.Prediction {
	return &v1.Prediction{
		Name:               id,
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
		Name:       id,
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
