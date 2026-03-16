package service

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	"github.com/2ajoyce/openmenses/engine/internal/storage"
	"github.com/2ajoyce/openmenses/engine/internal/timeline"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// ─── RPC: cycle statistics ────────────────────────────────────────────────────

// GetCycleStatistics computes aggregate statistics over the user's cycles.
// When window_size > 0, only the last N completed cycles are included.
func (s *CycleTrackerService) GetCycleStatistics(
	ctx context.Context,
	req *connect.Request[v1.GetCycleStatisticsRequest],
) (*connect.Response[v1.GetCycleStatisticsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	cycles, err := rules.DetectCycles(ctx, req.Msg.GetParent(), s.store)
	if err != nil {
		return nil, toConnectErr(err)
	}
	var stats rules.CycleStats
	if req.Msg.GetWindowSize() > 0 {
		stats = rules.WindowStats(cycles, int(req.Msg.GetWindowSize()))
	} else {
		stats = rules.Stats(cycles)
	}
	return connect.NewResponse(&v1.GetCycleStatisticsResponse{
		Statistics: &v1.CycleStatistics{
			Count:   int32(stats.Count),
			Average: stats.Average,
			Median:  stats.Median,
			Min:     int32(stats.Min),
			Max:     int32(stats.Max),
			StdDev:  stats.StdDev,
		},
	}), nil
}

// ─── RPC: timeline ────────────────────────────────────────────────────────────

// ListTimeline queries all record types within the requested date range, merges
// them into a single chronological (most-recent-first) list, and applies
// offset-based pagination. The core assembly logic lives in the timeline
// package and is invoked here.
func (s *CycleTrackerService) ListTimeline(
	ctx context.Context,
	req *connect.Request[v1.ListTimelineRequest],
) (*connect.Response[v1.ListTimelineResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	userID := req.Msg.GetParent()
	start, end := "0001-01-01", "9999-12-31"
	if r := req.Msg.GetRange(); r != nil {
		if r.GetStart().GetValue() != "" {
			start = r.GetStart().GetValue()
		}
		if r.GetEnd().GetValue() != "" {
			end = r.GetEnd().GetValue()
		}
	}

	records, nextToken, err := timeline.BuildTimeline(ctx, s.store, userID, start, end, pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListTimelineResponse{
		Records:    records,
		Pagination: &v1.PaginationResponse{NextPageToken: nextToken},
	}), nil
}

// ─── RPC: cycles ──────────────────────────────────────────────────────────────

// ListCycles returns cycles for the given user from the stored cycles table
// with offset-based pagination. Stored cycles are kept current by
// redetectAndStoreCycles, which is triggered on every bleeding observation
// mutation.
func (s *CycleTrackerService) ListCycles(
	ctx context.Context,
	req *connect.Request[v1.ListCyclesRequest],
) (*connect.Response[v1.ListCyclesResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	page, err := s.store.Cycles().ListByUser(ctx, req.Msg.GetParent(), pageReq(req.Msg.GetPagination()))
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListCyclesResponse{
		Cycles:     page.Items,
		Pagination: &v1.PaginationResponse{NextPageToken: page.NextPageToken},
	}), nil
}

// GetCycle fetches a single cycle by ID.
// Returns CodeNotFound if no cycle exists.
func (s *CycleTrackerService) GetCycle(
	ctx context.Context,
	req *connect.Request[v1.GetCycleRequest],
) (*connect.Response[v1.GetCycleResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	cycle, err := s.store.Cycles().GetByID(ctx, req.Msg.GetName())
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.GetCycleResponse{Cycle: cycle}), nil
}

// ─── RPC: predictions & insights (Phase 4/5 stubs) ───────────────────────────

// ListPredictions returns an empty list. Prediction generation is deferred to
// Phase 4.
func (s *CycleTrackerService) ListPredictions(
	_ context.Context,
	req *connect.Request[v1.ListPredictionsRequest],
) (*connect.Response[v1.ListPredictionsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListPredictionsResponse{}), nil
}

// ListInsights returns an empty list. Insight generation is deferred to
// Phase 5.
func (s *CycleTrackerService) ListInsights(
	_ context.Context,
	req *connect.Request[v1.ListInsightsRequest],
) (*connect.Response[v1.ListInsightsResponse], error) {
	if err := s.validator.ValidateRequest(req.Msg); err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(&v1.ListInsightsResponse{}), nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// redetectAndStoreCycles recomputes derived cycles for userID and replaces the
// stored derived cycles with the new set. User-confirmed cycles are never
// modified. Phase estimates are regenerated for all cycles after detection.
func (s *CycleTrackerService) redetectAndStoreCycles(ctx context.Context, userID string) error {
	// Delete all existing derived cycles and their associated phase estimates.
	existing, err := paginateAll(ctx, func(ctx context.Context, token string) ([]*v1.Cycle, string, error) {
		pg, err := s.store.Cycles().ListByUser(ctx, userID, storage.PageRequest{PageSize: 500, PageToken: token})
		return pg.Items, pg.NextPageToken, err
	})
	if err != nil {
		return err
	}
	for _, c := range existing {
		if c.GetSource() == v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING {
			if err := s.store.PhaseEstimates().DeleteByCycleID(ctx, c.GetName()); err != nil {
				return err
			}
			if err := s.store.Cycles().DeleteByID(ctx, c.GetName()); err != nil && !errors.Is(err, storage.ErrNotFound) {
				return err
			}
		}
	}

	// Compute fresh derived cycles and persist them.
	newCycles, err := rules.DetectCycles(ctx, userID, s.store)
	if err != nil {
		return err
	}
	for _, c := range newCycles {
		if c.GetSource() == v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING {
			if err := s.store.Cycles().Create(ctx, c); err != nil {
				return err
			}
		}
	}

	// Estimate and store phases for all cycles (derived + confirmed).
	return s.estimateAndStorePhases(ctx, userID, newCycles)
}

// estimateAndStorePhases computes PhaseEstimates for each cycle and persists
// them. Conflicts (already-existing estimates) are silently skipped.
func (s *CycleTrackerService) estimateAndStorePhases(ctx context.Context, userID string, cycles []*v1.Cycle) error {
	profile, err := s.store.UserProfiles().GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil // no profile yet — estimation requires profile
		}
		return err
	}

	stats := rules.Stats(cycles)
	avgLen := int(math.Round(stats.Average))
	completed := len(rules.CompletedCycles(cycles))

	for _, c := range cycles {
		estimates := rules.EstimatePhases(c, profile, avgLen, completed)
		for _, est := range estimates {
			if err := s.store.PhaseEstimates().Create(ctx, est); err != nil {
				if !errors.Is(err, storage.ErrConflict) {
					return err
				}
			}
		}
	}
	return nil
}
