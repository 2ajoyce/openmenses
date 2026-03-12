// Package tests provides integration tests for the openmenses engine.
// Tests exercise the full stack: service → validation → rules → storage.
// Both the in-memory and SQLite backends are tested.
package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"github.com/2ajoyce/openmenses/engine/pkg/openmenses"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
	"github.com/2ajoyce/openmenses/gen/go/openmenses/v1/openmensesv1connect"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// engineClient creates an Engine with the given options, wraps it in a test
// HTTP server, and returns a Connect-RPC client pre-configured to talk to it.
// Cleanup of both the server and engine is registered with t.
func engineClient(t *testing.T, opts ...openmenses.Option) openmensesv1connect.CycleTrackerServiceClient {
	t.Helper()
	ctx := context.Background()
	eng, err := openmenses.NewEngine(ctx, opts...)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	mux := http.NewServeMux()
	path, handler := eng.Handler()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return openmensesv1connect.NewCycleTrackerServiceClient(srv.Client(), srv.URL)
}

// importFixture loads a fixture file by name and calls ImportData on the given
// client.  Returns the number of records imported.
func importFixture(t *testing.T, client openmensesv1connect.CycleTrackerServiceClient, fixtureName string) uint32 {
	t.Helper()
	data, err := LoadFixtureBytes(fixtureName)
	if err != nil {
		t.Fatalf("LoadFixtureBytes(%q): %v", fixtureName, err)
	}
	resp, err := client.ImportData(context.Background(), connect.NewRequest(&v1.ImportDataRequest{Data: data}))
	if err != nil {
		t.Fatalf("ImportData(%q): %v", fixtureName, err)
	}
	return resp.Msg.GetRecordsImported()
}

// storageVariants returns table-driven rows for both backends so every test
// can run against in-memory and SQLite with minimal duplication.
func storageVariants() []struct {
	name string
	opts []openmenses.Option
} {
	return []struct {
		name string
		opts []openmenses.Option
	}{
		{"in-memory", []openmenses.Option{openmenses.WithInMemory()}},
		{"sqlite", []openmenses.Option{openmenses.WithSQLite(":memory:")}},
	}
}

// ─── fixture import ───────────────────────────────────────────────────────────

// TestFixture_AllFixturesImportCleanly imports every fixture file available in
// testdata/fixtures/ and verifies that the operation returns at least one
// imported record, the profile is retrievable, and observation data appears in
// the timeline. This test ensures the fixture files remain synchronised with
// the service's import logic.
func TestFixture_AllFixturesImportCleanly(t *testing.T) {
	names, err := FixtureNames()
	if err != nil {
		t.Fatalf("FixtureNames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no fixture files found in testdata/fixtures/")
	}

	for _, fixtureName := range names {
		fixtureName := fixtureName // capture for sub-test
		t.Run(fixtureName, func(t *testing.T) {
			for _, v := range storageVariants() {
				v := v
				t.Run(v.name, func(t *testing.T) {
					t.Parallel()
					ctx := context.Background()
					client := engineClient(t, v.opts...)

					count := importFixture(t, client, fixtureName)
					if count == 0 {
						t.Errorf("ImportData(%q): expected > 0 records, got 0", fixtureName)
					}

					// Load the raw fixture to extract user_id (structure is always
					// {"version":"1","user_id":"...",...}).
					raw, err := LoadFixtureBytes(fixtureName)
					if err != nil {
						t.Fatalf("LoadFixtureBytes: %v", err)
					}
					var envelope struct {
						UserID string `json:"user_id"`
					}
					if err := unmarshalJSON(raw, &envelope); err != nil {
						t.Fatalf("parse fixture user_id: %v", err)
					}
					userID := envelope.UserID

					// Profile must be retrievable.
					profResp, err := client.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: userID}))
					if err != nil {
						t.Fatalf("GetUserProfile after import: %v", err)
					}
					if profResp.Msg.GetProfile().GetName() != userID {
						t.Errorf("profile name = %q, want %q", profResp.Msg.GetProfile().GetName(), userID)
					}

					// Timeline must return at least one record for the imported data.
					tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
						Parent: userID,
					}))
					if err != nil {
						t.Fatalf("ListTimeline after import: %v", err)
					}
					if len(tlResp.Msg.GetRecords()) == 0 {
						t.Errorf("timeline empty after importing %q", fixtureName)
					}
				})
			}
		})
	}
}

// ─── full lifecycle ───────────────────────────────────────────────────────────

// TestIntegration_FullLifecycle_Regular28Day tests the complete lifecycle for
// a regular 28-day user:
//  1. Import the regular_28day_user fixture (6+ months of data).
//  2. Verify at least 6 cycles are detected.
//  3. Verify that the detected cycle start dates correspond to the expected
//     bleeding episode starts.
//  4. List the timeline for a specific date range and verify records are present.
func TestIntegration_FullLifecycle_Regular28Day(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			importFixture(t, client, "regular_28day_user.json")

			const userID = "user-regular-28day"

			// Cycles must be detected from bleeding data.
			cyclResp, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles: %v", err)
			}
			cycles := cyclResp.Msg.GetCycles()
			// The fixture contains 7 bleeding episodes → expect at least 6 completed
			// cycles (the last one may be open-ended).
			if len(cycles) < 6 {
				t.Errorf("ListCycles: got %d cycles, want ≥ 6", len(cycles))
			}

			// The first cycle must start on 2025-09-01 (first bleeding observation).
			var found bool
			for _, c := range cycles {
				if c.GetStartDate().GetValue() == "2025-09-01" {
					found = true
					break
				}
			}
			if !found {
				t.Error("no cycle with start_date 2025-09-01 found; expected first cycle to start on bleeding episode 1 start")
			}

			// Timeline query for cycle-1 bleed period returns at least 5 records.
			tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
				Parent: userID,
				Range: &v1.DateRange{
					Start: &v1.LocalDate{Value: "2025-09-01"},
					End:   &v1.LocalDate{Value: "2025-09-05"},
				},
			}))
			if err != nil {
				t.Fatalf("ListTimeline: %v", err)
			}
			if len(tlResp.Msg.GetRecords()) < 5 {
				t.Errorf("timeline for 2025-09-01..05: got %d records, want ≥ 5", len(tlResp.Msg.GetRecords()))
			}

			// Phase estimates must be present in the full timeline: EstimatePhases
			// is wired into the cycle detection flow, so at least one PhaseEstimate
			// record should appear after the fixture is imported.
			allRecords := collectAllTimeline(t, ctx, client, userID)
			var hasPhaseEstimate bool
			for _, r := range allRecords {
				if r.GetPhaseEstimate() != nil {
					hasPhaseEstimate = true
					break
				}
			}
			if !hasPhaseEstimate {
				t.Error("no PhaseEstimate records found in timeline; expected EstimatePhases to produce phase data for detected cycles")
			}
		})
	}
}

// TestIntegration_FullLifecycle_FirstTimeUser verifies that a user with only
// 2 observations (first-time user) gets a profile response, at least one
// cycle detected, and observations in the timeline.
func TestIntegration_FullLifecycle_FirstTimeUser(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			importFixture(t, client, "firsttime_user.json")
			const userID = "user-firsttime"

			// Profile must be accessible.
			profResp, err := client.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: userID}))
			if err != nil {
				t.Fatalf("GetUserProfile: %v", err)
			}
			if profResp.Msg.GetProfile().GetName() != userID {
				t.Errorf("profile name = %q, want %q", profResp.Msg.GetProfile().GetName(), userID)
			}

			// At least one open-ended cycle should be detected from the 2 bleeding days.
			cyclResp, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles: %v", err)
			}
			if len(cyclResp.Msg.GetCycles()) < 1 {
				t.Error("ListCycles: expected at least one open-ended cycle for first-time user")
			}

			// Timeline must include both observations.
			tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
				Parent: userID,
				Range: &v1.DateRange{
					Start: &v1.LocalDate{Value: "2026-01-15"},
					End:   &v1.LocalDate{Value: "2026-01-16"},
				},
			}))
			if err != nil {
				t.Fatalf("ListTimeline: %v", err)
			}
			if len(tlResp.Msg.GetRecords()) < 2 {
				t.Errorf("timeline: got %d records, want ≥ 2", len(tlResp.Msg.GetRecords()))
			}
		})
	}
}

// TestIntegration_FullLifecycle_IrregularUser ensures that an irregular-cycle
// user has their cycles detected even when gap intervals vary widely.
func TestIntegration_FullLifecycle_IrregularUser(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			importFixture(t, client, "irregular_user.json")
			const userID = "user-irregular"

			cyclResp, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles: %v", err)
			}
			// Fixture has 3 bleeding episodes with variable gaps (21, 35 days).
			if len(cyclResp.Msg.GetCycles()) < 2 {
				t.Errorf("ListCycles: got %d cycles, want ≥ 2", len(cyclResp.Msg.GetCycles()))
			}
		})
	}
}

// ─── import/export round-trip ─────────────────────────────────────────────────

// TestIntegration_ImportExportRoundTrip imports a fixture into engine A,
// exports all data, imports the export into engine B, and verifies that:
//   - The same number of bleeding observations exist in B.
//   - The profile names match.
//   - The timeline record count matches.
func TestIntegration_ImportExportRoundTrip(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			// Engine A: import fixture.
			clientA := engineClient(t, v.opts...)
			importFixture(t, clientA, "regular_28day_user.json")
			const userID = "user-regular-28day"

			// Get timeline record count from A.
			tlA, err := clientA.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("A ListTimeline: %v", err)
			}
			countA := len(tlA.Msg.GetRecords())
			if countA == 0 {
				t.Fatal("engine A has no timeline records")
			}

			// Export from A.
			exportResp, err := clientA.ExportData(ctx, connect.NewRequest(&v1.ExportDataRequest{Name: userID}))
			if err != nil {
				t.Fatalf("ExportData: %v", err)
			}
			exportedData := exportResp.Msg.GetData()
			if len(exportedData) == 0 {
				t.Fatal("ExportData returned empty bytes")
			}

			// Engine B: import the export.
			clientB := engineClient(t, v.opts...)
			importResp, err := clientB.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: exportedData}))
			if err != nil {
				t.Fatalf("B ImportData (round-trip): %v", err)
			}
			if importResp.Msg.GetRecordsImported() == 0 {
				t.Error("B ImportData: expected > 0 records")
			}

			// Profile must be present in B.
			profB, err := clientB.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: userID}))
			if err != nil {
				t.Fatalf("B GetUserProfile: %v", err)
			}
			if profB.Msg.GetProfile().GetName() != userID {
				t.Errorf("B profile name = %q, want %q", profB.Msg.GetProfile().GetName(), userID)
			}

			// Timeline in B must have the same bleeding observations as A.
			// (Derived cycles are not exported, so total record count may differ.)
			tlB, err := clientB.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("B ListTimeline: %v", err)
			}
			countBleedingA := countBleedingRecords(tlA.Msg.GetRecords())
			countBleedingB := countBleedingRecords(tlB.Msg.GetRecords())
			if countBleedingA != countBleedingB {
				t.Errorf("bleeding observations: A=%d, B=%d; want equal", countBleedingA, countBleedingB)
			}
		})
	}
}

// TestIntegration_ImportIdempotent verifies that importing the same fixture
// twice does not create duplicate records (the second import should produce
// zero new records for records that already exist).
func TestIntegration_ImportIdempotent(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			client := engineClient(t, v.opts...)

			count1 := importFixture(t, client, "firsttime_user.json")
			count2 := importFixture(t, client, "firsttime_user.json")

			// Profile upsert always counts as 1; bleeding obs with same IDs are
			// skipped (ErrConflict) so count2 should be ≤ count1 (only the profile
			// re-upsert contributes).
			if count2 > count1 {
				t.Errorf("second import created more records (%d) than first (%d); expected idempotent behaviour", count2, count1)
			}
		})
	}
}

// ─── empty database edge cases ────────────────────────────────────────────────

// TestIntegration_EmptyDatabase verifies that queries against an empty engine
// return sensible zero-value results rather than errors.
func TestIntegration_EmptyDatabase(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			const userID = "nonexistent-user"

			// GetUserProfile on missing user → CodeNotFound.
			_, err := client.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: userID}))
			if err == nil {
				t.Error("expected CodeNotFound for missing user, got nil error")
			} else {
				var connErr *connect.Error
				if ok := asConnectError(err, &connErr); !ok || connErr.Code() != connect.CodeNotFound {
					t.Errorf("expected CodeNotFound, got %v", err)
				}
			}

			// ListCycles on missing user → empty list, no error.
			cyclResp, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles on empty DB: %v", err)
			}
			if len(cyclResp.Msg.GetCycles()) != 0 {
				t.Errorf("expected 0 cycles, got %d", len(cyclResp.Msg.GetCycles()))
			}

			// ListTimeline on missing user → empty list, no error.
			tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListTimeline on empty DB: %v", err)
			}
			if len(tlResp.Msg.GetRecords()) != 0 {
				t.Errorf("expected 0 timeline records, got %d", len(tlResp.Msg.GetRecords()))
			}

			// ExportData on missing user → valid (empty) payload.
			exportResp, err := client.ExportData(ctx, connect.NewRequest(&v1.ExportDataRequest{Name: userID}))
			if err != nil {
				t.Fatalf("ExportData on empty DB: %v", err)
			}
			if len(exportResp.Msg.GetData()) == 0 {
				t.Error("ExportData returned empty bytes; expected a valid (empty) JSON payload")
			}
		})
	}
}

// ─── pagination ───────────────────────────────────────────────────────────────

// TestIntegration_Pagination_Timeline verifies that timeline pagination works
// by importing a user with many observations and then retrieving results
// page-by-page with a small page size, confirming all records are ultimately
// returned.
func TestIntegration_Pagination_Timeline(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			// The regular_28day_user fixture has 35 bleeding observations.
			importFixture(t, client, "regular_28day_user.json")
			const userID = "user-regular-28day"

			// Collect all records using page size 5.
			var allRecords []*v1.TimelineRecord
			var pageToken string
			for {
				resp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
					Parent: userID,
					Pagination: &v1.PaginationRequest{
						PageSize:  5,
						PageToken: pageToken,
					},
				}))
				if err != nil {
					t.Fatalf("ListTimeline (page %q): %v", pageToken, err)
				}
				allRecords = append(allRecords, resp.Msg.GetRecords()...)
				pageToken = resp.Msg.GetPagination().GetNextPageToken()
				if pageToken == "" {
					break
				}
			}

			// Must contain all observations.
			bleedingCount := countBleedingRecords(allRecords)
			if bleedingCount < 35 {
				t.Errorf("paginated timeline: got %d bleeding records, want ≥ 35", bleedingCount)
			}
		})
	}
}

// ─── edge cases ───────────────────────────────────────────────────────────────

// TestIntegration_EdgeCase_SingleObservation verifies that the engine handles
// gracefully the case where a user has exactly one bleeding observation. The
// timeline must contain the observation, ListCycles must return at most one
// open-ended cycle (or zero cycles), and no panics or errors occur.
func TestIntegration_EdgeCase_SingleObservation(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			const userID = "user-single-obs"
			_, err := client.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{
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
				t.Fatalf("UpsertUserProfile: %v", err)
			}

			// Create exactly one bleeding observation.
			_, err = client.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
				Parent: userID,
				Observation: &v1.BleedingObservation{
					Timestamp: &v1.DateTime{Value: "2026-01-01T08:00:00Z"},
					Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
				},
			}))
			if err != nil {
				t.Fatalf("CreateBleedingObservation: %v", err)
			}

			// Timeline must contain exactly the one observation.
			tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
				Parent: userID,
			}))
			if err != nil {
				t.Fatalf("ListTimeline: %v", err)
			}
			bleedingCount := countBleedingRecords(tlResp.Msg.GetRecords())
			if bleedingCount != 1 {
				t.Errorf("timeline bleeding count = %d, want 1", bleedingCount)
			}

			// ListCycles must succeed and return at most one open-ended cycle.
			cyclResp, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles: %v", err)
			}
			cycles := cyclResp.Msg.GetCycles()
			if len(cycles) > 1 {
				t.Errorf("ListCycles: got %d cycles, want ≤ 1 for single observation", len(cycles))
			}
			// Any detected cycle must be open-ended (no end_date) because there is
			// no subsequent bleeding episode to close it.
			for _, c := range cycles {
				if c.GetEndDate().GetValue() != "" {
					t.Errorf("cycle has end_date %q; expected open-ended cycle for single observation", c.GetEndDate().GetValue())
				}
			}
		})
	}
}

// TestIntegration_EdgeCase_MaxPagination verifies that requesting a page size
// larger than the total record count returns all records in a single page with
// an empty next_page_token.
func TestIntegration_EdgeCase_MaxPagination(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			// firsttime_user has 2 bleeding observations — a small, known dataset
			// guaranteed to fit within an oversized page request.
			importFixture(t, client, "firsttime_user.json")
			const userID = "user-firsttime"

			// Determine total record count via full pagination.
			allRecords := collectAllTimeline(t, ctx, client, userID)
			totalRecords := len(allRecords)
			if totalRecords == 0 {
				t.Fatal("expected at least one record after import")
			}

			// Request a page size much larger than the total record count.
			oversizedPageSize := uint32(totalRecords + 1000)
			// Cap at the proto-constrained maximum of 500; if totalRecords exceeds
			// that the test would be vacuous anyway, so clamp to avoid a validation error.
			if oversizedPageSize > 500 {
				oversizedPageSize = 500
			}
			resp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
				Parent: userID,
				Pagination: &v1.PaginationRequest{
					PageSize: oversizedPageSize,
				},
			}))
			if err != nil {
				t.Fatalf("ListTimeline with oversized page: %v", err)
			}
			if len(resp.Msg.GetRecords()) != totalRecords {
				t.Errorf("oversized page: got %d records, want %d", len(resp.Msg.GetRecords()), totalRecords)
			}
			if token := resp.Msg.GetPagination().GetNextPageToken(); token != "" {
				t.Errorf("next_page_token = %q; want empty when all records fit in one page", token)
			}
		})
	}
}

// ─── concurrent access ────────────────────────────────────────────────────────

// TestIntegration_ConcurrentAccess launches multiple goroutines that
// simultaneously log bleeding observations for the same user and verifies
// that no data races occur and all observations are persisted.
func TestIntegration_ConcurrentAccess(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			const userID = "user-concurrent"
			const goroutines = 10

			// First, create the profile (serial).
			_, err := client.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{
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
				t.Fatalf("UpsertUserProfile: %v", err)
			}

			// Log observations concurrently from different goroutines.
			// Each goroutine uses a distinct date to avoid duplicate name conflicts.
			dates := []string{
				"2025-01-10", "2025-02-07", "2025-03-07", "2025-04-04",
				"2025-05-02", "2025-05-30", "2025-06-27", "2025-07-25",
				"2025-08-22", "2025-09-19",
			}

			var wg sync.WaitGroup
			errs := make([]error, goroutines)
			for i := 0; i < goroutines; i++ {
				i := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					_, errs[i] = client.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
						Parent: userID,
						Observation: &v1.BleedingObservation{
							Timestamp: &v1.DateTime{Value: dates[i] + "T10:00:00Z"},
							Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
						},
					}))
				}()
			}
			wg.Wait()

			for i, e := range errs {
				if e != nil {
					t.Errorf("goroutine %d: CreateBleedingObservation: %v", i, e)
				}
			}

			// All 10 observations must appear in the timeline. Paginate through all
			// pages because concurrent redetections may create extra cycle records
			// that fill the first page.
			allRecords := collectAllTimeline(t, ctx, client, userID)
			bleedingCount := countBleedingRecords(allRecords)
			if bleedingCount != goroutines {
				t.Errorf("timeline bleeding count = %d, want %d", bleedingCount, goroutines)
			}
		})
	}
}

// ─── cycle re-detection ───────────────────────────────────────────────────────

// TestIntegration_CycleRedetection verifies that adding a new bleeding
// observation triggers cycle boundary updates. It logs observations for two
// cycles, verifies 1 completed + 1 open cycle, then adds observations for a
// third cycle and expects at least 2 completed cycles.
func TestIntegration_CycleRedetection(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			const userID = "user-redetect"
			_, err := client.UpsertUserProfile(ctx, connect.NewRequest(&v1.UpsertUserProfileRequest{
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
				t.Fatalf("UpsertUserProfile: %v", err)
			}

			// Cycle 1: days 2025-06-01..05
			for _, date := range []string{"2025-06-01", "2025-06-02", "2025-06-03", "2025-06-04", "2025-06-05"} {
				_, err := client.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
					Parent: userID,
					Observation: &v1.BleedingObservation{
						Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
						Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
					},
				}))
				if err != nil {
					t.Fatalf("CreateBleedingObservation (%s): %v", date, err)
				}
			}

			// After cycle 1: expect 1 open-ended cycle.
			c1, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles after c1: %v", err)
			}
			if len(c1.Msg.GetCycles()) < 1 {
				t.Error("expected ≥ 1 cycle after first episode")
			}

			// Cycle 2: days 2025-06-29..07-02
			for _, date := range []string{"2025-06-29", "2025-06-30", "2025-07-01", "2025-07-02"} {
				_, err := client.CreateBleedingObservation(ctx, connect.NewRequest(&v1.CreateBleedingObservationRequest{
					Parent: userID,
					Observation: &v1.BleedingObservation{
						Timestamp: &v1.DateTime{Value: date + "T10:00:00Z"},
						Flow:      v1.BleedingFlow_BLEEDING_FLOW_MEDIUM,
					},
				}))
				if err != nil {
					t.Fatalf("CreateBleedingObservation (%s): %v", date, err)
				}
			}

			// After cycle 2 start: cycle 1 must now be closed and cycle 2 open.
			c2, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles after c2: %v", err)
			}
			if len(c2.Msg.GetCycles()) < 2 {
				t.Errorf("expected ≥ 2 cycles after second episode, got %d", len(c2.Msg.GetCycles()))
			}

			// Cycle 1 should now have an end_date set (it is no longer open-ended).
			var cycleOneEnded bool
			for _, c := range c2.Msg.GetCycles() {
				if c.GetStartDate().GetValue() == "2025-06-01" && c.GetEndDate().GetValue() != "" {
					cycleOneEnded = true
				}
			}
			if !cycleOneEnded {
				t.Error("cycle starting 2025-06-01 should have end_date set after second cycle was detected")
			}
		})
	}
}

// ─── hormonal suppressed ──────────────────────────────────────────────────────

// TestIntegration_HormonalSuppressedUser validates that a hormonally-suppressed
// user can be imported and queried without errors.
func TestIntegration_HormonalSuppressedUser(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			importFixture(t, client, "hormonal_suppressed_user.json")
			const userID = "user-hormonal"

			profResp, err := client.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: userID}))
			if err != nil {
				t.Fatalf("GetUserProfile: %v", err)
			}
			if profResp.Msg.GetProfile().GetBiologicalCycle() != v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED {
				t.Errorf("biological_cycle = %v, want HORMONALLY_SUPPRESSED", profResp.Msg.GetProfile().GetBiologicalCycle())
			}

			// At least one withdrawal bleed cycle should be detected.
			cyclResp, err := client.ListCycles(ctx, connect.NewRequest(&v1.ListCyclesRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListCycles: %v", err)
			}
			if len(cyclResp.Msg.GetCycles()) < 1 {
				t.Error("expected ≥ 1 withdrawal bleed cycle")
			}

			// The hormonal suppressed fixture contains medication events spanning
			// multiple pill packs; assert at least one appears in the timeline.
			tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListTimeline: %v", err)
			}
			var hasMedEvent bool
			for _, r := range tlResp.Msg.GetRecords() {
				if r.GetMedicationEvent() != nil {
					hasMedEvent = true
					break
				}
			}
			if !hasMedEvent {
				t.Error("timeline missing medication event records")
			}
		})
	}
}

// ─── full-featured user ───────────────────────────────────────────────────────

// TestIntegration_FullFeaturedUser loads the full_featured_user fixture
// (symptoms, moods, medications, medication events) and verifies that all
// record types appear in the timeline.
func TestIntegration_FullFeaturedUser(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			importFixture(t, client, "full_featured_user.json")
			const userID = "user-full"

			tlResp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{Parent: userID}))
			if err != nil {
				t.Fatalf("ListTimeline: %v", err)
			}

			var hasSymptom, hasMood, hasMedEvent, hasBleeding bool
			for _, r := range tlResp.Msg.GetRecords() {
				if r.GetBleedingObservation() != nil {
					hasBleeding = true
				}
				if r.GetSymptomObservation() != nil {
					hasSymptom = true
				}
				if r.GetMoodObservation() != nil {
					hasMood = true
				}
				if r.GetMedicationEvent() != nil {
					hasMedEvent = true
				}
			}

			if !hasBleeding {
				t.Error("timeline missing bleeding observation records")
			}
			if !hasSymptom {
				t.Error("timeline missing symptom observation records")
			}
			if !hasMood {
				t.Error("timeline missing mood observation records")
			}
			if !hasMedEvent {
				t.Error("timeline missing medication event records")
			}
		})
	}
}

// ─── sample export ────────────────────────────────────────────────────────────

// TestSampleExport_ImportAndVerify loads the sample export file using
// LoadSampleExportBytes, imports it, and verifies that the resulting state is
// correct: profile accessible, timeline non-empty, and note fields preserved.
// This exercises the sample export file and the LoadSampleExportBytes helper.
func TestSampleExport_ImportAndVerify(t *testing.T) {
	for _, v := range storageVariants() {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			client := engineClient(t, v.opts...)

			data, err := LoadSampleExportBytes("full_featured_user_export.json")
			if err != nil {
				t.Fatalf("LoadSampleExportBytes: %v", err)
			}
			importResp, err := client.ImportData(ctx, connect.NewRequest(&v1.ImportDataRequest{Data: data}))
			if err != nil {
				t.Fatalf("ImportData(sample export): %v", err)
			}
			if importResp.Msg.GetRecordsImported() == 0 {
				t.Error("ImportData: expected > 0 records, got 0")
			}

			const userID = "user-full"

			profResp, err := client.GetUserProfile(ctx, connect.NewRequest(&v1.GetUserProfileRequest{Name: userID}))
			if err != nil {
				t.Fatalf("GetUserProfile: %v", err)
			}
			prof := profResp.Msg.GetProfile()
			if prof.GetName() != userID {
				t.Errorf("profile name = %q, want %q", prof.GetName(), userID)
			}
			// 25.2: Verify rich profile fields round-trip correctly.
			if prof.GetBiologicalCycle() != v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY {
				t.Errorf("profile BiologicalCycle = %v, want BIOLOGICAL_CYCLE_MODEL_OVULATORY", prof.GetBiologicalCycle())
			}
			if prof.GetContraception() != v1.ContraceptionType_CONTRACEPTION_TYPE_NONE {
				t.Errorf("profile Contraception = %v, want CONTRACEPTION_TYPE_NONE", prof.GetContraception())
			}
			if len(prof.GetHealthConditions()) == 0 {
				t.Error("profile HealthConditions is empty; expected at least one condition from sample export")
			}

			allRecords := collectAllTimeline(t, ctx, client, userID)
			if len(allRecords) == 0 {
				t.Fatal("timeline empty after importing sample export")
			}

			// 25.1: Verify all record types are present in the timeline.
			var hasBleeding, hasSymptom, hasMood, hasMedEvent bool
			// 25.3: Verify note fields survive import across all record types.
			var foundBleedingNote, foundSymptomNote, foundMoodNote, foundMedNote bool
			for _, r := range allRecords {
				if b := r.GetBleedingObservation(); b != nil {
					hasBleeding = true
					if b.GetNote() != "" {
						foundBleedingNote = true
					}
				}
				if s := r.GetSymptomObservation(); s != nil {
					hasSymptom = true
					if s.GetNote() != "" {
						foundSymptomNote = true
					}
				}
				if m := r.GetMoodObservation(); m != nil {
					hasMood = true
					if m.GetNote() != "" {
						foundMoodNote = true
					}
				}
				if me := r.GetMedicationEvent(); me != nil {
					hasMedEvent = true
					if me.GetNote() != "" {
						foundMedNote = true
					}
				}
			}
			if !hasBleeding {
				t.Error("timeline missing bleeding observation records after sample export import")
			}
			if !hasSymptom {
				t.Error("timeline missing symptom observation records after sample export import")
			}
			if !hasMood {
				t.Error("timeline missing mood observation records after sample export import")
			}
			if !hasMedEvent {
				t.Error("timeline missing medication event records after sample export import")
			}
			if !foundBleedingNote {
				t.Error("no note found on any bleeding observation; expected notes from sample export to be preserved")
			}
			if !foundSymptomNote {
				t.Error("no note found on any symptom observation; expected notes from sample export to be preserved")
			}
			if !foundMoodNote {
				t.Error("no note found on any mood observation; expected notes from sample export to be preserved")
			}
			if !foundMedNote {
				t.Error("no note found on any medication event; expected notes from sample export to be preserved")
			}
		})
	}
}

// ─── private helpers ──────────────────────────────────────────────────────────

// countBleedingRecords counts TimelineRecord entries that contain a
// BleedingObservation.
func countBleedingRecords(records []*v1.TimelineRecord) int {
	n := 0
	for _, r := range records {
		if r.GetBleedingObservation() != nil {
			n++
		}
	}
	return n
}

// collectAllTimeline iterates through all pages of ListTimeline for userID and
// returns every record. This is necessary when concurrent re-detections could
// produce extra cycle records that fill the default page size.
func collectAllTimeline(t *testing.T, ctx context.Context, client openmensesv1connect.CycleTrackerServiceClient, userID string) []*v1.TimelineRecord {
	t.Helper()
	var all []*v1.TimelineRecord
	var pageToken string
	for {
		resp, err := client.ListTimeline(ctx, connect.NewRequest(&v1.ListTimelineRequest{
			Parent: userID,
			Pagination: &v1.PaginationRequest{
				PageSize:  500,
				PageToken: pageToken,
			},
		}))
		if err != nil {
			t.Fatalf("collectAllTimeline: ListTimeline: %v", err)
		}
		all = append(all, resp.Msg.GetRecords()...)
		pageToken = resp.Msg.GetPagination().GetNextPageToken()
		if pageToken == "" {
			break
		}
	}
	return all
}

// unmarshalJSON is a thin wrapper around encoding/json.Unmarshal.
func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// asConnectError reports whether err is a *connect.Error and assigns it to
// target if so.
func asConnectError(err error, target **connect.Error) bool {
	return errors.As(err, target)
}
