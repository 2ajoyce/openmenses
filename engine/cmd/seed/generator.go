package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"
)

// cycleLengthForIndex returns the cycle length (in days) for cycle number i,
// applying the scenario's CycleLengthTrend on top of CycleLengthMean and adding
// Gaussian noise scaled by CycleLengthStdDev. The minimum returned value is 1.
func (g *Generator) cycleLengthForIndex(i int) int { //nolint:unused
	base := g.scenario.CycleLengthMean + g.scenario.CycleLengthTrend*float64(i)
	noise := g.rng.NormFloat64() * g.scenario.CycleLengthStdDev
	length := int(base + noise + 0.5)
	if length < 1 {
		length = 1
	}
	return length
}

// NewGeneratorWithClient creates a Generator with a Connect-RPC client
// pointing to the given base URL (e.g., "http://localhost:8080").
func NewGeneratorWithClient(g *Generator, baseURL string) (*Generator, error) { //nolint:unused
	client := openmensesv1connect.NewCycleTrackerServiceClient(
		http.DefaultClient,
		baseURL,
	)
	g.client = client
	return g, nil
}

// createProfile creates a UserProfile via RPC with reasonable defaults.
// The profile name is set to "users/default" so that the UI (which always
// queries under that parent) can see the seeded data.
func (g *Generator) createProfile(ctx context.Context) (*v1.UserProfile, error) { //nolint:unused
	profile := &v1.UserProfile{
		Name:             "users/default",
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		TrackingFocus: []v1.TrackingFocus{
			v1.TrackingFocus_TRACKING_FOCUS_BLEEDING,
			v1.TrackingFocus_TRACKING_FOCUS_SYMPTOMS,
		},
	}

	req := &v1.CreateUserProfileRequest{
		Profile: profile,
	}

	resp, err := g.client.CreateUserProfile(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	return resp.Msg.Profile, nil
}

// createBleedingEpisode creates a sequence of bleeding observations for a cycle.
// It distributes the bleed across the days according to the scenario's flow pattern.
func (g *Generator) createBleedingEpisode(ctx context.Context, userID string, cycleStartDate time.Time, bleedDuration int) error { //nolint:unused
	if bleedDuration <= 0 {
		return nil
	}

	for i := 0; i < bleedDuration; i++ {
		obsDate := cycleStartDate.AddDate(0, 0, i)

		// Get flow intensity for this day
		var flow v1.BleedingFlow
		if len(g.scenario.FlowPattern) > 0 {
			patternIdx := i % len(g.scenario.FlowPattern)
			flow = mapFlowIntensity(g.scenario.FlowPattern[patternIdx])
		} else {
			flow = v1.BleedingFlow_BLEEDING_FLOW_MEDIUM
		}

		req := &v1.CreateBleedingObservationRequest{
			Parent: userID,
			Observation: &v1.BleedingObservation{
				UserId:    userID,
				Timestamp: toDateTime(obsDate),
				Flow:      flow,
			},
		}

		_, err := g.client.CreateBleedingObservation(ctx, connect.NewRequest(req))
		if err != nil {
			return fmt.Errorf("create bleeding observation on %s: %w", obsDate.Format("2006-01-02"), err)
		}
	}

	return nil
}

// createSymptomObservations creates symptom observations for a cycle based on the scenario patterns.
func (g *Generator) createSymptomObservations(ctx context.Context, userID string, cycleStartDate time.Time) error { //nolint:unused
	for symptomType, preferredDays := range g.scenario.SymptomPatterns {
		symptomEnum := mapSymptomType(symptomType)
		for _, preferredDay := range preferredDays {
			// Add jitter: randomly offset by ±1 day
			dayOffset := preferredDay + g.rng.Intn(3) - 1 // -1, 0, or 1
			if dayOffset < 0 {
				dayOffset = 0
			}

			obsDate := cycleStartDate.AddDate(0, 0, dayOffset)

			req := &v1.CreateSymptomObservationRequest{
				Parent: userID,
				Observation: &v1.SymptomObservation{
					UserId:    userID,
					Timestamp: toDateTime(obsDate),
					Symptom:   symptomEnum,
					Severity:  v1.SymptomSeverity_SYMPTOM_SEVERITY_MILD,
				},
			}

			_, err := g.client.CreateSymptomObservation(ctx, connect.NewRequest(req))
			if err != nil {
				return fmt.Errorf("create symptom observation (%s) on %s: %w",
					symptomType, obsDate.Format("2006-01-02"), err)
			}
		}
	}

	return nil
}

// createMoodObservations creates mood observations scattered throughout the cycle.
func (g *Generator) createMoodObservations(ctx context.Context, userID string, cycleStartDate time.Time, cycleLengthDays int) error { //nolint:unused
	if !g.scenario.IncludeMood {
		return nil
	}

	// Create 2-3 random mood observations per cycle
	moodCount := 2 + g.rng.Intn(2)
	moodTypes := []v1.MoodType{
		v1.MoodType_MOOD_TYPE_CALM,
		v1.MoodType_MOOD_TYPE_HAPPY,
		v1.MoodType_MOOD_TYPE_IRRITABLE,
		v1.MoodType_MOOD_TYPE_ANXIOUS,
		v1.MoodType_MOOD_TYPE_SAD,
	}

	for i := 0; i < moodCount; i++ {
		dayOffset := g.rng.Intn(cycleLengthDays)
		obsDate := cycleStartDate.AddDate(0, 0, dayOffset)
		moodType := moodTypes[g.rng.Intn(len(moodTypes))]

		req := &v1.CreateMoodObservationRequest{
			Parent: userID,
			Observation: &v1.MoodObservation{
				UserId:    userID,
				Timestamp: toDateTime(obsDate),
				Mood:      moodType,
				Intensity: v1.MoodIntensity_MOOD_INTENSITY_MEDIUM,
			},
		}

		_, err := g.client.CreateMoodObservation(ctx, connect.NewRequest(req))
		if err != nil {
			return fmt.Errorf("create mood observation on %s: %w", obsDate.Format("2006-01-02"), err)
		}
	}

	return nil
}

// createMedicationWithEvents creates a medication and daily event records following the adherence pattern.
func (g *Generator) createMedicationWithEvents(ctx context.Context, userID string, medName string, adherenceRate float64, startDate time.Time, numDays int) error { //nolint:unused
	// Create the medication
	createMedReq := &v1.CreateMedicationRequest{
		Parent: userID,
		Medication: &v1.Medication{
			UserId:      userID,
			DisplayName: medName,
			Active:      true,
			Category:    v1.MedicationCategory_MEDICATION_CATEGORY_OTHER,
		},
	}

	medResp, err := g.client.CreateMedication(ctx, connect.NewRequest(createMedReq))
	if err != nil {
		return fmt.Errorf("create medication %s: %w", medName, err)
	}

	medID := medResp.Msg.Medication.Name

	// Create daily events with adherence pattern
	for i := 0; i < numDays; i++ {
		// Skip this day with probability (1 - adherenceRate)
		if g.rng.Float64() > adherenceRate {
			continue
		}

		eventDate := startDate.AddDate(0, 0, i)

		createEventReq := &v1.CreateMedicationEventRequest{
			Parent: userID,
			Event: &v1.MedicationEvent{
				UserId:       userID,
				MedicationId: medID,
				Timestamp:    toDateTime(eventDate),
				Status:       v1.MedicationEventStatus_MEDICATION_EVENT_STATUS_TAKEN,
			},
		}

		_, err := g.client.CreateMedicationEvent(ctx, connect.NewRequest(createEventReq))
		if err != nil {
			return fmt.Errorf("create medication event for %s on %s: %w",
				medName, eventDate.Format("2006-01-02"), err)
		}
	}

	return nil
}

// mapFlowIntensity converts the seed tool's FlowIntensity to the proto enum.
func mapFlowIntensity(fi FlowIntensity) v1.BleedingFlow { //nolint:unused
	switch fi {
	case FlowLight:
		return v1.BleedingFlow_BLEEDING_FLOW_LIGHT
	case FlowModerate:
		return v1.BleedingFlow_BLEEDING_FLOW_MEDIUM
	case FlowHeavy:
		return v1.BleedingFlow_BLEEDING_FLOW_HEAVY
	default:
		return v1.BleedingFlow_BLEEDING_FLOW_UNSPECIFIED
	}
}

// mapSymptomType converts a symptom name string to the proto enum.
func mapSymptomType(name string) v1.SymptomType { //nolint:unused
	switch name {
	case "Headache", "headache":
		return v1.SymptomType_SYMPTOM_TYPE_HEADACHE
	case "Migraine", "migraine":
		return v1.SymptomType_SYMPTOM_TYPE_MIGRAINE
	case "Cramps", "cramps":
		return v1.SymptomType_SYMPTOM_TYPE_CRAMPS
	case "Fatigue", "fatigue":
		return v1.SymptomType_SYMPTOM_TYPE_FATIGUE
	case "Bloating", "bloating":
		return v1.SymptomType_SYMPTOM_TYPE_BLOATING
	case "Breast tenderness", "breast_tenderness", "breast tenderness":
		return v1.SymptomType_SYMPTOM_TYPE_BREAST_TENDERNESS
	case "Nausea", "nausea":
		return v1.SymptomType_SYMPTOM_TYPE_NAUSEA
	case "Acne", "acne":
		return v1.SymptomType_SYMPTOM_TYPE_ACNE
	case "Back pain", "back_pain", "back pain":
		return v1.SymptomType_SYMPTOM_TYPE_BACK_PAIN
	default:
		return v1.SymptomType_SYMPTOM_TYPE_UNSPECIFIED
	}
}

// toDateTime converts a Go time.Time to a DateTime proto message.
func toDateTime(t time.Time) *v1.DateTime { //nolint:unused
	return &v1.DateTime{
		Value: t.Format("2006-01-02T15:04:05Z"),
	}
}

// queryCycles lists all cycles for the user.
func (g *Generator) queryCycles(ctx context.Context) ([]*v1.Cycle, error) {
	req := &v1.ListCyclesRequest{
		Parent: g.userID,
		Pagination: &v1.PaginationRequest{
			PageSize: 500, // Maximum page size
		},
	}

	resp, err := g.client.ListCycles(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("list cycles: %w", err)
	}

	return resp.Msg.Cycles, nil
}

// queryInsights lists all insights for the user.
func (g *Generator) queryInsights(ctx context.Context) ([]*v1.Insight, error) {
	req := &v1.ListInsightsRequest{
		Parent: g.userID,
		Pagination: &v1.PaginationRequest{
			PageSize: 500, // Maximum page size
		},
	}

	resp, err := g.client.ListInsights(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("list insights: %w", err)
	}

	return resp.Msg.Insights, nil
}

// queryPredictions lists all predictions for the user.
func (g *Generator) queryPredictions(ctx context.Context) ([]*v1.Prediction, error) {
	req := &v1.ListPredictionsRequest{
		Parent: g.userID,
		Pagination: &v1.PaginationRequest{
			PageSize: 500, // Maximum page size
		},
	}

	resp, err := g.client.ListPredictions(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("list predictions: %w", err)
	}

	return resp.Msg.Predictions, nil
}
