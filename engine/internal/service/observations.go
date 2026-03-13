package service

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

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
		slog.Error("redetectAndStoreCycles failed", "user_id", userID, "error", err)
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
