// Package storage defines the Repository interface and standard error types
// for all openmenses persistence backends.
package storage

import (
	"context"
	"errors"

	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// Standard sentinel errors returned by all backends.
var (
	// ErrNotFound is returned when a requested record does not exist.
	ErrNotFound = errors.New("record not found")

	// ErrConflict is returned when a record with the same ID already exists,
	// or when an operation would violate a uniqueness constraint.
	ErrConflict = errors.New("record conflict")

	// ErrInvalidInput is returned when the caller provides invalid arguments
	// that are caught at the storage layer (e.g. empty ID).
	ErrInvalidInput = errors.New("invalid input")
)

// ListPage carries a page of results and the token to fetch the next page.
// NextPageToken is empty when there are no more results.
type ListPage[T any] struct {
	Items         []T
	NextPageToken string
}

// PageRequest carries pagination parameters for list operations.
type PageRequest struct {
	// PageSize is the maximum number of items to return. 0 means use the
	// backend default (typically 50). Maximum is 500.
	PageSize uint32
	// PageToken is an opaque token returned by a previous list call.
	// Empty string means start from the beginning.
	PageToken string
}

// Repository is the aggregate interface that all storage backends must implement.
type Repository interface {
	UserProfiles() UserProfileRepository
	BleedingObservations() BleedingObservationRepository
	SymptomObservations() SymptomObservationRepository
	MoodObservations() MoodObservationRepository
	Medications() MedicationRepository
	MedicationEvents() MedicationEventRepository
	Cycles() CycleRepository
	PhaseEstimates() PhaseEstimateRepository
	Predictions() PredictionRepository
	Insights() InsightRepository
}

// UserProfileRepository manages UserProfile records.
type UserProfileRepository interface {
	GetByID(ctx context.Context, id string) (*v1.UserProfile, error)
	Upsert(ctx context.Context, profile *v1.UserProfile) error
}

// BleedingObservationRepository manages BleedingObservation records.
type BleedingObservationRepository interface {
	Create(ctx context.Context, obs *v1.BleedingObservation) error
	GetByID(ctx context.Context, id string) (*v1.BleedingObservation, error)
	ListByUserAndDateRange(ctx context.Context, userID string, start, end string, page PageRequest) (ListPage[*v1.BleedingObservation], error)
	DeleteByID(ctx context.Context, id string) error
}

// SymptomObservationRepository manages SymptomObservation records.
type SymptomObservationRepository interface {
	Create(ctx context.Context, obs *v1.SymptomObservation) error
	GetByID(ctx context.Context, id string) (*v1.SymptomObservation, error)
	ListByUserAndDateRange(ctx context.Context, userID string, start, end string, page PageRequest) (ListPage[*v1.SymptomObservation], error)
	DeleteByID(ctx context.Context, id string) error
}

// MoodObservationRepository manages MoodObservation records.
type MoodObservationRepository interface {
	Create(ctx context.Context, obs *v1.MoodObservation) error
	GetByID(ctx context.Context, id string) (*v1.MoodObservation, error)
	ListByUserAndDateRange(ctx context.Context, userID string, start, end string, page PageRequest) (ListPage[*v1.MoodObservation], error)
	DeleteByID(ctx context.Context, id string) error
}

// MedicationRepository manages Medication records.
type MedicationRepository interface {
	Create(ctx context.Context, med *v1.Medication) error
	GetByID(ctx context.Context, id string) (*v1.Medication, error)
	ListByUser(ctx context.Context, userID string, page PageRequest) (ListPage[*v1.Medication], error)
	Update(ctx context.Context, med *v1.Medication) error
	DeleteByID(ctx context.Context, id string) error
}

// MedicationEventRepository manages MedicationEvent records.
type MedicationEventRepository interface {
	Create(ctx context.Context, ev *v1.MedicationEvent) error
	GetByID(ctx context.Context, id string) (*v1.MedicationEvent, error)
	ListByUserAndDateRange(ctx context.Context, userID string, start, end string, page PageRequest) (ListPage[*v1.MedicationEvent], error)
	ListByMedicationID(ctx context.Context, medicationID string, page PageRequest) (ListPage[*v1.MedicationEvent], error)
	DeleteByID(ctx context.Context, id string) error
}

// CycleRepository manages Cycle records.
type CycleRepository interface {
	Create(ctx context.Context, cycle *v1.Cycle) error
	GetByID(ctx context.Context, id string) (*v1.Cycle, error)
	ListByUser(ctx context.Context, userID string, page PageRequest) (ListPage[*v1.Cycle], error)
	ListByUserAndDateRange(ctx context.Context, userID string, start, end string, page PageRequest) (ListPage[*v1.Cycle], error)
	Update(ctx context.Context, cycle *v1.Cycle) error
	DeleteByID(ctx context.Context, id string) error
}

// PhaseEstimateRepository manages PhaseEstimate records.
type PhaseEstimateRepository interface {
	Create(ctx context.Context, est *v1.PhaseEstimate) error
	ListByUserAndDateRange(ctx context.Context, userID string, start, end string, page PageRequest) (ListPage[*v1.PhaseEstimate], error)
	DeleteByCycleID(ctx context.Context, cycleID string) error
}

// PredictionRepository manages Prediction records.
type PredictionRepository interface {
	Create(ctx context.Context, pred *v1.Prediction) error
	ListByUser(ctx context.Context, userID string, page PageRequest) (ListPage[*v1.Prediction], error)
	DeleteByUser(ctx context.Context, userID string) error
}

// InsightRepository manages Insight records.
type InsightRepository interface {
	Create(ctx context.Context, insight *v1.Insight) error
	ListByUser(ctx context.Context, userID string, page PageRequest) (ListPage[*v1.Insight], error)
	DeleteByUser(ctx context.Context, userID string) error
}
