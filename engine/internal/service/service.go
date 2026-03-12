// Package service implements the Connect-RPC CycleTrackerService for the
// openmenses engine. It orchestrates storage, validation, and domain rules to
// fulfill each RPC defined in proto/openmenses/v1/service.proto.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

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

// applyUserProfileFieldMask merges updates into existing according to the
// field paths specified in mask. If mask is nil or empty, all fields from
// updates are used (full replace).
func applyUserProfileFieldMask(existing, updates *v1.UserProfile, mask *fieldmaskpb.FieldMask) *v1.UserProfile {
	// If no mask or empty mask, do a full replace of all fields
	if mask == nil || len(mask.GetPaths()) == 0 {
		return &v1.UserProfile{
			Name:             updates.GetName(),
			BiologicalCycle:  updates.GetBiologicalCycle(),
			Contraception:    updates.GetContraception(),
			CycleRegularity:  updates.GetCycleRegularity(),
			ReproductiveGoal: updates.GetReproductiveGoal(),
			HealthConditions: updates.GetHealthConditions(),
			TrackingFocus:    updates.GetTrackingFocus(),
		}
	}

	// Apply only the fields specified in the mask.
	// Note: "name" is intentionally excluded to prevent changing the profile ID.
	for _, path := range mask.GetPaths() {
		switch path {
		case "biological_cycle":
			existing.BiologicalCycle = updates.BiologicalCycle
		case "contraception":
			existing.Contraception = updates.Contraception
		case "cycle_regularity":
			existing.CycleRegularity = updates.CycleRegularity
		case "reproductive_goal":
			existing.ReproductiveGoal = updates.ReproductiveGoal
		case "health_conditions":
			existing.HealthConditions = updates.HealthConditions
		case "tracking_focus":
			existing.TrackingFocus = updates.TrackingFocus
		}
	}
	return existing
}

// ─── RPC: user profile ────────────────────────────────────────────────────────

// GetUserProfile fetches the profile for the given user name.
// Returns CodeNotFound if no profile exists.
func (s *CycleTrackerService) GetUserProfile(
	ctx context.Context,
	req *connect.Request[v1.GetUserProfileRequest],
) (*connect.Response[v1.GetUserProfileResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	profile, err := s.store.UserProfiles().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetUserProfileResponse{Profile: profile}), nil
}

// CreateUserProfile validates and persists a new user profile, assigning a
// ULID if the name is not provided. Returns CodeAlreadyExists if a profile
// with the same name already exists.
func (s *CycleTrackerService) CreateUserProfile(
	ctx context.Context,
	req *connect.Request[v1.CreateUserProfileRequest],
) (*connect.Response[v1.CreateUserProfileResponse], error) {
	profile := req.Msg.GetProfile()
	if profile == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile is required"))
	}
	if profile.GetName() == "" {
		profile.Name = newID()
	}
	if err := s.validator.ValidateUserProfile(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.UserProfiles().Create(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.CreateUserProfileResponse{Profile: profile}), nil
}

// UpdateUserProfile validates and updates an existing user profile.
// Only fields specified in the update_mask are modified. Returns CodeNotFound
// if the profile does not exist.
func (s *CycleTrackerService) UpdateUserProfile(
	ctx context.Context,
	req *connect.Request[v1.UpdateUserProfileRequest],
) (*connect.Response[v1.UpdateUserProfileResponse], error) {
	updates := req.Msg.GetProfile()
	if updates == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile is required"))
	}
	if updates.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("profile.name is required"))
	}

	// Fetch the existing profile
	existing, err := s.store.UserProfiles().GetByID(ctx, updates.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}

	// Apply the update mask to merge only specified fields
	mask := req.Msg.GetUpdateMask()
	profile := applyUserProfileFieldMask(existing, updates, mask)

	// Validate the merged profile
	if err := s.validator.ValidateUserProfile(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}

	// Persist the updated profile
	if err := s.store.UserProfiles().Update(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}

	return connect.NewResponse(&v1.UpdateUserProfileResponse{Profile: profile}), nil
}

// ─── RPC: observations ────────────────────────────────────────────────────────

// CreateBleedingObservation validates and persists a bleeding observation.
// A ULID is assigned when the request does not supply a name. Cycle
// re-detection is triggered after the observation is saved so that the stored
// cycle list stays current for timeline queries.
func (s *CycleTrackerService) CreateBleedingObservation(
	ctx context.Context,
	req *connect.Request[v1.CreateBleedingObservationRequest],
) (*connect.Response[v1.CreateBleedingObservationResponse], error) {
	userID := req.Msg.GetParent()
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	obs.UserId = userID
	if obs.GetName() == "" {
		obs.Name = newID()
	}
	if err := s.validator.ValidateBleedingObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.BleedingObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	// Best-effort: errors here do not fail the request because the observation
	// is already persisted and ListCycles re-derives on demand.
	if err := s.redetectAndStoreCycles(ctx, userID); err != nil {
		log.Printf("redetectAndStoreCycles: user %s: %v", userID, err)
	}
	return connect.NewResponse(&v1.CreateBleedingObservationResponse{Observation: obs}), nil
}

// CreateSymptomObservation validates and persists a symptom observation.
func (s *CycleTrackerService) CreateSymptomObservation(
	ctx context.Context,
	req *connect.Request[v1.CreateSymptomObservationRequest],
) (*connect.Response[v1.CreateSymptomObservationResponse], error) {
	userID := req.Msg.GetParent()
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	obs.UserId = userID
	if obs.GetName() == "" {
		obs.Name = newID()
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
	userID := req.Msg.GetParent()
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	obs.UserId = userID
	if obs.GetName() == "" {
		obs.Name = newID()
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
	userID := req.Msg.GetParent()
	med := req.Msg.GetMedication()
	if med == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("medication is required"))
	}
	med.UserId = userID
	if med.GetName() == "" {
		med.Name = newID()
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
	userID := req.Msg.GetParent()
	event := req.Msg.GetEvent()
	if event == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event is required"))
	}
	event.UserId = userID
	if event.GetName() == "" {
		event.Name = newID()
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
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
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
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	cycles, err := rules.DetectCycles(ctx, req.Msg.GetParent(), s.store)
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
	req *connect.Request[v1.ListPredictionsRequest],
) (*connect.Response[v1.ListPredictionsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListPredictionsResponse{}), nil
}

// ListInsights returns an empty list. Insight generation is deferred to
// Phase 5.
func (s *CycleTrackerService) ListInsights(
	_ context.Context,
	req *connect.Request[v1.ListInsightsRequest],
) (*connect.Response[v1.ListInsightsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListInsightsResponse{}), nil
}

// ─── RPC: bleeding observations ────────────────────────────────────────────────

// GetBleedingObservation fetches a single bleeding observation by ID.
// Returns CodeNotFound if no observation exists.
func (s *CycleTrackerService) GetBleedingObservation(
	ctx context.Context,
	req *connect.Request[v1.GetBleedingObservationRequest],
) (*connect.Response[v1.GetBleedingObservationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	obs, err := s.store.BleedingObservations().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetBleedingObservationResponse{Observation: obs}), nil
}

// UpdateBleedingObservation validates and updates an existing bleeding observation.
// Since the storage layer does not have a direct Update method for observations,
// this implementation deletes the old record and creates a new one with the same ID.
func (s *CycleTrackerService) UpdateBleedingObservation(
	ctx context.Context,
	req *connect.Request[v1.UpdateBleedingObservationRequest],
) (*connect.Response[v1.UpdateBleedingObservationResponse], error) {
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	if obs.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation.name is required"))
	}
	if err := s.validator.ValidateBleedingObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	// Delete the old record and create the updated one with the same ID
	if err := s.store.BleedingObservations().DeleteByID(ctx, obs.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, toConnectErr(err)
	}
	if err := s.store.BleedingObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.UpdateBleedingObservationResponse{Observation: obs}), nil
}

// DeleteBleedingObservation removes a bleeding observation by ID.
// Returns CodeNotFound if the observation does not exist.
func (s *CycleTrackerService) DeleteBleedingObservation(
	ctx context.Context,
	req *connect.Request[v1.DeleteBleedingObservationRequest],
) (*connect.Response[v1.DeleteBleedingObservationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.BleedingObservations().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.DeleteBleedingObservationResponse{}), nil
}

// ListBleedingObservations returns all bleeding observations for the given user
// within the optional date range, with offset-based pagination.
func (s *CycleTrackerService) ListBleedingObservations(
	ctx context.Context,
	req *connect.Request[v1.ListBleedingObservationsRequest],
) (*connect.Response[v1.ListBleedingObservationsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
	// Use full date range by default; observations list does not support date range filtering
	const start, end = "0001-01-01", "9999-12-31"
	page, err := s.store.BleedingObservations().ListByUserAndDateRange(ctx, userID, start, end, pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListBleedingObservationsResponse{
		Observations: page.Items,
		Pagination:   &v1.PaginationResponse{NextPageToken: page.NextPageToken},
	}), nil
}

// ─── RPC: symptom observations ────────────────────────────────────────────────

// GetSymptomObservation fetches a single symptom observation by ID.
// Returns CodeNotFound if no observation exists.
func (s *CycleTrackerService) GetSymptomObservation(
	ctx context.Context,
	req *connect.Request[v1.GetSymptomObservationRequest],
) (*connect.Response[v1.GetSymptomObservationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	obs, err := s.store.SymptomObservations().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetSymptomObservationResponse{Observation: obs}), nil
}

// UpdateSymptomObservation validates and updates an existing symptom observation.
// Since the storage layer does not have a direct Update method for observations,
// this implementation deletes the old record and creates a new one with the same ID.
func (s *CycleTrackerService) UpdateSymptomObservation(
	ctx context.Context,
	req *connect.Request[v1.UpdateSymptomObservationRequest],
) (*connect.Response[v1.UpdateSymptomObservationResponse], error) {
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	if obs.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation.name is required"))
	}
	if err := s.validator.ValidateSymptomObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	// Delete the old record and create the updated one with the same ID
	if err := s.store.SymptomObservations().DeleteByID(ctx, obs.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, toConnectErr(err)
	}
	if err := s.store.SymptomObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.UpdateSymptomObservationResponse{Observation: obs}), nil
}

// DeleteSymptomObservation removes a symptom observation by ID.
// Returns CodeNotFound if the observation does not exist.
func (s *CycleTrackerService) DeleteSymptomObservation(
	ctx context.Context,
	req *connect.Request[v1.DeleteSymptomObservationRequest],
) (*connect.Response[v1.DeleteSymptomObservationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.SymptomObservations().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.DeleteSymptomObservationResponse{}), nil
}

// ListSymptomObservations returns all symptom observations for the given user
// with offset-based pagination.
func (s *CycleTrackerService) ListSymptomObservations(
	ctx context.Context,
	req *connect.Request[v1.ListSymptomObservationsRequest],
) (*connect.Response[v1.ListSymptomObservationsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
	// Use full date range by default; observations list does not support date range filtering
	const start, end = "0001-01-01", "9999-12-31"
	page, err := s.store.SymptomObservations().ListByUserAndDateRange(ctx, userID, start, end, pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListSymptomObservationsResponse{
		Observations: page.Items,
		Pagination:   &v1.PaginationResponse{NextPageToken: page.NextPageToken},
	}), nil
}

// ─── RPC: mood observations ───────────────────────────────────────────────────

// GetMoodObservation fetches a single mood observation by ID.
// Returns CodeNotFound if no observation exists.
func (s *CycleTrackerService) GetMoodObservation(
	ctx context.Context,
	req *connect.Request[v1.GetMoodObservationRequest],
) (*connect.Response[v1.GetMoodObservationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	obs, err := s.store.MoodObservations().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetMoodObservationResponse{Observation: obs}), nil
}

// UpdateMoodObservation validates and updates an existing mood observation.
// Since the storage layer does not have a direct Update method for observations,
// this implementation deletes the old record and creates a new one with the same ID.
func (s *CycleTrackerService) UpdateMoodObservation(
	ctx context.Context,
	req *connect.Request[v1.UpdateMoodObservationRequest],
) (*connect.Response[v1.UpdateMoodObservationResponse], error) {
	obs := req.Msg.GetObservation()
	if obs == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation is required"))
	}
	if obs.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("observation.name is required"))
	}
	if err := s.validator.ValidateMoodObservation(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	// Delete the old record and create the updated one with the same ID
	if err := s.store.MoodObservations().DeleteByID(ctx, obs.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, toConnectErr(err)
	}
	if err := s.store.MoodObservations().Create(ctx, obs); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.UpdateMoodObservationResponse{Observation: obs}), nil
}

// DeleteMoodObservation removes a mood observation by ID.
// Returns CodeNotFound if the observation does not exist.
func (s *CycleTrackerService) DeleteMoodObservation(
	ctx context.Context,
	req *connect.Request[v1.DeleteMoodObservationRequest],
) (*connect.Response[v1.DeleteMoodObservationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.MoodObservations().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.DeleteMoodObservationResponse{}), nil
}

// ListMoodObservations returns all mood observations for the given user
// with offset-based pagination.
func (s *CycleTrackerService) ListMoodObservations(
	ctx context.Context,
	req *connect.Request[v1.ListMoodObservationsRequest],
) (*connect.Response[v1.ListMoodObservationsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
	// Use full date range by default; observations list does not support date range filtering
	const start, end = "0001-01-01", "9999-12-31"
	page, err := s.store.MoodObservations().ListByUserAndDateRange(ctx, userID, start, end, pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListMoodObservationsResponse{
		Observations: page.Items,
		Pagination:   &v1.PaginationResponse{NextPageToken: page.NextPageToken},
	}), nil
}

// ─── RPC: medications ─────────────────────────────────────────────────────────

// GetMedication fetches a single medication by ID.
// Returns CodeNotFound if no medication exists.
func (s *CycleTrackerService) GetMedication(
	ctx context.Context,
	req *connect.Request[v1.GetMedicationRequest],
) (*connect.Response[v1.GetMedicationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	med, err := s.store.Medications().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetMedicationResponse{Medication: med}), nil
}

// UpdateMedication validates and updates an existing medication.
// Returns CodeNotFound if the medication does not exist.
func (s *CycleTrackerService) UpdateMedication(
	ctx context.Context,
	req *connect.Request[v1.UpdateMedicationRequest],
) (*connect.Response[v1.UpdateMedicationResponse], error) {
	med := req.Msg.GetMedication()
	if med == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("medication is required"))
	}
	if med.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("medication.name is required"))
	}
	if err := s.validator.ValidateMedication(ctx, med); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.Medications().Update(ctx, med); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.UpdateMedicationResponse{Medication: med}), nil
}

// DeleteMedication removes a medication by ID.
// Returns CodeNotFound if the medication does not exist.
func (s *CycleTrackerService) DeleteMedication(
	ctx context.Context,
	req *connect.Request[v1.DeleteMedicationRequest],
) (*connect.Response[v1.DeleteMedicationResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.Medications().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.DeleteMedicationResponse{}), nil
}

// ListMedications returns all medications for the given user, with offset-based pagination.
func (s *CycleTrackerService) ListMedications(
	ctx context.Context,
	req *connect.Request[v1.ListMedicationsRequest],
) (*connect.Response[v1.ListMedicationsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	page, err := s.store.Medications().ListByUser(ctx, req.Msg.GetParent(), pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListMedicationsResponse{
		Medications: page.Items,
		Pagination:  &v1.PaginationResponse{NextPageToken: page.NextPageToken},
	}), nil
}

// ─── RPC: medication events ───────────────────────────────────────────────────

// GetMedicationEvent fetches a single medication event by ID.
// Returns CodeNotFound if no event exists.
func (s *CycleTrackerService) GetMedicationEvent(
	ctx context.Context,
	req *connect.Request[v1.GetMedicationEventRequest],
) (*connect.Response[v1.GetMedicationEventResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	event, err := s.store.MedicationEvents().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetMedicationEventResponse{Event: event}), nil
}

// UpdateMedicationEvent validates and updates an existing medication event.
// Since the storage layer does not have a direct Update method for events,
// this implementation deletes the old record and creates a new one with the same ID.
func (s *CycleTrackerService) UpdateMedicationEvent(
	ctx context.Context,
	req *connect.Request[v1.UpdateMedicationEventRequest],
) (*connect.Response[v1.UpdateMedicationEventResponse], error) {
	event := req.Msg.GetEvent()
	if event == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event is required"))
	}
	if event.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("event.name is required"))
	}
	if err := s.validator.ValidateMedicationEvent(ctx, event); err != nil {
		return nil, toConnectErr(err)
	}
	// Delete the old record and create the updated one with the same ID
	if err := s.store.MedicationEvents().DeleteByID(ctx, event.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, toConnectErr(err)
	}
	if err := s.store.MedicationEvents().Create(ctx, event); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.UpdateMedicationEventResponse{Event: event}), nil
}

// DeleteMedicationEvent removes a medication event by ID.
// Returns CodeNotFound if the event does not exist.
func (s *CycleTrackerService) DeleteMedicationEvent(
	ctx context.Context,
	req *connect.Request[v1.DeleteMedicationEventRequest],
) (*connect.Response[v1.DeleteMedicationEventResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	if err := s.store.MedicationEvents().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.DeleteMedicationEventResponse{}), nil
}

// ListMedicationEvents returns all medication events for the given user
// with offset-based pagination.
func (s *CycleTrackerService) ListMedicationEvents(
	ctx context.Context,
	req *connect.Request[v1.ListMedicationEventsRequest],
) (*connect.Response[v1.ListMedicationEventsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
	// Use full date range by default; events list does not support date range filtering
	const start, end = "0001-01-01", "9999-12-31"
	page, err := s.store.MedicationEvents().ListByUserAndDateRange(ctx, userID, start, end, pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListMedicationEventsResponse{
		Events:     page.Items,
		Pagination: &v1.PaginationResponse{NextPageToken: page.NextPageToken},
	}), nil
}

// ─── RPC: cycles ──────────────────────────────────────────────────────────────

// GetCycle fetches a single cycle by ID.
// Returns CodeNotFound if no cycle exists.
func (s *CycleTrackerService) GetCycle(
	ctx context.Context,
	req *connect.Request[v1.GetCycleRequest],
) (*connect.Response[v1.GetCycleResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	cycle, err := s.store.Cycles().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetCycleResponse{Cycle: cycle}), nil
}

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
	userID := req.Msg.GetName()
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
			log.Printf("CreateDataImport: redetectAndStoreCycles: user %s: %v", payload.UserID, err)
		}
	}

	return connect.NewResponse(&v1.CreateDataImportResponse{RecordsImported: count}), nil
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
			if err := s.store.PhaseEstimates().DeleteByCycleID(ctx, c.GetName()); err != nil {
				return err
			}
			if err := s.store.Cycles().DeleteByID(ctx, c.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
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
