// Package service implements the Connect-RPC CycleTrackerService for the
// openmenses engine. It orchestrates storage, validation, and domain rules to
// fulfill each RPC defined in proto/openmenses/v1/service.proto.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/oklog/ulid/v2"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/timeline"
	"github.com/2ajoyce/openmenses/engine/internal/validation"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"
)

// CycleTrackerService implements openmensesv1connect.CycleTrackerServiceHandler.
type CycleTrackerService struct {
	store     storage.Repository
	validator *validation.Validator
}

// Compile-time assertion that CycleTrackerService satisfies the handler interface.
var _ openmensesv1connect.CycleTrackerServiceHandler = (*CycleTrackerService)(nil)

// New creates a CycleTrackerService backed by the provided store.
func New(store storage.Repository) (*CycleTrackerService, error) {
	v, err := validation.New(store)
	if err != nil {
		return nil, fmt.Errorf("service init: %w", err)
	}
	return &CycleTrackerService{store: store, validator: v}, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// newID returns a fresh ULID string for use as a record identifier.
func newID() string {
	return ulid.Make().String()
}

// toConnectErr maps internal errors to Connect-RPC error codes.
func toConnectErr(err error) error {
	if err == nil {
		return nil
	}
	var valErr *validation.Error
	if errors.As(err, &valErr) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, storage.ErrConflict) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	if errors.Is(err, storage.ErrInvalidInput) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// pageReq converts a proto PaginationRequest to a storage PageRequest.
func pageReq(p *v1.PaginationRequest) storage.PageRequest {
	if p == nil {
		return storage.PageRequest{}
	}
	return storage.PageRequest{PageSize: p.GetPageSize(), PageToken: p.GetPageToken()}
}

// paginateAll calls fetch repeatedly with successive page tokens until all
// pages have been collected and returns the combined slice.
func paginateAll[T any](ctx context.Context, fetch func(ctx context.Context, token string) ([]T, string, error)) ([]T, error) {
	var all []T
	token := ""
	for {
		items, next, err := fetch(ctx, token)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if next == "" {
			break
		}
		token = next
	}
	return all, nil
}

var protoJSONMarshal = protojson.MarshalOptions{EmitUnpopulated: false}

// marshalProtoJSON serialises a proto message to a protojson-encoded
// json.RawMessage.
func marshalProtoJSON(msg proto.Message) (json.RawMessage, error) {
	b, err := protoJSONMarshal.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// marshalAll serialises a slice of proto messages to a slice of
// json.RawMessage values.
func marshalAll[T proto.Message](items []T) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		b, err := marshalProtoJSON(item)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

// ─── RPC: user profile ────────────────────────────────────────────────────────

// GetUserProfile fetches the profile for the given user ID.
// Returns CodeNotFound if no profile exists.
func (s *CycleTrackerService) GetUserProfile(
	ctx context.Context,
	req *connect.Request[v1.GetUserProfileRequest],
) (*connect.Response[v1.GetUserProfileResponse], error) {
	profile, err := s.store.UserProfiles().GetByID(ctx, req.Msg.GetUserId())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetUserProfileResponse{Profile: profile}), nil
}

// UpsertUserProfile validates and persists a user profile, creating or
// replacing any existing profile for the same user ID.
func (s *CycleTrackerService) UpsertUserProfile(
	ctx context.Context,
	req *connect.Request[v1.UpsertUserProfileRequest],
) (*connect.Response[v1.UpsertUserProfileResponse], error) {
	profile := req.Msg.GetProfile()
	if err := s.validator.ValidateUserProfile(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.UserProfiles().Upsert(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.UpsertUserProfileResponse{Profile: profile}), nil
}

// ─── RPC: observations ────────────────────────────────────────────────────────

// CreateBleedingObservation validates and persists a bleeding observation.
// A ULID is assigned when the request does not supply an ID. Cycle
// re-detection is triggered after the observation is saved so that the stored
// cycle list stays current for timeline queries.
func (s *CycleTrackerService) CreateBleedingObservation(
	ctx context.Context,
	req *connect.Request[v1.CreateBleedingObservationRequest],
) (*connect.Response[v1.CreateBleedingObservationResponse], error) {
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	if obs.GetId() == "" {
		obs.Id = newID()
	}
	if err := s.validator.ValidateBleedingObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.BleedingObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	// Best-effort: errors here do not fail the request because the observation
	// is already persisted and ListCycles re-derives on demand.
	_ = s.redetectAndStoreCycles(ctx, obs.GetUserId())
	return connect.NewResponse(&v1.CreateBleedingObservationResponse{Observation: obs}), nil
}

// CreateSymptomObservation validates and persists a symptom observation.
func (s *CycleTrackerService) CreateSymptomObservation(
	ctx context.Context,
	req *connect.Request[v1.CreateSymptomObservationRequest],
) (*connect.Response[v1.CreateSymptomObservationResponse], error) {
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	if obs.GetId() == "" {
		obs.Id = newID()
	}
	if err := s.validator.ValidateSymptomObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.SymptomObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.CreateSymptomObservationResponse{Observation: obs}), nil
}

// CreateMoodObservation validates and persists a mood observation.
func (s *CycleTrackerService) CreateMoodObservation(
	ctx context.Context,
	req *connect.Request[v1.CreateMoodObservationRequest],
) (*connect.Response[v1.CreateMoodObservationResponse], error) {
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	if obs.GetId() == "" {
		obs.Id = newID()
	}
	if err := s.validator.ValidateMoodObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.MoodObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.CreateMoodObservationResponse{Observation: obs}), nil
}

// ─── RPC: medications ─────────────────────────────────────────────────────────

// CreateMedication validates and persists a new medication record.
func (s *CycleTrackerService) CreateMedication(
	ctx context.Context,
	req *connect.Request[v1.CreateMedicationRequest],
) (*connect.Response[v1.CreateMedicationResponse], error) {
	med := req.Msg.GetMedication()
	if med == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("medication is required"))
	}
	if med.GetId() == "" {
		med.Id = newID()
	}
	if err := s.validator.ValidateMedication(ctx, med); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.Medications().Create(ctx, med); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.CreateMedicationResponse{Medication: med}), nil
}

// CreateMedicationEvent validates (including referential integrity) and
// persists a medication event.
func (s *CycleTrackerService) CreateMedicationEvent(
	ctx context.Context,
	req *connect.Request[v1.CreateMedicationEventRequest],
) (*connect.Response[v1.CreateMedicationEventResponse], error) {
	event := req.Msg.GetEvent()
	if event == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event is required"))
	}
	if event.GetId() == "" {
		event.Id = newID()
	}
	if err := s.validator.ValidateMedicationEvent(ctx, event); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.MedicationEvents().Create(ctx, event); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.CreateMedicationEventResponse{Event: event}), nil
}

// ─── RPC: timeline ────────────────────────────────────────────────────────────

// ListTimeline queries all record types within the requested date range, merges
// them into a single chronological (most-recent-first) list, and applies
// offset-based pagination. The core assembly logic lives in the timeline
// package and is invoked here.
func (s *CycleTrackerService) ListTimeline(
	ctx context.Context,
	req *connect.Request[v1.ListTimelineRequest],
) (*connect.Response[v1.ListTimelineResponse], error) {
	userID := req.Msg.GetUserId()
	start, end := "0001-01-01", "9999-12-31"
	if r := req.Msg.GetRange(); r != nil {
		if r.GetStart().GetValue() != "" {
			start = r.GetStart().GetValue()
		}
		if r.GetEnd().GetValue() != "" {
			end = r.GetEnd().GetValue()
		}
	}

	records, nextToken, err := timeline.BuildTimeline(ctx, s.store, userID, start, end, pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListTimelineResponse{
		Records:    records,
		Pagination: &v1.PaginationResponse{NextPageToken: nextToken},
	}), nil
}

// ─── RPC: cycles ──────────────────────────────────────────────────────────────

// ListCycles returns all cycles for the given user, computing derived cycles
// on demand from stored bleeding observations.
func (s *CycleTrackerService) ListCycles(
	ctx context.Context,
	req *connect.Request[v1.ListCyclesRequest],
) (*connect.Response[v1.ListCyclesResponse], error) {
	cycles, err := rules.DetectCycles(ctx, req.Msg.GetUserId(), s.store)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListCyclesResponse{Cycles: cycles}), nil
}

// ─── RPC: predictions & insights (Phase 4/5 stubs) ───────────────────────────

// ListPredictions returns an empty list. Prediction generation is deferred to
// Phase 4.
func (s *CycleTrackerService) ListPredictions(
	_ context.Context,
	_ *connect.Request[v1.ListPredictionsRequest],
) (*connect.Response[v1.ListPredictionsResponse], error) {
	return connect.NewResponse(&v1.ListPredictionsResponse{}), nil
}

// ListInsights returns an empty list. Insight generation is deferred to
// Phase 5.
func (s *CycleTrackerService) ListInsights(
	_ context.Context,
	_ *connect.Request[v1.ListInsightsRequest],
) (*connect.Response[v1.ListInsightsResponse], error) {
	return connect.NewResponse(&v1.ListInsightsResponse{}), nil
}

// ─── RPC: export / import ─────────────────────────────────────────────────────

// exportPayload is the JSON envelope used by ExportData / ImportData.
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

// ExportData serialises all data for the requested user to a JSON-encoded byte
// slice. Derived cycles are excluded; only user-confirmed cycles are exported.
func (s *CycleTrackerService) ExportData(
	ctx context.Context,
	req *connect.Request[v1.ExportDataRequest],
) (*connect.Response[v1.ExportDataResponse], error) {
	userID := req.Msg.GetUserId()
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
	return connect.NewResponse(&v1.ExportDataResponse{Data: data}), nil
}

// ImportData deserialises a byte slice produced by ExportData, validates every
// record, and persists it. Records whose ID already exists are skipped without
// error. Returns the count of newly created records.
func (s *CycleTrackerService) ImportData(
	ctx context.Context,
	req *connect.Request[v1.ImportDataRequest],
) (*connect.Response[v1.ImportDataResponse], error) {
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

	return connect.NewResponse(&v1.ImportDataResponse{RecordsImported: count}), nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// redetectAndStoreCycles recomputes derived cycles for userID and replaces the
// stored derived cycles with the new set. User-confirmed cycles are never
// modified. Phase estimates are regenerated for all cycles after detection.
func (s *CycleTrackerService) redetectAndStoreCycles(ctx context.Context, userID string) error {
	// Delete all existing derived cycles and their associated phase estimates.
	existing, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.Cycle, string, error) {
		pg, err := s.store.Cycles().ListByUser(ctx, userID, storage.PageRequest{PageSize: 500, PageToken: token})
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return err
	}
	for _, c := range existing {
		if c.GetSource() == v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING {
			if err := s.store.PhaseEstimates().DeleteByCycleID(ctx, c.GetId()); err != nil {
				return err
			}
			if err := s.store.Cycles().DeleteByID(ctx, c.GetId()); err != nil && !errors.Is(err, storage.ErrNotFound) {
				return err
			}
		}
	}

	// Compute fresh derived cycles and persist them.
	newCycles, err := rules.DetectCycles(ctx, userID, s.store)
	if err != nil {
		return err
	}
	for _, c := range newCycles {
		if c.GetSource() == v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING {
			if err := s.store.Cycles().Create(ctx, c); err != nil {
				return err
			}
		}
	}

	// Estimate and store phases for all cycles (derived + confirmed).
	return s.estimateAndStorePhases(ctx, userID, newCycles)
}

// estimateAndStorePhases computes PhaseEstimates for each cycle and persists
// them. Conflicts (already-existing estimates) are silently skipped.
func (s *CycleTrackerService) estimateAndStorePhases(ctx context.Context, userID string, cycles []*v1.Cycle) error {
	profile, err := s.store.UserProfiles().GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil // no profile yet — estimation requires profile
		}
		return err
	}

	stats := rules.Stats(cycles)
	avgLen := int(math.Round(stats.Average))
	completed := countCompletedCycles(cycles)

	for _, c := range cycles {
		estimates := rules.EstimatePhases(c, profile, avgLen, completed)
		for _, est := range estimates {
			if err := s.store.PhaseEstimates().Create(ctx, est); err != nil {
				if !errors.Is(err, storage.ErrConflict) {
					return err
				}
			}
		}
	}
	return nil
}

// countCompletedCycles returns the number of cycles with a non-empty end_date.
func countCompletedCycles(cycles []*v1.Cycle) int {
	n := 0
	for _, c := range cycles {
		if c.GetEndDate().GetValue() != "" {
			n++
		}
	}
	return n
}
