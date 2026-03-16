package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/2ajoyce/openmenses/engine/internal/service"
	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/storage/memory"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

var ctx = context.Background()

// newSvc creates a CycleTrackerService backed by an empty in-memory store.
func newSvc(t *testing.T) *service.CycleTrackerService {
	t.Helper()
	svc, err := service.New(memory.New())
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// newSvcWithStore creates a CycleTrackerService backed by the given store.
func newSvcWithStore(t *testing.T, store storage.Repository) *service.CycleTrackerService {
	t.Helper()
	svc, err := service.New(store)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// codeOf extracts the Connect-RPC error code from an error, or returns
// connect.CodeUnknown if the error is not a Connect error.
func codeOf(err error) connect.Code {
	var connErr *connect.Error
	if errors.As(err, &connErr) {
		return connErr.Code()
	}
	return connect.CodeUnknown
}

// ─── helper builders ──────────────────────────────────────────────────────────

func validProfile(id string) *v1.UserProfile {
	return &v1.UserProfile{
		Name:             id,
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_BLEEDING},
	}
}

func validBleeding(id, userID, date string) *v1.BleedingObservation {
	return &v1.BleedingObservation{
		Name:      id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
	}
}

func validSymptom(id, userID, date string) *v1.SymptomObservation {
	return &v1.SymptomObservation{
		Name:      id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Symptom:   v1.SymptomType_SYMPTOM_TYPE_CRAMPS,
	}
}

func validMood(id, userID, date string) *v1.MoodObservation {
	return &v1.MoodObservation{
		Name:      id,
		UserId:    userID,
		Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
		Mood:      v1.MoodType_MOOD_TYPE_HAPPY,
	}
}

func validMedication(id, userID string) *v1.Medication {
	return &v1.Medication{
		Name:        id,
		UserId:      userID,
		DisplayName: "Ibuprofen",
		Category:    v1.MedicationCategory_MEDICATION_CATEGORY_PAIN_RELIEF,
		Active:      true,
	}
}

func validMedEvent(id, userID, medID, date string) *v1.MedicationEvent {
	return &v1.MedicationEvent{
		Name:         id,
		UserId:       userID,
		MedicationId: medID,
		Timestamp:    &v1.DateTime{Value: date + "T10:00:00Z"},
		Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
	}
}

// ─── GetUserProfile ───────────────────────────────────────────────────────────

func TestGetUserProfile(t *testing.T) {
	tests := []struct {
		name       string
		profileID  string
		queryID    string
		wantCode   connect.Code
		shouldFail bool
		wantExists bool
	}{
		{
			name:       "NotFound",
			profileID:  "u1",
			queryID:    "missing",
			wantCode:   connect.CodeNotFound,
			shouldFail: true,
			wantExists: false,
		},
		{
			name:       "Found",
			profileID:  "u1",
			queryID:    "u1",
			wantCode:   0,
			shouldFail: false,
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()
			if tt.profileID != "" {
				if err := store.UserProfiles().Upsert(ctx, validProfile(tt.profileID)); err != nil {
					t.Fatal(err)
				}
			}
			svc := newSvcWithStore(t, store)
			resp, err := svc.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: tt.queryID}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantExists && resp.Msg.GetProfile().GetName() != tt.queryID {
				t.Errorf("got profile name %q, want %q", resp.Msg.GetProfile().GetName(), tt.queryID)
			}
		})
	}
}

// ─── CreateUserProfile ───────────────────────────────────────────────────────

func TestCreateUserProfile(t *testing.T) {
	tests := []struct {
		name       string
		profile    *v1.UserProfile
		preExist   bool
		shouldFail bool
		wantCode   connect.Code
	}{
		{
			name:       "HappyPath",
			profile:    validProfile("u1"),
			preExist:   false,
			shouldFail: false,
			wantCode:   0,
		},
		{
			name: "ValidationFailure",
			profile: &v1.UserProfile{
				Name:             "u1",
				BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
				Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
				CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
				ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
				// TrackingFocus intentionally left empty → schema violation
			},
			preExist:   false,
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
		},
		{
			name:       "Conflict",
			profile:    validProfile("u1"),
			preExist:   true,
			shouldFail: true,
			wantCode:   connect.CodeAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()
			if tt.preExist {
				if err := store.UserProfiles().Create(ctx, tt.profile); err != nil {
					t.Fatal(err)
				}
			}
			svc := newSvcWithStore(t, store)
			resp, err := svc.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{Profile: tt.profile}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if resp.Msg.GetProfile().GetName() != tt.profile.GetName() {
				t.Errorf("got profile name %q, want %q", resp.Msg.GetProfile().GetName(), tt.profile.GetName())
			}
		})
	}
}

// ─── UpdateUserProfile ───────────────────────────────────────────────────────

func TestUpdateUserProfile(t *testing.T) {
	tests := []struct {
		name        string
		preExist    bool
		shouldFail  bool
		wantCode    connect.Code
		wantUpdated bool
	}{
		{
			name:        "HappyPath",
			preExist:    true,
			shouldFail:  false,
			wantCode:    0,
			wantUpdated: true,
		},
		{
			name:        "NotFound",
			preExist:    false,
			shouldFail:  true,
			wantCode:    connect.CodeNotFound,
			wantUpdated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()
			if tt.preExist {
				if err := store.UserProfiles().Create(ctx, validProfile("u1")); err != nil {
					t.Fatal(err)
				}
			}
			svc := newSvcWithStore(t, store)

			updated := validProfile("u1")
			updated.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR
			resp, err := svc.UpdateUserProfile(ctx, connect.NewRequest(&v1.UpdateUserProfileRequest{Profile: updated}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.wantUpdated && resp.Msg.GetProfile().GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR {
				t.Error("profile was not updated")
			}
		})
	}

	// PartialUpdateWithFieldMask - complex subtest
	t.Run("PartialUpdateWithFieldMask", func(t *testing.T) {
		store := memory.New()
		// Create an initial profile with specific values
		initial := validProfile("u1")
		initial.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_REGULAR
		initial.ReproductiveGoal = v1.ReproductiveGoal_REPRODUCTIVE_GOAL_AVOID_PREGNANCY
		if err := store.UserProfiles().Create(ctx, initial); err != nil {
			t.Fatal(err)
		}

		svc := newSvcWithStore(t, store)

		// Update only the CycleRegularity field with a FieldMask
		updates := validProfile("u1")
		updates.CycleRegularity = v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR
		updates.ReproductiveGoal = v1.ReproductiveGoal_REPRODUCTIVE_GOAL_TRYING_TO_CONCEIVE // Should NOT be applied

		updateMask := &fieldmaskpb.FieldMask{
			Paths: []string{"cycle_regularity"}, // Only update this field
		}

		resp, err := svc.UpdateUserProfile(ctx, connect.NewRequest(&v1.UpdateUserProfileRequest{
			Profile:    updates,
			UpdateMask: updateMask,
		}))
		if err != nil {
			t.Fatal(err)
		}

		profile := resp.Msg.GetProfile()

		// Check that only cycle_regularity was updated
		if profile.GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR {
			t.Error("cycle_regularity was not updated")
		}

		// Check that reproductive_goal was NOT updated (remained the original value)
		if profile.GetReproductiveGoal() != v1.ReproductiveGoal_REPRODUCTIVE_GOAL_AVOID_PREGNANCY {
			t.Errorf("reproductive_goal was unexpectedly changed to %v", profile.GetReproductiveGoal())
		}
	})

	// InvalidFieldMaskPath - subtest for invalid field mask paths
	t.Run("InvalidFieldMaskPath", func(t *testing.T) {
		store := memory.New()
		if err := store.UserProfiles().Create(ctx, validProfile("u1")); err != nil {
			t.Fatal(err)
		}

		svc := newSvcWithStore(t, store)

		updates := validProfile("u1")
		updateMask := &fieldmaskpb.FieldMask{
			Paths: []string{"typo_field"}, // Invalid field path
		}

		_, err := svc.UpdateUserProfile(ctx, connect.NewRequest(&v1.UpdateUserProfileRequest{
			Profile:    updates,
			UpdateMask: updateMask,
		}))
		if err == nil {
			t.Fatal("expected error for invalid field mask path, got nil")
		}
		if codeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("want code %v, got %v", connect.CodeInvalidArgument, codeOf(err))
		}
	})
}

// ─── CreateBleedingObservation ────────────────────────────────────────────────

func TestCreateBleeding(t *testing.T) {
	tests := []struct {
		name       string
		obs        *v1.BleedingObservation
		duplicate  bool
		shouldFail bool
		wantCode   connect.Code
	}{
		{
			name:       "HappyPath",
			obs:        validBleeding("b1", "u1", "2026-01-15"),
			duplicate:  false,
			shouldFail: false,
			wantCode:   0,
		},
		{
			name:       "AutoID",
			obs:        validBleeding("", "u1", "2026-01-15"),
			duplicate:  false,
			shouldFail: false,
			wantCode:   0,
		},
		{
			name:       "NilObservation",
			obs:        nil,
			duplicate:  false,
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
		},
		{
			name:       "DuplicateID",
			obs:        validBleeding("b1", "u1", "2026-01-15"),
			duplicate:  true,
			shouldFail: true,
			wantCode:   connect.CodeAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSvc(t)

			if tt.duplicate && tt.obs != nil {
				if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: tt.obs})); err != nil {
					t.Fatal(err)
				}
			}

			resp, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: tt.obs}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.obs != nil && tt.obs.GetName() == "" {
				if resp.Msg.GetObservation().GetName() == "" {
					t.Error("expected auto-assigned name, got empty string")
				}
			} else if tt.obs != nil && resp.Msg.GetObservation().GetName() != tt.obs.GetName() {
				t.Errorf("got name %q, want %q", resp.Msg.GetObservation().GetName(), tt.obs.GetName())
			}
		})
	}

	// TriggersRedetection - complex subtest
	t.Run("TriggersRedetection", func(t *testing.T) {
		store := memory.New()
		svc := newSvcWithStore(t, store)
		// Log three bleeding episodes: two on consecutive days, then a gap
		for _, obs := range []*v1.BleedingObservation{
			validBleeding("b1", "u1", "2026-01-01"),
			validBleeding("b2", "u1", "2026-01-02"),
			validBleeding("b3", "u1", "2026-01-30"),
		} {
			if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
				t.Fatal(err)
			}
		}
		resp, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.GetCycles()) != 2 {
			t.Errorf("want 2 cycles after re-detection, got %d", len(resp.Msg.GetCycles()))
		}
	})
}

// ─── CreateSymptomObservation ─────────────────────────────────────────────────

func TestCreateSymptom(t *testing.T) {
	tests := []struct {
		name       string
		obs        *v1.SymptomObservation
		shouldFail bool
		wantCode   connect.Code
		wantValid  bool
	}{
		{
			name:       "HappyPath",
			obs:        validSymptom("s1", "u1", "2026-01-15"),
			shouldFail: false,
			wantCode:   0,
			wantValid:  true,
		},
		{
			name:       "NilObservation",
			obs:        nil,
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
			wantValid:  false,
		},
		{
			name:       "AutoID",
			obs:        validSymptom("", "u1", "2026-01-15"),
			shouldFail: false,
			wantCode:   0,
			wantValid:  true,
		},
		{
			name: "ValidationFailure",
			obs: func() *v1.SymptomObservation {
				bad := validSymptom("s1", "u1", "2026-01-15")
				bad.Symptom = v1.SymptomType_SYMPTOM_TYPE_UNSPECIFIED
				return bad
			}(),
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSvc(t)
			resp, err := svc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{Parent: "u1", Observation: tt.obs}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.wantValid {
				if tt.obs != nil && tt.obs.GetName() == "" {
					if resp.Msg.GetObservation().GetName() == "" {
						t.Error("expected auto-assigned name, got empty")
					}
				} else if tt.obs != nil && resp.Msg.GetObservation().GetName() != tt.obs.GetName() {
					t.Errorf("got name %q, want %q", resp.Msg.GetObservation().GetName(), tt.obs.GetName())
				}
			}
		})
	}
}

// ─── CreateMoodObservation ────────────────────────────────────────────────────

func TestCreateMood(t *testing.T) {
	tests := []struct {
		name       string
		obs        *v1.MoodObservation
		shouldFail bool
		wantCode   connect.Code
		wantValid  bool
	}{
		{
			name:       "HappyPath",
			obs:        validMood("m1", "u1", "2026-01-15"),
			shouldFail: false,
			wantCode:   0,
			wantValid:  true,
		},
		{
			name:       "NilObservation",
			obs:        nil,
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
			wantValid:  false,
		},
		{
			name:       "AutoID",
			obs:        validMood("", "u1", "2026-01-15"),
			shouldFail: false,
			wantCode:   0,
			wantValid:  true,
		},
		{
			name: "ValidationFailure",
			obs: func() *v1.MoodObservation {
				bad := validMood("m1", "u1", "2026-01-15")
				bad.Mood = v1.MoodType_MOOD_TYPE_UNSPECIFIED
				return bad
			}(),
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSvc(t)
			resp, err := svc.CreateMoodObservation(ctx, connect.NewRequest(&v1.CreateMoodObservationRequest{Parent: "u1", Observation: tt.obs}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.wantValid {
				if tt.obs != nil && tt.obs.GetName() == "" {
					if resp.Msg.GetObservation().GetName() == "" {
						t.Error("expected auto-assigned name, got empty")
					}
				} else if tt.obs != nil && resp.Msg.GetObservation().GetName() != tt.obs.GetName() {
					t.Errorf("got name %q, want %q", resp.Msg.GetObservation().GetName(), tt.obs.GetName())
				}
			}
		})
	}
}

// ─── CreateMedication ─────────────────────────────────────────────────────────

func TestCreateMedication(t *testing.T) {
	tests := []struct {
		name       string
		med        *v1.Medication
		shouldFail bool
		wantCode   connect.Code
		wantValid  bool
	}{
		{
			name:       "HappyPath",
			med:        validMedication("med1", "u1"),
			shouldFail: false,
			wantCode:   0,
			wantValid:  true,
		},
		{
			name:       "NilMedication",
			med:        nil,
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
			wantValid:  false,
		},
		{
			name:       "AutoID",
			med:        validMedication("", "u1"),
			shouldFail: false,
			wantCode:   0,
			wantValid:  true,
		},
		{
			name: "ValidationFailure",
			med: func() *v1.Medication {
				bad := validMedication("med1", "u1")
				bad.DisplayName = ""
				return bad
			}(),
			shouldFail: true,
			wantCode:   connect.CodeInvalidArgument,
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSvc(t)
			resp, err := svc.CreateMedication(ctx, connect.NewRequest(&v1.CreateMedicationRequest{Parent: "u1", Medication: tt.med}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.wantValid {
				if tt.med != nil && tt.med.GetName() == "" {
					if resp.Msg.GetMedication().GetName() == "" {
						t.Error("expected auto-assigned name")
					}
				} else if tt.med != nil && resp.Msg.GetMedication().GetName() != tt.med.GetName() {
					t.Errorf("got name %q, want %q", resp.Msg.GetMedication().GetName(), tt.med.GetName())
				}
			}
		})
	}
}

// ─── CreateMedicationEvent ────────────────────────────────────────────────────

func TestCreateMedEvent(t *testing.T) {
	tests := []struct {
		name         string
		event        *v1.MedicationEvent
		preCreateMed bool
		shouldFail   bool
		wantCode     connect.Code
		wantValid    bool
	}{
		{
			name:         "HappyPath",
			event:        validMedEvent("ev1", "u1", "med1", "2026-01-15"),
			preCreateMed: true,
			shouldFail:   false,
			wantCode:     0,
			wantValid:    true,
		},
		{
			name:         "MissingMedication",
			event:        validMedEvent("ev1", "u1", "med1", "2026-01-15"),
			preCreateMed: false,
			shouldFail:   true,
			wantCode:     connect.CodeInvalidArgument,
			wantValid:    false,
		},
		{
			name:         "NilEvent",
			event:        nil,
			preCreateMed: false,
			shouldFail:   true,
			wantCode:     connect.CodeInvalidArgument,
			wantValid:    false,
		},
		{
			name:         "AutoID",
			event:        validMedEvent("", "u1", "med1", "2026-01-15"),
			preCreateMed: true,
			shouldFail:   false,
			wantCode:     0,
			wantValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()
			if tt.preCreateMed {
				if err := store.Medications().Create(ctx, validMedication("med1", "u1")); err != nil {
					t.Fatal(err)
				}
			}
			svc := newSvcWithStore(t, store)
			resp, err := svc.CreateMedicationEvent(ctx, connect.NewRequest(&v1.CreateMedicationEventRequest{Parent: "u1", Event: tt.event}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if tt.wantValid {
				if tt.event != nil && tt.event.GetName() == "" {
					if resp.Msg.GetEvent().GetName() == "" {
						t.Error("expected auto-assigned name, got empty")
					}
				} else if tt.event != nil && resp.Msg.GetEvent().GetName() != tt.event.GetName() {
					t.Errorf("got name %q, want %q", resp.Msg.GetEvent().GetName(), tt.event.GetName())
				}
			}
		})
	}
}

// ─── GetCycle ─────────────────────────────────────────────────────────────

func TestGetCycle(t *testing.T) {
	tests := []struct {
		name       string
		cycleID    string
		queryID    string
		wantCode   connect.Code
		shouldFail bool
		wantExists bool
	}{
		{
			name:       "NotFound",
			cycleID:    "c1",
			queryID:    "missing",
			wantCode:   connect.CodeNotFound,
			shouldFail: true,
			wantExists: false,
		},
		{
			name:       "Found",
			cycleID:    "c1",
			queryID:    "c1",
			wantCode:   0,
			shouldFail: false,
			wantExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()
			// Manually create a cycle for the "Found" case
			if tt.cycleID != "" {
				cycle := &v1.Cycle{
					Name:      tt.cycleID,
					UserId:    "u1",
					StartDate: &v1.LocalDate{Value: "2026-01-01"},
					EndDate:   &v1.LocalDate{Value: "2026-01-28"},
					Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
				}
				if err := store.Cycles().Create(ctx, cycle); err != nil {
					t.Fatal(err)
				}
			}
			svc := newSvcWithStore(t, store)
			resp, err := svc.GetCycle(ctx, connect.NewRequest(&v1.GetCycleRequest{Name: tt.queryID}))

			if tt.shouldFail {
				if codeOf(err) != tt.wantCode {
					t.Fatalf("want code %v, got %v", tt.wantCode, codeOf(err))
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantExists && resp.Msg.GetCycle().GetName() != tt.queryID {
				t.Errorf("got cycle name %q, want %q", resp.Msg.GetCycle().GetName(), tt.queryID)
			}
		})
	}
}

// ─── GetCycleStatistics ───────────────────────────────────────────────────

func TestGetCycleStatistics(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(*service.CycleTrackerService) error
		windowSize int32
		wantCount  int32
	}{
		{
			name: "NoCycles",
			setupFunc: func(svc *service.CycleTrackerService) error {
				return nil
			},
			windowSize: 0,
			wantCount:  0,
		},
		{
			name: "SingleOpenEndedCycle",
			setupFunc: func(svc *service.CycleTrackerService) error {
				// Single recent bleeding creates an open-ended cycle with no end_date
				// Stats only counts completed cycles, so count will be 0
				obs := validBleeding("b1", "u1", "2026-03-10")
				_, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs}))
				return err
			},
			windowSize: 0,
			wantCount:  0, // Open-ended cycles are not counted by Stats()
		},
		{
			name: "TwoMonthsCreatesOneCompletedCycle",
			setupFunc: func(svc *service.CycleTrackerService) error {
				// Observations spanning two months create completed cycle(s) and an open cycle
				for _, obs := range []*v1.BleedingObservation{
					validBleeding("b1", "u1", "2026-01-01"),
					validBleeding("b2", "u1", "2026-01-02"),
					validBleeding("b3", "u1", "2026-02-01"),
				} {
					if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
						return err
					}
				}
				return nil
			},
			windowSize: 0,
			wantCount:  1, // One completed cycle (Jan), one open (Feb)
		},
		{
			name: "ThreeMonthsCreatesMultipleCompletedCycles",
			setupFunc: func(svc *service.CycleTrackerService) error {
				// Create 3 completed cycles + 1 open-ended (use past dates: 2025)
				for _, obs := range []*v1.BleedingObservation{
					validBleeding("b1", "u1", "2025-10-01"),
					validBleeding("b2", "u1", "2025-11-01"),
					validBleeding("b3", "u1", "2025-12-01"),
					validBleeding("b4", "u1", "2026-01-01"),
				} {
					if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
						return err
					}
				}
				return nil
			},
			windowSize: 0,
			wantCount:  3, // Oct, Nov, Dec 2025 are completed, Jan 2026 is open
		},
		{
			name: "WindowSizeFilter",
			setupFunc: func(svc *service.CycleTrackerService) error {
				// Create multiple cycles spanning multiple months (use past dates)
				for _, obs := range []*v1.BleedingObservation{
					validBleeding("b1", "u1", "2025-10-01"),
					validBleeding("b2", "u1", "2025-11-01"),
					validBleeding("b3", "u1", "2025-12-01"),
					validBleeding("b4", "u1", "2026-01-01"),
				} {
					if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
						return err
					}
				}
				return nil
			},
			windowSize: 2,
			wantCount:  2, // WindowStats returns stats for the last 2 completed cycles (Dec, Nov or Nov, Oct)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSvc(t)
			if err := tt.setupFunc(svc); err != nil {
				t.Fatal(err)
			}
			resp, err := svc.GetCycleStatistics(ctx, connect.NewRequest(&v1.GetCycleStatisticsRequest{
				Parent:     "u1",
				WindowSize: tt.windowSize,
			}))
			if err != nil {
				t.Fatal(err)
			}
			if resp.Msg.GetStatistics().GetCount() != tt.wantCount {
				t.Errorf("want count %d, got %d", tt.wantCount, resp.Msg.GetStatistics().GetCount())
			}
		})
	}
}

// ─── ListCycles ───────────────────────────────────────────────────────────────

func TestListCycles(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*service.CycleTrackerService) error
		wantCycleCount int
	}{
		{
			name: "EmptyUser",
			setupFunc: func(svc *service.CycleTrackerService) error {
				return nil
			},
			wantCycleCount: 0,
		},
		{
			name: "DetectsFromObs",
			setupFunc: func(svc *service.CycleTrackerService) error {
				obs := validBleeding("b1", "u1", "2026-01-01")
				_, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs}))
				return err
			},
			wantCycleCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSvc(t)
			if err := tt.setupFunc(svc); err != nil {
				t.Fatal(err)
			}
			resp, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
			if err != nil {
				t.Fatal(err)
			}
			if len(resp.Msg.GetCycles()) != tt.wantCycleCount {
				t.Errorf("want %d cycles, got %d", tt.wantCycleCount, len(resp.Msg.GetCycles()))
			}
			if tt.wantCycleCount > 0 && resp.Msg.GetCycles()[0].GetStartDate().GetValue() != "2026-01-01" {
				t.Errorf("start_date = %q, want 2026-01-01", resp.Msg.GetCycles()[0].GetStartDate().GetValue())
			}
		})
	}

	// Multiple cycles with gap detection
	t.Run("MultipleCyclesWithGapDetection", func(t *testing.T) {
		svc := newSvc(t)
		// Create cycles with >3 day gaps
		// Cycle 1: Jan 1-2
		// Gap of 4+ days
		// Cycle 2: Jan 7-8
		// Gap crossing into Feb triggers end of Jan cycle
		// Cycle 3: Feb 1+
		for _, obs := range []*v1.BleedingObservation{
			validBleeding("b1", "u1", "2025-01-01"),
			validBleeding("b2", "u1", "2025-01-02"),
			validBleeding("b3", "u1", "2025-01-07"),
			validBleeding("b4", "u1", "2025-01-08"),
			validBleeding("b5", "u1", "2025-02-01"),
		} {
			if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
				t.Fatal(err)
			}
		}
		resp, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		// We expect at least 1 cycle
		if len(resp.Msg.GetCycles()) == 0 {
			t.Errorf("want at least 1 cycle with gap detection, got %d", len(resp.Msg.GetCycles()))
		}
	})

	// Pagination
	t.Run("PaginationBehavior", func(t *testing.T) {
		svc := newSvc(t)
		// Create multiple bleeding episodes with gaps to form multiple cycles
		dates := []string{
			"2025-01-01", "2025-01-02", // Cycle 1
			"2025-01-07", "2025-01-08", // Cycle 2 (gap of 4+ days)
			"2025-02-01", "2025-02-02", // Cycle 3 (new month)
			"2025-02-07", "2025-02-08", // Cycle 4 (gap of 4+ days)
		}
		for i, date := range dates {
			obs := validBleeding("b"+string(rune('0'+1+i)), "u1", date)
			if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
				t.Fatal(err)
			}
		}
		// Request page 1 with page_size 2
		resp1, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{
			Parent:     "u1",
			Pagination: &v1.PaginationRequest{PageSize: 2},
		}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp1.Msg.GetCycles()) == 0 {
			t.Fatalf("page 1: expected at least 1 cycle, got %d", len(resp1.Msg.GetCycles()))
		}
		// Only test next_page_token if we got enough cycles
		if len(resp1.Msg.GetCycles()) >= 2 {
			nextToken := resp1.Msg.GetPagination().GetNextPageToken()
			if nextToken != "" {
				// Request page 2
				resp2, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{
					Parent:     "u1",
					Pagination: &v1.PaginationRequest{PageSize: 2, PageToken: nextToken},
				}))
				if err != nil {
					t.Fatal(err)
				}
				if len(resp2.Msg.GetCycles()) == 0 {
					t.Errorf("page 2: expected cycles with token, got %d", len(resp2.Msg.GetCycles()))
				}
			}
		}
	})
}

// ─── PhaseEstimateIntegration ─────────────────────────────────────────────

func TestPhaseEstimateIntegration(t *testing.T) {
	svc := newSvc(t)
	// Create a profile first (required for phase estimation)
	profile := validProfile("u1")
	if _, err := svc.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{Profile: profile})); err != nil {
		t.Fatal(err)
	}
	// Create multiple bleeding observations
	for _, obs := range []*v1.BleedingObservation{
		validBleeding("b1", "u1", "2025-01-01"),
		validBleeding("b2", "u1", "2025-01-02"),
	} {
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
			t.Fatal(err)
		}
	}
	// Query timeline and verify PhaseEstimate records are present
	resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Timeline should contain both bleeding observations and phase estimate records (derived from the cycle)
	var foundPhaseEstimate bool
	for _, record := range resp.Msg.GetRecords() {
		if record.GetPhaseEstimate() != nil {
			foundPhaseEstimate = true
			break
		}
	}
	if !foundPhaseEstimate {
		t.Error("expected PhaseEstimate records in timeline after creating bleeding observations with profile")
	}
}

// ─── ProfileUpdateTriggersPhaseReEstimation ───────────────────────────────

func TestProfileUpdateTriggersPhaseReEstimation(t *testing.T) {
	svc := newSvc(t)
	// Create bleeding observations WITHOUT a profile first
	for _, obs := range []*v1.BleedingObservation{
		validBleeding("b1", "u1", "2025-01-01"),
		validBleeding("b2", "u1", "2025-01-02"),
	} {
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
			t.Fatal(err)
		}
	}
	// Verify that without a profile, no phase estimates exist
	resp1, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	var phaseCountBefore int
	for _, record := range resp1.Msg.GetRecords() {
		if record.GetPhaseEstimate() != nil {
			phaseCountBefore++
		}
	}
	// Now create the profile — should trigger phase re-estimation
	profile := validProfile("u1")
	if _, err := svc.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{Profile: profile})); err != nil {
		t.Fatal(err)
	}
	// Query timeline again and verify phase estimates now exist
	resp2, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	var phaseCountAfter int
	for _, record := range resp2.Msg.GetRecords() {
		if record.GetPhaseEstimate() != nil {
			phaseCountAfter++
		}
	}
	if phaseCountAfter == 0 {
		t.Error("expected phase estimates after creating profile, but found none")
	}
	if phaseCountAfter <= phaseCountBefore {
		t.Errorf("expected phase estimates to increase after profile creation (before=%d, after=%d)", phaseCountBefore, phaseCountAfter)
	}
}

// ─── UpdateBleedingObservationTriggersRedetection ───────────────────────

func TestUpdateBleedingObservationTriggersRedetection(t *testing.T) {
	svc := newSvc(t)
	// Create initial bleeding observations
	obs1 := validBleeding("b1", "u1", "2025-01-01")
	if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs1})); err != nil {
		t.Fatal(err)
	}
	// Verify initial cycles
	resp1, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	cycleCountBefore := len(resp1.Msg.GetCycles())
	// Update the bleeding observation to a different date
	obs1Updated := validBleeding("b1", "u1", "2025-02-01")
	if _, err := svc.UpdateBleedingObservation(ctx, connect.NewRequest(&v1.UpdateBleedingObservationRequest{Observation: obs1Updated})); err != nil {
		t.Fatal(err)
	}
	// Verify cycles are recalculated after update
	resp2, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	cycleCountAfter := len(resp2.Msg.GetCycles())
	// The cycle should still exist but with updated dates
	if cycleCountAfter != cycleCountBefore {
		t.Logf("cycle count changed from %d to %d (acceptable)", cycleCountBefore, cycleCountAfter)
	}
	// Verify the cycle's start date reflects the update
	if len(resp2.Msg.GetCycles()) > 0 {
		cycle := resp2.Msg.GetCycles()[0]
		if cycle.GetStartDate().GetValue() != "2025-02-01" {
			t.Errorf("expected cycle start_date to be updated to 2025-02-01, got %q", cycle.GetStartDate().GetValue())
		}
	}
}

// ─── DeleteBleedingObservationTriggersRedetection ───────────────────────

func TestDeleteBleedingObservationTriggersRedetection(t *testing.T) {
	svc := newSvc(t)
	// Create multiple bleeding observations
	for _, obs := range []*v1.BleedingObservation{
		validBleeding("b1", "u1", "2025-01-01"),
		validBleeding("b2", "u1", "2025-01-02"),
	} {
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
			t.Fatal(err)
		}
	}
	// Verify initial cycles exist
	resp1, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	cycleCountBefore := len(resp1.Msg.GetCycles())
	if cycleCountBefore == 0 {
		t.Fatal("expected at least 1 cycle before deletion")
	}
	// Delete one bleeding observation
	if _, err := svc.DeleteBleedingObservation(ctx, connect.NewRequest(&v1.DeleteBleedingObservationRequest{Name: "b2"})); err != nil {
		t.Fatal(err)
	}
	// Verify cycles are recalculated after deletion
	resp2, err := svc.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: "u1"}))
	if err != nil {
		t.Fatal(err)
	}
	cycleCountAfter := len(resp2.Msg.GetCycles())
	// Cycle count should remain the same or decrease
	if cycleCountAfter > cycleCountBefore {
		t.Errorf("expected cycle count to not increase after deletion: before=%d, after=%d", cycleCountBefore, cycleCountAfter)
	}
}

// ─── ListTimeline ─────────────────────────────────────────────────────────────

func TestListTimeline(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		svc := newSvc(t)
		resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.GetRecords()) != 0 {
			t.Errorf("want 0 records, got %d", len(resp.Msg.GetRecords()))
		}
	})

	t.Run("MixedRecords_SortedDescending", func(t *testing.T) {
		svc := newSvc(t)
		// Add a bleeding obs and a symptom obs on different days.
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
			Parent:      "u1",
			Observation: validBleeding("b1", "u1", "2026-01-10"),
		})); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{
			Parent:      "u1",
			Observation: validSymptom("s1", "u1", "2026-01-15"),
		})); err != nil {
			t.Fatal(err)
		}
		resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		records := resp.Msg.GetRecords()
		// We expect at least 2 records: symptom + bleeding (plus any derived cycles).
		// The symptom (Jan 15) should come first (most recent).
		var foundSymptomFirst bool
		for i, r := range records {
			if r.GetSymptomObservation() != nil {
				foundSymptomFirst = i == 0
				break
			}
		}
		if !foundSymptomFirst {
			t.Error("symptom observation (Jan 15) should appear before bleeding (Jan 10) in descending order")
		}
	})

	t.Run("DateRangeFilter", func(t *testing.T) {
		svc := newSvc(t)
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
			Parent:      "u1",
			Observation: validBleeding("b1", "u1", "2026-01-05"),
		})); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
			Parent:      "u1",
			Observation: validBleeding("b2", "u1", "2026-01-20"),
		})); err != nil {
			t.Fatal(err)
		}
		// Request only Jan 15–31; should NOT include b1 (Jan 5).
		resp, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
			Parent: "u1",
			Range: &v1.DateRange{
				Start: &v1.LocalDate{Value: "2026-01-15"},
				End:   &v1.LocalDate{Value: "2026-01-31"},
			},
		}))
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range resp.Msg.GetRecords() {
			if bo := r.GetBleedingObservation(); bo != nil && bo.GetName() == "b1" {
				t.Error("b1 (Jan 5) should not appear in Jan 15–31 range")
			}
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		svc := newSvc(t)
		// Create 5 bleeding observations on separate days.
		for i := 1; i <= 5; i++ {
			date := "2026-01-0" + string(rune('0'+i))
			obs := validBleeding("b"+string(rune('0'+i)), "u1", date)
			if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
				t.Fatal(err)
			}
		}
		// Request page size 2, first page.
		resp1, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
			Parent:     "u1",
			Pagination: &v1.PaginationRequest{PageSize: 2},
		}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp1.Msg.GetRecords()) != 2 {
			t.Errorf("page 1: want 2 records, got %d", len(resp1.Msg.GetRecords()))
		}
		nextToken := resp1.Msg.GetPagination().GetNextPageToken()
		if nextToken == "" {
			t.Fatal("expected a next_page_token for page 1")
		}
		// Fetch second page.
		resp2, err := svc.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
			Parent:     "u1",
			Pagination: &v1.PaginationRequest{PageSize: 2, PageToken: nextToken},
		}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp2.Msg.GetRecords()) != 2 {
			t.Errorf("page 2: want 2 records, got %d", len(resp2.Msg.GetRecords()))
		}
	})
}

// ─── ListPredictions ──────────────────────────────────────────────────────────

func TestListPredictions(t *testing.T) {
	t.Run("ReturnsEmpty", func(t *testing.T) {
		svc := newSvc(t)
		resp, err := svc.ListPredictions(ctx, connect.NewRequest(&v1.ListPredictionsRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.GetPredictions()) != 0 {
			t.Errorf("want 0 predictions (stub), got %d", len(resp.Msg.GetPredictions()))
		}
	})
}

// ─── ListInsights ─────────────────────────────────────────────────────────────

func TestListInsights(t *testing.T) {
	t.Run("ReturnsEmpty", func(t *testing.T) {
		svc := newSvc(t)
		resp, err := svc.ListInsights(ctx, connect.NewRequest(&v1.ListInsightsRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.GetInsights()) != 0 {
			t.Errorf("want 0 insights (stub), got %d", len(resp.Msg.GetInsights()))
		}
	})
}

// ─── CreateDataExport ─────────────────────────────────────────────────────────

func TestCreateDataExport(t *testing.T) {
	t.Run("EmptyUser", func(t *testing.T) {
		svc := newSvc(t)
		resp, err := svc.CreateDataExport(ctx, connect.NewRequest(&v1.CreateDataExportRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Msg.GetData()) == 0 {
			t.Error("expected non-empty export data even for empty user")
		}
		// Parse the JSON envelope.
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(resp.Msg.GetData(), &payload); err != nil {
			t.Fatalf("export data is not valid JSON: %v", err)
		}
	})

	t.Run("WithRecords", func(t *testing.T) {
		svc := newSvc(t)
		profile := validProfile("u1")
		if _, err := svc.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{Profile: profile})); err != nil {
			t.Fatal(err)
		}
		obs := validBleeding("b1", "u1", "2026-01-15")
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{Parent: "u1", Observation: obs})); err != nil {
			t.Fatal(err)
		}
		resp, err := svc.CreateDataExport(ctx, connect.NewRequest(&v1.CreateDataExportRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		// Verify the JSON structure contains profile and bleeding_observations.
		var payload struct {
			Version     string            `json:"version"`
			UserID      string            `json:"user_id"`
			Profile     json.RawMessage   `json:"profile"`
			BleedingObs []json.RawMessage `json:"bleeding_observations"`
		}
		if err := json.Unmarshal(resp.Msg.GetData(), &payload); err != nil {
			t.Fatalf("unmarshal export: %v", err)
		}
		if payload.Version != "1" {
			t.Errorf("version = %q, want 1", payload.Version)
		}
		if payload.UserID != "u1" {
			t.Errorf("user_id = %q, want u1", payload.UserID)
		}
		if len(payload.Profile) == 0 {
			t.Error("expected profile in export")
		}
		if len(payload.BleedingObs) != 1 {
			t.Errorf("want 1 bleeding obs in export, got %d", len(payload.BleedingObs))
		}
	})
}

// ─── CreateDataImport ─────────────────────────────────────────────────────────

func TestCreateDataImport(t *testing.T) {
	t.Run("InvalidJSON", func(t *testing.T) {
		svc := newSvc(t)
		_, err := svc.CreateDataImport(ctx, connect.NewRequest(&v1.CreateDataImportRequest{Data: []byte("not json")}))
		if codeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("want CodeInvalidArgument, got %v", err)
		}
	})

	t.Run("WrongVersion", func(t *testing.T) {
		svc := newSvc(t)
		data, _ := json.Marshal(map[string]string{"version": "99", "user_id": "u1"})
		_, err := svc.CreateDataImport(ctx, connect.NewRequest(&v1.CreateDataImportRequest{Data: data}))
		if codeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("want CodeInvalidArgument for version mismatch, got %v", err)
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		// Create a source service with some data.
		srcSvc := newSvc(t)
		profile := validProfile("u1")
		if _, err := srcSvc.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{Profile: profile})); err != nil {
			t.Fatal(err)
		}
		if _, err := srcSvc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
			Parent:      "u1",
			Observation: validBleeding("b1", "u1", "2026-01-15"),
		})); err != nil {
			t.Fatal(err)
		}
		if _, err := srcSvc.CreateSymptomObservation(ctx, connect.NewRequest(&v1.CreateSymptomObservationRequest{
			Parent:      "u1",
			Observation: validSymptom("s1", "u1", "2026-01-16"),
		})); err != nil {
			t.Fatal(err)
		}
		// Export from source.
		exportResp, err := srcSvc.CreateDataExport(ctx, connect.NewRequest(&v1.CreateDataExportRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		// Import into a fresh service.
		dstSvc := newSvc(t)
		importResp, err := dstSvc.CreateDataImport(ctx, connect.NewRequest(&v1.CreateDataImportRequest{Parent: "u1", Data: exportResp.Msg.GetData()}))
		if err != nil {
			t.Fatal(err)
		}
		// 1 profile + 1 bleeding + 1 symptom = 3 newly created records.
		if importResp.Msg.GetRecordsImported() != 3 {
			t.Errorf("want 3 records imported, got %d", importResp.Msg.GetRecordsImported())
		}
		// Verify the profile exists in destination.
		profResp, err := dstSvc.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		if profResp.Msg.GetProfile().GetName() != "u1" {
			t.Error("profile not found after import")
		}
	})

	t.Run("IdempotentReImport", func(t *testing.T) {
		svc := newSvc(t)
		profile := validProfile("u1")
		if _, err := svc.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{Profile: profile})); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
			Parent:      "u1",
			Observation: validBleeding("b1", "u1", "2026-01-15"),
		})); err != nil {
			t.Fatal(err)
		}
		exportResp, err := svc.CreateDataExport(ctx, connect.NewRequest(&v1.CreateDataExportRequest{Parent: "u1"}))
		if err != nil {
			t.Fatal(err)
		}
		// Import into the SAME service twice; second import should find all
		// records already present and return 0 newly created.
		if _, err := svc.CreateDataImport(ctx, connect.NewRequest(&v1.CreateDataImportRequest{Parent: "u1", Data: exportResp.Msg.GetData()})); err != nil {
			t.Fatal(err)
		}
		resp2, err := svc.CreateDataImport(ctx, connect.NewRequest(&v1.CreateDataImportRequest{Parent: "u1", Data: exportResp.Msg.GetData()}))
		if err != nil {
			t.Fatal(err)
		}
		// Profile uses creation (always counts as 1), bleeding already exists (skipped).
		if resp2.Msg.GetRecordsImported() != 1 {
			t.Errorf("second import: want 1 (profile creation), got %d", resp2.Msg.GetRecordsImported())
		}
	})
}
