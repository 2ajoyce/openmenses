package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// ─── RPC: export / import ─────────────────────────────────────────────────────

// exportPayload is the JSON envelope used by CreateDataExport / CreateDataImport.
// Each slice field holds protojson-encoded representations of the corresponding
// proto messages.
type exportPayload struct {
	Version          string            `json:"version"`
	UserID           string            `json:"user_id"`
	Profile          json.RawMessage   `json:"profile,omitempty"`
	BleedingObs      []json.RawMessage `json:"bleeding_observations,omitempty"`
	SymptomObs       []json.RawMessage `json:"symptom_observations,omitempty"`
	MoodObs          []json.RawMessage `json:"mood_observations,omitempty"`
	Medications      []json.RawMessage `json:"medications,omitempty"`
	MedicationEvents []json.RawMessage `json:"medication_events,omitempty"`
	Cycles           []json.RawMessage `json:"cycles,omitempty"`
}

const exportFormatVersion = "1"

// CreateDataExport serialises all data for the requested user to a JSON-encoded byte
// slice. Derived cycles are excluded; only user-confirmed cycles are exported.
func (s *CycleTrackerService) CreateDataExport(
	ctx context.Context,
	req *connect.Request[v1.CreateDataExportRequest],
) (*connect.Response[v1.CreateDataExportResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
	payload := exportPayload{Version: exportFormatVersion, UserID: userID}

	// UserProfile (optional – may not exist for new users)
	prof, err := s.store.UserProfiles().GetByID(ctx, userID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, toConnectErr(err)
	}
	if prof != nil {
		b, err := marshalProtoJSON(prof)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal profile: %w", err))
		}
		payload.Profile = b
	}

	// Bleeding observations
	bleedings, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.BleedingObservation, string, error) {
		pg, err := s.store.BleedingObservations().ListByUserAndDateRange(
			ctx, userID, "0001-01-01", "9999-12-31",
			storage.PageRequest{PageSize: 500, PageToken: token},
		)
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return nil, toConnectErr(err)
	}
	if payload.BleedingObs, err = marshalAll(bleedings); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal bleeding: %w", err))
	}

	// Symptom observations
	symptoms, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.SymptomObservation, string, error) {
		pg, err := s.store.SymptomObservations().ListByUserAndDateRange(
			ctx, userID, "0001-01-01", "9999-12-31",
			storage.PageRequest{PageSize: 500, PageToken: token},
		)
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return nil, toConnectErr(err)
	}
	if payload.SymptomObs, err = marshalAll(symptoms); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal symptoms: %w", err))
	}

	// Mood observations
	moods, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.MoodObservation, string, error) {
		pg, err := s.store.MoodObservations().ListByUserAndDateRange(
			ctx, userID, "0001-01-01", "9999-12-31",
			storage.PageRequest{PageSize: 500, PageToken: token},
		)
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return nil, toConnectErr(err)
	}
	if payload.MoodObs, err = marshalAll(moods); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal moods: %w", err))
	}

	// Medications
	meds, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.Medication, string, error) {
		pg, err := s.store.Medications().ListByUser(
			ctx, userID,
			storage.PageRequest{PageSize: 500, PageToken: token},
		)
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return nil, toConnectErr(err)
	}
	if payload.Medications, err = marshalAll(meds); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal medications: %w", err))
	}

	// Medication events
	medEvents, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.MedicationEvent, string, error) {
		pg, err := s.store.MedicationEvents().ListByUserAndDateRange(
			ctx, userID, "0001-01-01", "9999-12-31",
			storage.PageRequest{PageSize: 500, PageToken: token},
		)
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return nil, toConnectErr(err)
	}
	if payload.MedicationEvents, err = marshalAll(medEvents); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal medication events: %w", err))
	}

	// Cycles: export only user-confirmed cycles (derived cycles are recomputed).
	allCycles, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.Cycle, string, error) {
		pg, err := s.store.Cycles().ListByUser(
			ctx, userID,
			storage.PageRequest{PageSize: 500, PageToken: token},
		)
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return nil, toConnectErr(err)
	}
	var confirmedCycles []*v1.Cycle
	for _, c := range allCycles {
		if c.GetSource() == v1.CycleSource_CYCLE_SOURCE_USER_CONFIRMED {
			confirmedCycles = append(confirmedCycles, c)
		}
	}
	if payload.Cycles, err = marshalAll(confirmedCycles); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal cycles: %w", err))
	}

	data, err := json.Marshal(&payload)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal export payload: %w", err))
	}
	return connect.NewResponse(&v1.CreateDataExportResponse{Data: data}), nil
}

// CreateDataImport deserialises a byte slice produced by CreateDataExport, validates every
// record, and persists it. Records whose name already exists are skipped without
// error. Returns the count of newly created records.
func (s *CycleTrackerService) CreateDataImport(
	ctx context.Context,
	req *connect.Request[v1.CreateDataImportRequest],
) (*connect.Response[v1.CreateDataImportResponse], error) {
	var payload exportPayload
	if err := json.Unmarshal(req.Msg.GetData(), &payload); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid import data: %w", err))
	}
	if payload.Version != exportFormatVersion {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unsupported export format version %q (expected %q)", payload.Version, exportFormatVersion))
	}

	var count uint32

	// UserProfile
	if len(payload.Profile) > 0 {
		var prof v1.UserProfile
		if err := protojson.Unmarshal(payload.Profile, &prof); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal profile: %w", err))
		}
		if err := s.validator.ValidateUserProfile(ctx, &prof); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid profile: %w", err))
		}
		if err := s.store.UserProfiles().Upsert(ctx, &prof); err != nil {
			return nil, toConnectErr(err)
		}
		count++
	}

	// Bleeding observations
	for _, raw := range payload.BleedingObs {
		var obs v1.BleedingObservation
		if err := protojson.Unmarshal(raw, &obs); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal bleeding: %w", err))
		}
		if err := s.validator.ValidateBleedingObservation(ctx, &obs); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid bleeding obs: %w", err))
		}
		if err := s.store.BleedingObservations().Create(ctx, &obs); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				continue
			}
			return nil, toConnectErr(err)
		}
		count++
	}

	// Symptom observations
	for _, raw := range payload.SymptomObs {
		var obs v1.SymptomObservation
		if err := protojson.Unmarshal(raw, &obs); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal symptom: %w", err))
		}
		if err := s.validator.ValidateSymptomObservation(ctx, &obs); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid symptom obs: %w", err))
		}
		if err := s.store.SymptomObservations().Create(ctx, &obs); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				continue
			}
			return nil, toConnectErr(err)
		}
		count++
	}

	// Mood observations
	for _, raw := range payload.MoodObs {
		var obs v1.MoodObservation
		if err := protojson.Unmarshal(raw, &obs); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal mood: %w", err))
		}
		if err := s.validator.ValidateMoodObservation(ctx, &obs); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid mood obs: %w", err))
		}
		if err := s.store.MoodObservations().Create(ctx, &obs); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				continue
			}
			return nil, toConnectErr(err)
		}
		count++
	}

	// Medications (imported before medication events to satisfy referential integrity)
	for _, raw := range payload.Medications {
		var med v1.Medication
		if err := protojson.Unmarshal(raw, &med); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal medication: %w", err))
		}
		if err := s.validator.ValidateMedication(ctx, &med); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid medication: %w", err))
		}
		if err := s.store.Medications().Create(ctx, &med); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				continue
			}
			return nil, toConnectErr(err)
		}
		count++
	}

	// Medication events
	for _, raw := range payload.MedicationEvents {
		var ev v1.MedicationEvent
		if err := protojson.Unmarshal(raw, &ev); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal medication event: %w", err))
		}
		if err := s.validator.ValidateMedicationEvent(ctx, &ev); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid medication event: %w", err))
		}
		if err := s.store.MedicationEvents().Create(ctx, &ev); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				continue
			}
			return nil, toConnectErr(err)
		}
		count++
	}

	// User-confirmed cycles
	for _, raw := range payload.Cycles {
		var c v1.Cycle
		if err := protojson.Unmarshal(raw, &c); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unmarshal cycle: %w", err))
		}
		if err := s.validator.ValidateCycle(ctx, &c); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid cycle: %w", err))
		}
		if err := s.store.Cycles().Create(ctx, &c); err != nil {
			if errors.Is(err, storage.ErrConflict) {
				continue
			}
			return nil, toConnectErr(err)
		}
		count++
	}

	// Trigger cycle re-detection and phase estimation now that all observations
	// have been persisted. Errors are logged but not propagated so that the
	// import itself is not rolled back.
	if payload.UserID != "" {
		if err := s.redetectAndStoreCycles(ctx, payload.UserID); err != nil {
			slog.Error("CreateDataImport: redetectAndStoreCycles failed", "user_id", payload.UserID, "error", err)
		}
	}

	return connect.NewResponse(&v1.CreateDataImportResponse{RecordsImported: count}), nil
}
