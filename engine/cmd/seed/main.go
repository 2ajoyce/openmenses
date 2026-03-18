// Command seed generates realistic test data by calling the engine's service layer
// via Connect-RPC. This makes it easy for developers to populate a local database
// with enough data to manually verify features like insights, predictions, and
// cycle trends without hand-creating dozens of observations.
//
// Usage:
//
//	seed [--scenario regular-12] [--cycles 12] [--seed 42] [--user-id test-user] [--db openmenses.db] [--list-scenarios]
//
// Scenarios include:
//   - regular-12: 12 cycles, mean length 28 days, consistent symptom patterns
//   - irregular: 8 cycles with variable lengths, tests irregular pattern detection
//   - shortening: 10 cycles with shortening trend
//   - medication-gaps: 6 cycles with medication at ~60% adherence
//   - minimal: 3 cycles, bare minimum data for threshold testing
//
// Flags:
//
//	--scenario:        Named scenario to use (default: regular-12)
//	--cycles:          Override cycle count for the scenario
//	--seed:            Integer seed for PRNG (default: 42); same seed produces identical data
//	--user-id:         User ID to populate (default: test-user)
//	--db:              SQLite database path (default: openmenses.db)
//	--list-scenarios:  Print available scenarios and exit
package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/2ajoyce/openmenses/engine/pkg/openmenses"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"
)

// FlowIntensity represents the intensity of menstrual flow on a given day.
type FlowIntensity int

const (
	FlowLight FlowIntensity = iota
	FlowModerate
	FlowHeavy
)

// Scenario defines the parameters for generating a test dataset.
type Scenario struct {
	// Name is the human-readable scenario name.
	Name string

	// Description explains what patterns this scenario produces.
	Description string

	// CycleCount is the number of complete cycles to generate.
	CycleCount int

	// CycleLengthMean is the average cycle length in days.
	CycleLengthMean float64

	// CycleLengthStdDev is the standard deviation for cycle lengths.
	CycleLengthStdDev float64

	// CycleLengthTrend is the linear slope in days-per-cycle applied on top of CycleLengthMean.
	// Positive values lengthen cycles over time; negative values shorten them.
	// Example: -0.667 over 10 cycles shifts from mean+3 to mean-3 (a 6-day decrease).
	// Zero (default) means no trend — cycles vary only by CycleLengthStdDev.
	CycleLengthTrend float64

	// BleedDurationMean is the average bleed duration in days.
	BleedDurationMean float64

	// BleedDurationStdDev is the standard deviation for bleed duration.
	BleedDurationStdDev float64

	// FlowPattern defines the intensity pattern across bleed days.
	// Example: [Light, Moderate, Heavy, Moderate, Light]
	FlowPattern []FlowIntensity

	// SymptomPatterns maps symptom type names to their preferred cycle days.
	// Example: {"Headache": []int{12}, "Cramps": []int{1, 2, 3}}
	SymptomPatterns map[string][]int

	// MedicationNames is the list of medications to create and track.
	MedicationNames []string

	// MedicationAdherence maps medication name to adherence rate (0.0 to 1.0).
	// Example: {"Ibuprofen": 0.95}
	MedicationAdherence map[string]float64

	// IncludeMood whether to generate mood observations.
	IncludeMood bool
}

// Generator holds PRNG state, scenario config, and a Connect-RPC client
// for generating and persisting test data.
type Generator struct {
	// rng is the pseudo-random number generator seeded with a fixed seed.
	rng *rand.Rand

	// scenario is the configured scenario to generate.
	scenario *Scenario

	// userID is the user identifier for all generated observations.
	userID string

	// client is a Connect-RPC client pointing to the local listener.
	// This field is populated after starting the engine.
	client openmensesv1connect.CycleTrackerServiceClient

	// stats tracks counts of generated data
	stats struct {
		bleedingObs    int
		symptomObs     int
		moodObs        int
		medicationObs  int
		cyclesDetected int
		insights       int
		predictions    int
	}
}

// scenarioRegistry maps scenario names to their configurations.
var scenarioRegistry = map[string]*Scenario{
	"regular-12":      regularScenario(),
	"irregular":       irregularScenario(),
	"shortening":      shorteningScenario(),
	"medication-gaps": medicationGapsScenario(),
	"minimal":         minimalScenario(),
}

func main() {
	scenarioName := flag.String("scenario", "regular-12", "Named scenario to use")
	cycleOverride := flag.Int("cycles", 0, "Override cycle count for the scenario (0 = use scenario default)")
	seed := flag.Int64("seed", 42, "Integer seed for PRNG (same seed produces identical data)")
	userID := flag.String("user-id", "test-user", "User ID to populate")
	dbPath := flag.String("db", "openmenses.db", "SQLite database path")
	listScenarios := flag.Bool("list-scenarios", false, "Print available scenarios and exit")

	flag.Parse()

	// Handle --list-scenarios
	if *listScenarios {
		if len(scenarioRegistry) == 0 {
			fmt.Println("No scenarios registered yet (defined in Step 3)")
		} else {
			fmt.Println("Available scenarios:")
			for name, scenario := range scenarioRegistry {
				fmt.Printf("  %s: %s\n", name, scenario.Description)
			}
		}
		return
	}

	// Look up the scenario
	scenario, ok := scenarioRegistry[*scenarioName]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: scenario %q not found\n", *scenarioName)
		if len(scenarioRegistry) > 0 {
			fmt.Fprintf(os.Stderr, "available scenarios: ")
			for name := range scenarioRegistry {
				fmt.Fprintf(os.Stderr, "%s ", name)
			}
			fmt.Fprintf(os.Stderr, "\n")
		} else {
			fmt.Fprintf(os.Stderr, "no scenarios registered yet (defined in Step 3)\n")
		}
		os.Exit(1)
	}

	// Apply cycle override if provided
	if *cycleOverride > 0 {
		scenario.CycleCount = *cycleOverride
	}

	// Create the generator with seeded PRNG
	rng := rand.New(rand.NewSource(*seed))
	generator := &Generator{
		rng:      rng,
		scenario: scenario,
		userID:   *userID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start the engine
	fmt.Printf("Starting engine with %s backend...\n", *dbPath)
	engine, err := openmenses.NewEngine(ctx, openmenses.WithSQLite(*dbPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to start engine: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := engine.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close engine: %v\n", err)
		}
	}()

	// Start a local HTTP listener on a random port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create listener: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	baseURL := "http://" + addr

	// Mount the engine handler on an HTTP mux
	mux := http.NewServeMux()
	path, handler := engine.Handler()
	mux.Handle(path, handler)

	// Start the HTTP server in a goroutine
	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "warning: server error: %v\n", err)
		}
	}()
	defer server.Close()

	// Create a Connect-RPC client pointing to the local listener
	client := openmensesv1connect.NewCycleTrackerServiceClient(http.DefaultClient, baseURL)
	generator.client = client

	fmt.Printf("Engine started at %s\n", baseURL)
	fmt.Printf("\nGenerating scenario: %s\n", scenario.Description)
	fmt.Printf("  cycles: %d\n", scenario.CycleCount)
	fmt.Printf("  seed: %d\n", *seed)
	fmt.Printf("  user-id: %s\n", *userID)
	fmt.Printf("\n")

	// Create the user profile
	if err := generator.generateAll(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: generation failed: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("\nGeneration complete. Summary:\n")
	fmt.Printf("  Bleeding observations: %d\n", generator.stats.bleedingObs)
	fmt.Printf("  Symptom observations: %d\n", generator.stats.symptomObs)
	fmt.Printf("  Mood observations: %d\n", generator.stats.moodObs)
	fmt.Printf("  Medication events: %d\n", generator.stats.medicationObs)
	fmt.Printf("  Cycles detected: %d\n", generator.stats.cyclesDetected)
	fmt.Printf("  Insights generated: %d\n", generator.stats.insights)
	fmt.Printf("  Predictions generated: %d\n", generator.stats.predictions)
}

// generateAll runs the full generation pipeline: create profile, then generate cycles
// chronologically with bleeding, symptoms, moods, and medication events.
func (g *Generator) generateAll(ctx context.Context) error {
	// Create user profile
	fmt.Println("Creating user profile...")
	profile, err := g.createProfile(ctx)
	if err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	// Use the server-assigned profile name as the userID for subsequent observations
	g.userID = profile.GetName()

	// Determine the first cycle start date (some time in the past)
	// Use a buffer larger than total cycle days to ensure all dates are in the past
	now := time.Now()
	totalDays := g.scenario.CycleCount * 35 // Conservative estimate with 35 days per cycle
	cycleStartDate := now.AddDate(0, 0, -totalDays)

	// Generate cycles chronologically
	for cycleIdx := 0; cycleIdx < g.scenario.CycleCount; cycleIdx++ {
		bleedDuration := int(g.scenario.BleedDurationMean + g.rng.NormFloat64()*g.scenario.BleedDurationStdDev + 0.5)
		if bleedDuration < 1 {
			bleedDuration = 1
		}

		cycleLengthDays := g.cycleLengthForIndex(cycleIdx)

		fmt.Printf("  Cycle %d/%d (start: %s, length: %d days, bleed: %d days)\n",
			cycleIdx+1, g.scenario.CycleCount, cycleStartDate.Format("2006-01-02"), cycleLengthDays, bleedDuration)

		// Create bleeding observations
		if err := g.createBleedingEpisode(ctx, g.userID, cycleStartDate, bleedDuration); err != nil {
			return fmt.Errorf("create bleeding for cycle %d: %w", cycleIdx, err)
		}
		g.stats.bleedingObs += bleedDuration

		// Create symptom observations
		if err := g.createSymptomObservations(ctx, g.userID, cycleStartDate); err != nil {
			return fmt.Errorf("create symptoms for cycle %d: %w", cycleIdx, err)
		}
		// Count symptom observations (rough estimate: num symptom types * num preferred days)
		for _, days := range g.scenario.SymptomPatterns {
			g.stats.symptomObs += len(days)
		}

		// Create mood observations
		if err := g.createMoodObservations(ctx, g.userID, cycleStartDate, cycleLengthDays); err != nil {
			return fmt.Errorf("create moods for cycle %d: %w", cycleIdx, err)
		}
		if g.scenario.IncludeMood {
			g.stats.moodObs += 2 + g.rng.Intn(2) // Same logic as createMoodObservations
		}

		// Create medication events (only on the first cycle to avoid duplicates)
		if cycleIdx == 0 {
			for medName, adherence := range g.scenario.MedicationAdherence {
				totalDays := cycleLengthDays * g.scenario.CycleCount
				if err := g.createMedicationWithEvents(ctx, g.userID, medName, adherence, cycleStartDate, totalDays); err != nil {
					return fmt.Errorf("create medication %s: %w", medName, err)
				}
				// Count events (approximate: adherence rate * total days)
				g.stats.medicationObs += int(adherence * float64(totalDays))
			}
		}

		// Advance to next cycle start
		cycleStartDate = cycleStartDate.AddDate(0, 0, cycleLengthDays)
	}

	// Query cycles to verify detection
	fmt.Println("Querying cycles...")
	cycles, err := g.queryCycles(ctx)
	if err != nil {
		return fmt.Errorf("query cycles: %w", err)
	}
	g.stats.cyclesDetected = len(cycles)

	// Query insights
	fmt.Println("Querying insights...")
	insights, err := g.queryInsights(ctx)
	if err != nil {
		return fmt.Errorf("query insights: %w", err)
	}
	g.stats.insights = len(insights)

	// Query predictions
	fmt.Println("Querying predictions...")
	predictions, err := g.queryPredictions(ctx)
	if err != nil {
		return fmt.Errorf("query predictions: %w", err)
	}
	g.stats.predictions = len(predictions)

	return nil
}
