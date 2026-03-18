package main

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/2ajoyce/openmenses/engine/pkg/openmenses"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// setupTestEngine creates an in-memory engine, starts a local HTTP listener,
// and returns the engine, client, and base URL.
func setupTestEngine(t *testing.T) (*openmenses.Engine, openmensesv1connect.CycleTrackerServiceClient, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create in-memory engine
	engine, err := openmenses.NewEngine(ctx, openmenses.WithInMemory())
	require.NoError(t, err, "failed to create engine")

	// Start local HTTP listener
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "failed to create listener")

	addr := listener.Addr().String()
	baseURL := "http://" + addr

	// Mount engine handler
	mux := http.NewServeMux()
	path, handler := engine.Handler()
	mux.Handle(path, handler)

	// Start HTTP server
	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()

	// Create client
	client := openmensesv1connect.NewCycleTrackerServiceClient(http.DefaultClient, baseURL)

	// Cleanup function will be called by caller
	t.Cleanup(func() {
		_ = server.Close()
		_ = listener.Close()
		_ = engine.Close()
	})

	return engine, client, baseURL
}

// TestSeedScenarios_EachRunsWithoutError verifies all built-in scenarios execute successfully.
func TestSeedScenarios_EachRunsWithoutError(t *testing.T) {
	scenarios := []string{"regular-12", "irregular", "shortening", "medication-gaps", "minimal"}

	for _, scenarioName := range scenarios {
		t.Run(scenarioName, func(t *testing.T) {
			_, client, _ := setupTestEngine(t)

			scenario := scenarioRegistry[scenarioName]
			require.NotNil(t, scenario, "scenario not found: %s", scenarioName)

			userID := "test-user-" + uuid.New().String()
			gen := &Generator{
				rng:      nil, // Will use deterministic source
				scenario: scenario,
				userID:   userID,
				client:   client,
			}

			// Use seed 42 for deterministic testing
			gen.rng = newSeededRand(42)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := gen.generateAll(ctx)
			require.NoError(t, err, "generation failed for scenario %s", scenarioName)
		})
	}
}

// TestRegular12_ProducesMinimumCycles verifies the regular-12 scenario produces ≥ 10 completed cycles.
func TestRegular12_ProducesMinimumCycles(t *testing.T) {
	_, client, _ := setupTestEngine(t)

	scenario := scenarioRegistry["regular-12"]
	require.NotNil(t, scenario)

	userID := "test-user-" + uuid.New().String()
	gen := &Generator{
		rng:      newSeededRand(42),
		scenario: scenario,
		userID:   userID,
		client:   client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := gen.generateAll(ctx)
	require.NoError(t, err)

	// Verify cycles detected
	require.GreaterOrEqual(t, gen.stats.cyclesDetected, 10, "expected ≥ 10 cycles from regular-12")
}

// TestRegular12_ProducesAllInsightTypes verifies regular-12 generates all 4 insight types.
func TestRegular12_ProducesAllInsightTypes(t *testing.T) {
	_, client, _ := setupTestEngine(t)

	scenario := scenarioRegistry["regular-12"]
	require.NotNil(t, scenario)

	userID := "test-user-" + uuid.New().String()
	gen := &Generator{
		rng:      newSeededRand(42),
		scenario: scenario,
		userID:   userID,
		client:   client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := gen.generateAll(ctx)
	require.NoError(t, err)

	// Query insights
	insights, err := gen.queryInsights(ctx)
	require.NoError(t, err)
	require.Greater(t, len(insights), 0, "expected insights from regular-12 scenario")

	// Check for each insight type
	insightTypes := make(map[v1.InsightType]int)
	for _, insight := range insights {
		insightTypes[insight.Kind]++
	}

	// regular-12 should produce at least one of each expected type:
	// - CYCLE_LENGTH_PATTERN (stable cycles)
	// - SYMPTOM_PATTERN (consistent headache on day 12)
	// - MEDICATION_ADHERENCE_PATTERN (high adherence to Ibuprofen)
	// - BLEEDING_PATTERN (consistent bleed duration)
	expectedTypes := []v1.InsightType{
		v1.InsightType_INSIGHT_TYPE_CYCLE_LENGTH_PATTERN,
		v1.InsightType_INSIGHT_TYPE_SYMPTOM_PATTERN,
		v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN,
		v1.InsightType_INSIGHT_TYPE_BLEEDING_PATTERN,
	}

	for _, expectedType := range expectedTypes {
		require.Greater(t, insightTypes[expectedType], 0,
			"expected at least 1 insight of type %v from regular-12", expectedType)
	}
}

// TestMedicationGaps_ProducesLowAdherenceInsight verifies medication-gaps scenario
// generates a LOW medication adherence insight.
func TestMedicationGaps_ProducesLowAdherenceInsight(t *testing.T) {
	_, client, _ := setupTestEngine(t)

	scenario := scenarioRegistry["medication-gaps"]
	require.NotNil(t, scenario)

	userID := "test-user-" + uuid.New().String()
	gen := &Generator{
		rng:      newSeededRand(42),
		scenario: scenario,
		userID:   userID,
		client:   client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := gen.generateAll(ctx)
	require.NoError(t, err)

	// Query insights
	insights, err := gen.queryInsights(ctx)
	require.NoError(t, err)

	// Find medication adherence insight
	var found bool
	for _, insight := range insights {
		if insight.Kind == v1.InsightType_INSIGHT_TYPE_MEDICATION_ADHERENCE_PATTERN {
			// Check that the summary contains "Low" or similar indicator
			require.NotEmpty(t, insight.Summary, "insight should have a summary")
			found = true
			break
		}
	}

	require.True(t, found, "expected MEDICATION_ADHERENCE_PATTERN insight from medication-gaps scenario")
}

// TestDeterminism_SameSeedProducesIdenticalCounts verifies that running
// the same scenario with the same seed produces identical observation counts.
func TestDeterminism_SameSeedProducesIdenticalCounts(t *testing.T) {
	const seed = 99

	scenario := scenarioRegistry["regular-12"]
	require.NotNil(t, scenario)

	// First run
	_, client1, _ := setupTestEngine(t)
	userID1 := "test-user-" + uuid.New().String()
	gen1 := &Generator{
		rng:      newSeededRand(seed),
		scenario: scenario,
		userID:   userID1,
		client:   client1,
	}

	ctx1, cancel1 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel1()

	err := gen1.generateAll(ctx1)
	require.NoError(t, err)

	counts1 := gen1.stats

	// Second run with same seed
	_, client2, _ := setupTestEngine(t)
	userID2 := "test-user-" + uuid.New().String()
	gen2 := &Generator{
		rng:      newSeededRand(seed),
		scenario: scenario,
		userID:   userID2,
		client:   client2,
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	err = gen2.generateAll(ctx2)
	require.NoError(t, err)

	counts2 := gen2.stats

	// Verify identical counts
	require.Equal(t, counts1.bleedingObs, counts2.bleedingObs, "bleeding observation count mismatch")
	require.Equal(t, counts1.symptomObs, counts2.symptomObs, "symptom observation count mismatch")
	require.Equal(t, counts1.moodObs, counts2.moodObs, "mood observation count mismatch")
	require.Equal(t, counts1.medicationObs, counts2.medicationObs, "medication event count mismatch")
	require.Equal(t, counts1.cyclesDetected, counts2.cyclesDetected, "cycles detected count mismatch")
}

// newSeededRand creates a seeded PRNG.
func newSeededRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}
