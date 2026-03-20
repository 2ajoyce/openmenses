package service

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

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
	s.triggerInsightRegeneration(ctx, userID)
	return connect.NewResponse(&v1.CreateMedicationResponse{Medication: med}), nil
}

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
	s.triggerInsightRegeneration(ctx, med.GetUserId())
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
	med, err := s.store.Medications().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	userID := med.GetUserId()
	if err := s.store.Medications().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	s.triggerInsightRegeneration(ctx, userID)
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
	s.triggerInsightRegeneration(ctx, userID)
	return connect.NewResponse(&v1.CreateMedicationEventResponse{Event: event}), nil
}

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
	// Fetch the existing event to get the authoritative user ID before overwriting.
	existing, err := s.store.MedicationEvents().GetByID(ctx, event.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	userID := existing.GetUserId()
	// Delete the old record and create the updated one with the same ID
	if err := s.store.MedicationEvents().DeleteByID(ctx, event.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
		return nil, toConnectErr(err)
	}
	if err := s.store.MedicationEvents().Create(ctx, event); err != nil {
		return nil, toConnectErr(err)
	}
	s.triggerInsightRegeneration(ctx, userID)
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
	event, err := s.store.MedicationEvents().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	userID := event.GetUserId()
	if err := s.store.MedicationEvents().DeleteByID(ctx, req.Msg.GetName()); err != nil {
		return nil, toConnectErr(err)
	}
	s.triggerInsightRegeneration(ctx, userID)
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

// triggerInsightRegeneration fetches all cycles for userID and regenerates
// insights. Errors are logged but do not surface to the caller.
func (s *CycleTrackerService) triggerInsightRegeneration(ctx context.Context, userID string) {
	cycles, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.Cycle, string, error) {
		pg, err := s.store.Cycles().ListByUser(ctx, userID, storage.PageRequest{PageSize: 500, PageToken: token})
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		slog.Error("failed to fetch cycles for insight regeneration", "user_id", userID, "error", err)
		return
	}
	if err := s.regenerateAndStoreInsights(ctx, userID, cycles); err != nil {
		slog.Error("failed to regenerate insights after medication change", "user_id", userID, "error", err)
	}
}
