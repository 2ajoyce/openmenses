package openmenses_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/2ajoyce/openmenses/engine/pkg/openmenses"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"
)

// newTestEngine creates an in-memory engine and a test HTTP server that serves
// its Connect-RPC handler.  The returned client is pre-configured to talk to the server.
func newTestEngine(t *testing.T) (*openmenses.Engine, openmensesv1connect.CycleTrackerServiceClient) {
	t.Helper()
	ctx := context.Background()

	eng, err := openmenses.NewEngine(ctx, openmenses.WithInMemory())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	t.Cleanup(func() { eng.Close() }) //nolint:errcheck

	mux := http.NewServeMux()
	path, handler := eng.Handler()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := openmensesv1connect.NewCycleTrackerServiceClient(
		srv.Client(),
		srv.URL,
	)
	return eng, client
}

// TestIntegration_CreateProfileLogObservationListTimeline exercises the core
// happy path: create a user profile, log a bleeding observation, then verify
// the observation appears in the timeline.
func TestIntegration_CreateProfileLogObservationListTimeline(t *testing.T) {
	ctx := context.Background()
	_, client := newTestEngine(t)

	const userID = "user-integration-test"

	// 1. Create user profile.
	profile := &v1.UserProfile{
		Name:             userID,
		BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
		CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
		TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_BLEEDING},
	}

	upsertResp, err := client.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{
		Profile: profile,
	}))
	if err != nil {
		t.Fatalf("CreateUserProfile: %v", err)
	}
	if upsertResp.Msg.GetProfile().GetName() != userID {
		t.Errorf("upserted profile name = %q, want %q", upsertResp.Msg.GetProfile().GetName(), userID)
	}

	// 2. Log a bleeding observation on 2026-01-01.
	const obsDate = "2026-01-01"
	createResp, err := client.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
		Parent: userID,
		Observation: &v1.BleedingObservation{
			Timestamp: &v1.DateTime{Value: obsDate + "T10:00:00Z"},
			Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
		},
	}))
	if err != nil {
		t.Fatalf("CreateBleedingObservation: %v", err)
	}
	obsID := createResp.Msg.GetObservation().GetName()
	if obsID == "" {
		t.Error("created observation has empty name")
	}

	// 3. List the timeline for that day and verify the observation is present.
	tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
		Parent: userID,
		Range: &v1.DateRange{
			Start: &v1.LocalDate{Value: obsDate},
			End:   &v1.LocalDate{Value: obsDate},
		},
	}))
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}

	records := tlResp.Msg.GetRecords()
	if len(records) == 0 {
		t.Fatal("timeline returned no records, expected at least one bleeding observation")
	}

	// Find the bleeding record by name.
	var found bool
	for _, r := range records {
		if b := r.GetBleedingObservation(); b != nil && b.GetName() == obsID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("timeline did not contain bleeding observation %q; got %d record(s)", obsID, len(records))
	}
}

// TestIntegration_InMemoryEngineHandler verifies that NewEngine returns a
// functional handler (non-nil path and handler).
func TestIntegration_InMemoryEngineHandler(t *testing.T) {
	ctx := context.Background()
	eng, err := openmenses.NewEngine(ctx, openmenses.WithInMemory())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer eng.Close() //nolint:errcheck

	path, handler := eng.Handler()
	if path == "" {
		t.Error("Handler() returned empty path")
	}
	if handler == nil {
		t.Error("Handler() returned nil handler")
	}
}

// TestIntegration_SQLiteEngine verifies the SQLite backend initialises and
// accepts profile upserts.
func TestIntegration_SQLiteEngine(t *testing.T) {
	ctx := context.Background()

	// Use ":memory:" so no file is written during tests.
	eng, err := openmenses.NewEngine(ctx, openmenses.WithSQLite(":memory:"))
	if err != nil {
		t.Fatalf("NewEngine (sqlite): %v", err)
	}
	defer eng.Close() //nolint:errcheck

	mux := http.NewServeMux()
	path, handler := eng.Handler()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := openmensesv1connect.NewCycleTrackerServiceClient(srv.Client(), srv.URL)

	const userID = "user-sqlite-test"
	resp, err := client.CreateUserProfile(ctx, connect.NewRequest(&v1.CreateUserProfileRequest{
		Profile: &v1.UserProfile{
			Name:             userID,
			BiologicalCycle:  v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
			Contraception:    v1.ContraceptionType_CONTRACEPTION_TYPE_NONE,
			CycleRegularity:  v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
			ReproductiveGoal: v1.ReproductiveGoal_REPRODUCTIVE_GOAL_PREGNANCY_IRRELEVANT,
			TrackingFocus:    []v1.TrackingFocus{v1.TrackingFocus_TRACKING_FOCUS_BLEEDING},
		},
	}))
	if err != nil {
		t.Fatalf("CreateUserProfile (sqlite): %v", err)
	}
	if resp.Msg.GetProfile().GetName() != userID {
		t.Errorf("profile name = %q, want %q", resp.Msg.GetProfile().GetName(), userID)
	}
}
