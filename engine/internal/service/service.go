// Package service implements the Connect-RPC CycleTrackerService for the
// openmenses engine. It orchestrates storage, validation, and domain rules to
// fulfill each RPC defined in proto/openmenses/v1/service.proto.
//
// Methods are split across multiple files for maintainability:
//   - service.go: Core service type and constructor
//   - helpers.go: Common helper functions
//   - profiles.go: User profile methods
//   - observations.go: Observation (bleeding, symptom, mood) methods
//   - medications.go: Medication and medication event methods
//   - cycles.go: Cycle, timeline, predictions, insights methods
//   - exports.go: Data export/import methods
package service

import (
	"fmt"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/validation"
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
