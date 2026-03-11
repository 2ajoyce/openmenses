// Package validation provides schema-level and domain-level validation for all
// openmenses proto messages. Schema validation delegates to protovalidate, which
// enforces the constraints declared in the .proto files as buf.validate
// annotations. Domain validation enforces cross-field, cross-record, temporal,
// and referential-integrity rules that cannot be expressed in the schema.
package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/proto"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// FieldViolation describes a single constraint violation on a named field.
type FieldViolation struct {
	// Field is a dot-separated path to the offending field, e.g. "cycle.end_date".
	Field string
	// Description is a human-readable English explanation of the failure.
	Description string
}

// Error is a validation error that carries one or more FieldViolations.
// All violations from a single validation pass are captured together so
// the caller can report all of them at once.
type Error struct {
	Violations []FieldViolation
}

// Error implements the error interface.
func (e *Error) Error() string {
	if len(e.Violations) == 0 {
		return "validation error"
	}
	msgs := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		msgs[i] = v.Field + ": " + v.Description
	}
	if len(msgs) == 1 {
		return "validation error: " + msgs[0]
	}
	return "validation errors: " + strings.Join(msgs, "; ")
}

// Validator runs schema-level (protovalidate) and domain-level validation.
// Schema validation always runs first; domain checks run only when schema passes.
type Validator struct {
	pv    protovalidate.Validator
	store storage.Repository
	// Now returns the current wall-clock time. Override in tests for
	// deterministic temporal validation.
	Now func() time.Time
}

// New creates a Validator backed by the given storage for cross-record checks.
func New(store storage.Repository) (*Validator, error) {
	pv, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("protovalidate init: %w", err)
	}
	return &Validator{pv: pv, store: store, Now: time.Now}, nil
}

// schemaValidate runs protovalidate and converts its violations to FieldViolations.
func (v *Validator) schemaValidate(msg proto.Message) []FieldViolation {
	err := v.pv.Validate(msg)
	if err == nil {
		return nil
	}
	var valErr *protovalidate.ValidationError
	if !errors.As(err, &valErr) {
		return []FieldViolation{{Field: "<message>", Description: err.Error()}}
	}
	out := make([]FieldViolation, 0, len(valErr.Violations))
	for _, viol := range valErr.Violations {
		out = append(out, FieldViolation{
			Field:       protovalidate.FieldPathString(viol.Proto.GetField()),
			Description: viol.Proto.GetMessage(),
		})
	}
	return out
}

// finalise returns nil when there are no violations, otherwise wraps them.
func finalise(viols []FieldViolation) error {
	if len(viols) == 0 {
		return nil
	}
	return &Error{Violations: viols}
}

// futureViolation returns a FieldViolation when tsValue is more than 1 minute
// in the future relative to v.Now(). Returns a FieldViolation when tsValue is
// not a valid RFC3339 timestamp (defense-in-depth). Returns nil otherwise.
func (v *Validator) futureViolation(field, tsValue string) *FieldViolation {
	if tsValue == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, tsValue)
	if err != nil {
		return &FieldViolation{
			Field:       field,
			Description: "timestamp is not a valid RFC3339 datetime",
		}
	}
	if t.After(v.Now().Add(time.Minute)) {
		return &FieldViolation{
			Field:       field,
			Description: "timestamp must not be more than 1 minute in the future",
		}
	}
	return nil
}

// ValidateBleedingObservation runs schema and temporal validation.
func (v *Validator) ValidateBleedingObservation(_ context.Context, obs *v1.BleedingObservation) error {
	viols := v.schemaValidate(obs)
	if len(viols) > 0 {
		return finalise(viols)
	}
	if viol := v.futureViolation("timestamp", obs.GetTimestamp().GetValue()); viol != nil {
		viols = append(viols, *viol)
	}
	return finalise(viols)
}

// ValidateSymptomObservation runs schema and temporal validation.
func (v *Validator) ValidateSymptomObservation(_ context.Context, obs *v1.SymptomObservation) error {
	viols := v.schemaValidate(obs)
	if len(viols) > 0 {
		return finalise(viols)
	}
	if viol := v.futureViolation("timestamp", obs.GetTimestamp().GetValue()); viol != nil {
		viols = append(viols, *viol)
	}
	return finalise(viols)
}

// ValidateMoodObservation runs schema and temporal validation.
func (v *Validator) ValidateMoodObservation(_ context.Context, obs *v1.MoodObservation) error {
	viols := v.schemaValidate(obs)
	if len(viols) > 0 {
		return finalise(viols)
	}
	if viol := v.futureViolation("timestamp", obs.GetTimestamp().GetValue()); viol != nil {
		viols = append(viols, *viol)
	}
	return finalise(viols)
}

// ValidateMedication runs schema validation on a Medication.
func (v *Validator) ValidateMedication(_ context.Context, med *v1.Medication) error {
	return finalise(v.schemaValidate(med))
}

// ValidateMedicationEvent runs schema, temporal, and referential-integrity validation.
// All violations are collected and returned together.
func (v *Validator) ValidateMedicationEvent(ctx context.Context, event *v1.MedicationEvent) error {
	viols := v.schemaValidate(event)
	if len(viols) > 0 {
		return finalise(viols)
	}

	// Temporal: timestamp must not be more than 1 minute in the future.
	if viol := v.futureViolation("timestamp", event.GetTimestamp().GetValue()); viol != nil {
		viols = append(viols, *viol)
	}

	// Referential integrity: medication_id must reference an existing, active
	// Medication belonging to the same user.
	if event.GetMedicationId() != "" {
		med, err := v.store.Medications().GetByID(ctx, event.GetMedicationId())
		switch {
		case errors.Is(err, storage.ErrNotFound):
			viols = append(viols, FieldViolation{
				Field:       "medication_id",
				Description: "referenced medication does not exist",
			})
		case err != nil:
			viols = append(viols, FieldViolation{
				Field:       "medication_id",
				Description: "could not verify medication: " + err.Error(),
			})
		case med.GetUserId() != event.GetUserId():
			viols = append(viols, FieldViolation{
				Field:       "medication_id",
				Description: "referenced medication belongs to a different user",
			})
		case !med.GetActive():
			viols = append(viols, FieldViolation{
				Field:       "medication_id",
				Description: "referenced medication is inactive",
			})
		}
	}

	return finalise(viols)
}

// ValidateCycle runs schema, cross-field, and cross-record validation.
// Cross-field: end_date must be ≥ start_date when both are set.
// Cross-record: the cycle must not overlap any existing cycle for the same user.
func (v *Validator) ValidateCycle(ctx context.Context, c *v1.Cycle) error {
	viols := v.schemaValidate(c)
	if len(viols) > 0 {
		return finalise(viols)
	}

	start := c.GetStartDate().GetValue()
	end := c.GetEndDate().GetValue()

	// Cross-field: end_date >= start_date.
	if start != "" && end != "" && end < start {
		viols = append(viols, FieldViolation{
			Field:       "end_date",
			Description: "end_date must be on or after start_date",
		})
	}

	// Cross-record: no overlapping cycles for the same user.
	if c.GetUserId() != "" && start != "" {
		rangeEnd := end
		if rangeEnd == "" {
			// Open-ended cycle: check against all future cycles.
			rangeEnd = "9999-12-31"
		}
		page, err := v.store.Cycles().ListByUserAndDateRange(
			ctx, c.GetUserId(), start, rangeEnd, storage.PageRequest{PageSize: 10},
		)
		if err == nil {
			for _, existing := range page.Items {
				if existing.GetId() != c.GetId() {
					viols = append(viols, FieldViolation{
						Field: "cycle",
						Description: fmt.Sprintf(
							"date range overlaps with existing cycle %s (%s\u2013%s)",
							existing.GetId(),
							existing.GetStartDate().GetValue(),
							existing.GetEndDate().GetValue(),
						),
					})
					break
				}
			}
		}
	}

	return finalise(viols)
}

// ValidateUserProfile runs schema validation. The proto annotations enforce
// required fields (tracking_focus ≥ 1, enum fields non-UNSPECIFIED). Use
// IsProfileComplete to test whether the profile allows predictions/phase estimates.
func (v *Validator) ValidateUserProfile(_ context.Context, profile *v1.UserProfile) error {
	return finalise(v.schemaValidate(profile))
}

// ValidateRequest runs protovalidate schema constraints on an RPC request
// message. This enforces buf.validate annotations on the request wrapper itself
// (e.g. min_len constraints on user_id fields) before the handler is entered.
func (v *Validator) ValidateRequest(msg proto.Message) error {
	return finalise(v.schemaValidate(msg))
}

// IsProfileComplete reports whether the profile has the fields required for
// predictions and phase estimates: biological_cycle and cycle_regularity must
// both be non-UNSPECIFIED. tracking_focus is enforced by proto schema.
func IsProfileComplete(profile *v1.UserProfile) bool {
	return profile.GetBiologicalCycle() != v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_UNSPECIFIED &&
		profile.GetCycleRegularity() != v1.CycleRegularity_CYCLE_REGULARITY_UNSPECIFIED
}
