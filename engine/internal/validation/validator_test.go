package validation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/memory"
	"github.com/2ajoyce/openmenses/engine/internal/validation"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

var ctx = context.Background()

func newValidator(t *testing.T) *validation.Validator {
	t.Helper()
	val, err := validation.New(memory.New())
	if err != nil {
		t.Fatal(err)
	}
	return val
}

func newValidatorWithStore(t *testing.T, store storage.Repository) *validation.Validator {
	t.Helper()
	val, err := validation.New(store)
	if err != nil {
		t.Fatal(err)
	}
	return val
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// assertViolation verifies err is a *validation.Error with a violation for wantField.
func assertViolation(t *testing.T, err error, wantField string) {
	t.Helper()
	var valErr *validation.Error
	if !errors.As(err, &valErr) {
		t.Fatalf("want *validation.Error, got %T: %v", err, err)
	}
	for _, v := range valErr.Violations {
		if v.Field == wantField {
			return
		}
	}
	t.Fatalf("no violation for field %q; got violations: %+v", wantField, valErr.Violations)
}

// fixedNow returns a func() time.Time that always returns 2026-03-09T12:00:00Z.
func fixedNow() func() time.Time {
	fixed := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return fixed }
}

// ---- BleedingObservation -------------------------------------------------- //

func TestBleeding_ValidPass(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.BleedingObservation{
		Name:      "b1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
	mustNoErr(t, val.ValidateBleedingObservation(ctx, obs))
}

func TestBleeding_MissingFlow(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.BleedingObservation{
		Name:      "b1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		// Flow is UNSPECIFIED (0): proto says not_in:[0], so schema rejects it.
	}
	if err := val.ValidateBleedingObservation(ctx, obs); err == nil {
		t.Fatal("expected validation error for unspecified flow")
	}
}

func TestBleeding_MissingUserID(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.BleedingObservation{
		Name:      "b1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
	if err := val.ValidateBleedingObservation(ctx, obs); err == nil {
		t.Fatal("expected validation error for missing user_id")
	}
}

func TestBleeding_FutureTimestamp(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.BleedingObservation{
		Name:      "b1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T14:00:00Z"}, // 2 h in future
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
	assertViolation(t, val.ValidateBleedingObservation(ctx, obs), "timestamp")
}

func TestBleeding_TimestampWithinTolerance(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.BleedingObservation{
		Name:      "b1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T12:00:30Z"}, // 30 s in future
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
	mustNoErr(t, val.ValidateBleedingObservation(ctx, obs))
}

// ---- SymptomObservation --------------------------------------------------- //

func TestSymptom_ValidPass(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.SymptomObservation{
		Name:      "s1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
	}
	mustNoErr(t, val.ValidateSymptomObservation(ctx, obs))
}

func TestSymptom_FutureTimestamp(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.SymptomObservation{
		Name:      "s1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-10T00:00:00Z"}, // next day
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
	}
	assertViolation(t, val.ValidateSymptomObservation(ctx, obs), "timestamp")
}

// ---- MoodObservation ------------------------------------------------------ //

func TestMood_ValidPass(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.MoodObservation{
		Name:      "m1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
	}
	mustNoErr(t, val.ValidateMoodObservation(ctx, obs))
}

func TestMood_FutureTimestamp(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	obs := &v1.MoodObservation{
		Name:      "m1",
		UserId:    "u1",
		Timestamp: &v1.DateTime{Value: "2026-03-09T14:00:00Z"},
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
	}
	assertViolation(t, val.ValidateMoodObservation(ctx, obs), "timestamp")
}

// ---- Medication ----------------------------------------------------------- //

func TestMedication_ValidPass(t *testing.T) {
	val := newValidator(t)
	med := &v1.Medication{
		Name:        "med1",
		UserId:      "u1",
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      true,
	}
	mustNoErr(t, val.ValidateMedication(ctx, med))
}

func TestMedication_MissingName(t *testing.T) {
	val := newValidator(t)
	med := &v1.Medication{
		Name:     "med1",
		UserId:   "u1",
		Category: v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
	}
	if err := val.ValidateMedication(ctx, med); err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

// ---- MedicationEvent ------------------------------------------------------ //

func TestMedEvent_ValidPass(t *testing.T) {
	store := memory.New()
	mustNoErr(t, store.Medications().Create(ctx, &v1.Medication{
		Name:        "med1",
		UserId:      "u1",
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      true,
	}))
	val := newValidatorWithStore(t, store)
	val.Now = fixedNow()
	event := &v1.MedicationEvent{
		Name:         "e1",
		UserId:       "u1",
		MedicationId: "med1",
		Timestamp:    &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
	mustNoErr(t, val.ValidateMedicationEvent(ctx, event))
}

func TestMedEvent_MissingMedication(t *testing.T) {
	val := newValidator(t)
	val.Now = fixedNow()
	event := &v1.MedicationEvent{
		Name:         "e1",
		UserId:       "u1",
		MedicationId: "nonexistent",
		Timestamp:    &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
	assertViolation(t, val.ValidateMedicationEvent(ctx, event), "medication_id")
}

func TestMedEvent_InactiveMedication(t *testing.T) {
	store := memory.New()
	mustNoErr(t, store.Medications().Create(ctx, &v1.Medication{
		Name:        "med1",
		UserId:      "u1",
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      false,
	}))
	val := newValidatorWithStore(t, store)
	val.Now = fixedNow()
	event := &v1.MedicationEvent{
		Name:         "e1",
		UserId:       "u1",
		MedicationId: "med1",
		Timestamp:    &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
	assertViolation(t, val.ValidateMedicationEvent(ctx, event), "medication_id")
}

func TestMedEvent_MedicationWrongUser(t *testing.T) {
	store := memory.New()
	mustNoErr(t, store.Medications().Create(ctx, &v1.Medication{
		Name:        "med1",
		UserId:      "u2", // belongs to u2
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      true,
	}))
	val := newValidatorWithStore(t, store)
	val.Now = fixedNow()
	event := &v1.MedicationEvent{
		Name:         "e1",
		UserId:       "u1", // u1 trying to reference u2's medication
		MedicationId: "med1",
		Timestamp:    &v1.DateTime{Value: "2026-03-09T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
	assertViolation(t, val.ValidateMedicationEvent(ctx, event), "medication_id")
}

func TestMedEvent_FutureAndMissingMed_BothReported(t *testing.T) {
	val := newValidator(t) // empty store, so medication won't be found
	val.Now = fixedNow()
	event := &v1.MedicationEvent{
		Name:         "e1",
		UserId:       "u1",
		MedicationId: "nonexistent",
		Timestamp:    &v1.DateTime{Value: "2026-03-09T15:00:00Z"}, // 3 h in future
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
	err := val.ValidateMedicationEvent(ctx, event)
	var valErr *validation.Error
	if !errors.As(err, &valErr) {
		t.Fatalf("want *validation.Error, got %T: %v", err, err)
	}
	if len(valErr.Violations) < 2 {
		t.Fatalf("expected at least 2 violations; got %d: %+v", len(valErr.Violations), valErr.Violations)
	}
}

// ---- Cycle ---------------------------------------------------------------- //

func TestCycle_ValidPass(t *testing.T) {
	val := newValidator(t)
	c := &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	mustNoErr(t, val.ValidateCycle(ctx, c))
}

func TestCycle_OpenEnded_ValidPass(t *testing.T) {
	val := newValidator(t)
	c := &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		// EndDate deliberately omitted
		Source: v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	mustNoErr(t, val.ValidateCycle(ctx, c))
}

func TestCycle_EndBeforeStart(t *testing.T) {
	val := newValidator(t)
	c := &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-28"},
		EndDate:   &v1.LocalDate{Value: "2026-01-01"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	assertViolation(t, val.ValidateCycle(ctx, c), "end_date")
}

func TestCycle_OverlapRejected(t *testing.T) {
	store := memory.New()
	mustNoErr(t, store.Cycles().Create(ctx, &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}))
	val := newValidatorWithStore(t, store)
	overlap := &v1.Cycle{
		Name:      "cy2",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-15"},
		EndDate:   &v1.LocalDate{Value: "2026-02-10"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	assertViolation(t, val.ValidateCycle(ctx, overlap), "cycle")
}

func TestCycle_NoOverlapAdjacentCycles(t *testing.T) {
	store := memory.New()
	mustNoErr(t, store.Cycles().Create(ctx, &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}))
	val := newValidatorWithStore(t, store)
	// Starts the day after the existing cycle ends — no overlap.
	next := &v1.Cycle{
		Name:      "cy2",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-29"},
		EndDate:   &v1.LocalDate{Value: "2026-02-25"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	mustNoErr(t, val.ValidateCycle(ctx, next))
}

func TestCycle_NoOverlapDifferentUser(t *testing.T) {
	store := memory.New()
	mustNoErr(t, store.Cycles().Create(ctx, &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}))
	val := newValidatorWithStore(t, store)
	// Same date range but different user — must pass.
	c := &v1.Cycle{
		Name:      "cy2",
		UserId:    "u2",
		StartDate: &v1.LocalDate{Value: "2026-01-15"},
		EndDate:   &v1.LocalDate{Value: "2026-02-10"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	mustNoErr(t, val.ValidateCycle(ctx, c))
}

func TestCycle_UpdateIgnoresOwnId(t *testing.T) {
	store := memory.New()
	existing := &v1.Cycle{
		Name:      "cy1",
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-28"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	mustNoErr(t, store.Cycles().Create(ctx, existing))
	val := newValidatorWithStore(t, store)
	// Updating cy1 with a different end_date — must not flag overlap with itself.
	updated := &v1.Cycle{
		Name:      "cy1", // same name
		UserId:    "u1",
		StartDate: &v1.LocalDate{Value: "2026-01-01"},
		EndDate:   &v1.LocalDate{Value: "2026-01-30"},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	mustNoErr(t, val.ValidateCycle(ctx, updated))
}

// ---- UserProfile ---------------------------------------------------------- //

func validProfile() *v1.UserProfile {
	return &v1.UserProfile{
		Name:             "u1",
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_BLEEDING},
	}
}

func TestUserProfile_ValidPass(t *testing.T) {
	val := newValidator(t)
	mustNoErr(t, val.ValidateUserProfile(ctx, validProfile()))
}

func TestUserProfile_MissingTrackingFocus(t *testing.T) {
	val := newValidator(t)
	p := validProfile()
	p.TrackingFocus = nil // remove required field
	if err := val.ValidateUserProfile(ctx, p); err == nil {
		t.Fatal("expected validation error for empty tracking_focus")
	}
}

func TestUserProfile_DuplicateTrackingFocus(t *testing.T) {
	val := newValidator(t)
	p := validProfile()
	p.TrackingFocus = []v1.TrackingFocus{
		v1.TrackingFocus_TRACKING_FOCUS_BLEEDING,
		v1.TrackingFocus_TRACKING_FOCUS_BLEEDING, // duplicate
	}
	if err := val.ValidateUserProfile(ctx, p); err == nil {
		t.Fatal("expected validation error for duplicate tracking_focus")
	}
}

// ---- IsProfileComplete ---------------------------------------------------- //

func TestIsProfileComplete_AllSet(t *testing.T) {
	if !validation.IsProfileComplete(validProfile()) {
		t.Fatal("expected complete profile to return true")
	}
}

func TestIsProfileComplete_MissingBiologicalCycle(t *testing.T) {
	p := validProfile()
	p.BiologicalCycle = v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_UNSPECIFIED
	if validation.IsProfileComplete(p) {
		t.Fatal("expected incomplete profile to return false")
	}
}

func TestIsProfileComplete_MissingCycleRegularity(t *testing.T) {
	p := validProfile()
	p.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_UNSPECIFIED
	if validation.IsProfileComplete(p) {
		t.Fatal("expected incomplete profile to return false")
	}
}

// ---- Malformed timestamp strings ------------------------------------------ //

// malformedTimestamps contains strings that are 20–64 characters but do not
// conform to RFC3339 format. All should be rejected after 16.1 and 16.2 fixes.
var malformedTimestamps = []string{
	"not-a-valid-timestamp-xx",        // garbage, 24 chars
	"2026/03/09 10:00:00 +00:00 UTC!", // slashes/space instead of dashes/T
	"20260309T100000Z000000000000000", // no dashes or colons, 31 chars
}

func TestBleeding_MalformedTimestamp(t *testing.T) {
	for _, ts := range malformedTimestamps {
		t.Run(ts, func(t *testing.T) {
			val := newValidator(t)
			val.Now = fixedNow()
			obs := &v1.BleedingObservation{
				Name:      "b1",
				UserId:    "u1",
				Timestamp: &v1.DateTime{Value: ts},
				Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
			}
			if err := val.ValidateBleedingObservation(ctx, obs); err == nil {
				t.Fatalf("expected validation error for malformed timestamp %q", ts)
			}
		})
	}
}

func TestSymptom_MalformedTimestamp(t *testing.T) {
	for _, ts := range malformedTimestamps {
		t.Run(ts, func(t *testing.T) {
			val := newValidator(t)
			val.Now = fixedNow()
			obs := &v1.SymptomObservation{
				Name:      "s1",
				UserId:    "u1",
				Timestamp: &v1.DateTime{Value: ts},
				Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
			}
			if err := val.ValidateSymptomObservation(ctx, obs); err == nil {
				t.Fatalf("expected validation error for malformed timestamp %q", ts)
			}
		})
	}
}

func TestMood_MalformedTimestamp(t *testing.T) {
	for _, ts := range malformedTimestamps {
		t.Run(ts, func(t *testing.T) {
			val := newValidator(t)
			val.Now = fixedNow()
			obs := &v1.MoodObservation{
				Name:      "m1",
				UserId:    "u1",
				Timestamp: &v1.DateTime{Value: ts},
				Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
			}
			if err := val.ValidateMoodObservation(ctx, obs); err == nil {
				t.Fatalf("expected validation error for malformed timestamp %q", ts)
			}
		})
	}
}

func TestMedEvent_MalformedTimestamp(t *testing.T) {
	store := memory.New()
	if err := store.Medications().Create(ctx, &v1.Medication{
		Name:        "med1",
		UserId:      "u1",
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      true,
	}); err != nil {
		panic(err)
	}
	for _, ts := range malformedTimestamps {
		t.Run(ts, func(t *testing.T) {
			val := newValidatorWithStore(t, store)
			val.Now = fixedNow()
			event := &v1.MedicationEvent{
				Name:         "e1",
				UserId:       "u1",
				MedicationId: "med1",
				Timestamp:    &v1.DateTime{Value: ts},
				Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
			}
			if err := val.ValidateMedicationEvent(ctx, event); err == nil {
				t.Fatalf("expected validation error for malformed timestamp %q", ts)
			}
		})
	}
}
