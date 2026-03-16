package service

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"

	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

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
	// Best-effort: creating a profile may unlock phase estimation for users
	// who already have bleeding observations.
	if profile.GetBiologicalCycle() != v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_UNSPECIFIED ||
		profile.GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_UNSPECIFIED {
		if err := s.redetectAndStoreCycles(ctx, profile.GetName()); err != nil {
			slog.Error("redetectAndStoreCycles failed after profile create", "user_id", profile.GetName(), "error", err)
		}
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
	profile, err := applyUserProfileFieldMask(existing, updates, mask)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Validate the merged profile
	if err := s.validator.ValidateUserProfile(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}

	// Persist the updated profile
	if err := s.store.UserProfiles().Update(ctx, profile); err != nil {
		return nil, toConnectErr(err)
	}

	// Best-effort: re-run cycle detection and phase estimation if fields that
	// affect phase estimates changed, so existing cycles get updated estimates
	// without requiring a new bleeding observation.
	biologicalCycleChanged := existing.GetBiologicalCycle() != profile.GetBiologicalCycle()
	cycleRegularityChanged := existing.GetCycleRegularity() != profile.GetCycleRegularity()
	if biologicalCycleChanged || cycleRegularityChanged {
		if err := s.redetectAndStoreCycles(ctx, profile.GetName()); err != nil {
			slog.Error("redetectAndStoreCycles failed after profile update", "user_id", profile.GetName(), "error", err)
		}
	}

	return connect.NewResponse(&v1.UpdateUserProfileResponse{Profile: profile}), nil
}
