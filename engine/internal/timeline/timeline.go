// Package timeline provides BuildTimeline, which queries all observable record
// types for a user within a date range, merges them into a single
// chronologically ordered slice, and applies offset-based pagination.
package timeline

import (
	"context"
	"fmt"
	"sort"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

const (
	defaultPageSize = 50
	maxPageSize     = 500
	fetchPageSize   = 500
)

// entry is an intermediate type used to accumulate records before sorting.
type entry struct {
	// key is the YYYY-MM-DD date prefix (or the full timestamp) used for
	// lexicographic descending sort (most-recent-first).
	key    string
	record *v1.TimelineRecord
}

// dateKey extracts the YYYY-MM-DD prefix from a timestamp or date string.
// If the string is shorter than 10 characters it is returned as-is.
func dateKey(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// BuildTimeline queries every observable record type (bleeding observations,
// symptom observations, mood observations, medication events, cycles, and
// phase estimates) for the given userID within the [start, end] date range
// (both in YYYY-MM-DD format).
//
// Results are merged into a single slice sorted most-recent-first and then
// paginated using offset-based tokens compatible with the proto
// PaginationRequest / PaginationResponse contract.
//
// page.PageSize defaults to 50 and is capped at 500.
// page.PageToken must be the opaque token returned by the previous call (or
// empty to begin from the first page).
//
// Returns the page of TimelineRecord pointers and the next-page token (empty
// when there are no further pages).
func BuildTimeline(
	ctx context.Context,
	store storage.Repository,
	userID, start, end string,
	page storage.PageRequest,
) ([]*v1.TimelineRecord, string, error) {
	var entries []entry

	add := func(key string, rec *v1.TimelineRecord) {
		entries = append(entries, entry{key: dateKey(key), record: rec})
	}

	// ── bleeding observations ────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.BleedingObservations().ListByUserAndDateRange(ctx, userID, start, end, req)
			if err != nil {
				return nil, "", fmt.Errorf("list bleeding observations: %w", err)
			}
			for _, o := range pg.Items {
				add(o.GetTimestamp().GetValue(), &v1.TimelineRecord{
					Record: &v1.TimelineRecord_BleedingObservation{BleedingObservation: o},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// ── symptom observations ─────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.SymptomObservations().ListByUserAndDateRange(ctx, userID, start, end, req)
			if err != nil {
				return nil, "", fmt.Errorf("list symptom observations: %w", err)
			}
			for _, o := range pg.Items {
				add(o.GetTimestamp().GetValue(), &v1.TimelineRecord{
					Record: &v1.TimelineRecord_SymptomObservation{SymptomObservation: o},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// ── mood observations ────────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.MoodObservations().ListByUserAndDateRange(ctx, userID, start, end, req)
			if err != nil {
				return nil, "", fmt.Errorf("list mood observations: %w", err)
			}
			for _, o := range pg.Items {
				add(o.GetTimestamp().GetValue(), &v1.TimelineRecord{
					Record: &v1.TimelineRecord_MoodObservation{MoodObservation: o},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// ── medication events ────────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.MedicationEvents().ListByUserAndDateRange(ctx, userID, start, end, req)
			if err != nil {
				return nil, "", fmt.Errorf("list medication events: %w", err)
			}
			for _, o := range pg.Items {
				add(o.GetTimestamp().GetValue(), &v1.TimelineRecord{
					Record: &v1.TimelineRecord_MedicationEvent{MedicationEvent: o},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// ── cycles ───────────────────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.Cycles().ListByUserAndDateRange(ctx, userID, start, end, req)
			if err != nil {
				return nil, "", fmt.Errorf("list cycles: %w", err)
			}
			for _, c := range pg.Items {
				add(c.GetStartDate().GetValue(), &v1.TimelineRecord{
					Record: &v1.TimelineRecord_Cycle{Cycle: c},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// ── phase estimates ───────────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.PhaseEstimates().ListByUserAndDateRange(ctx, userID, start, end, req)
			if err != nil {
				return nil, "", fmt.Errorf("list phase estimates: %w", err)
			}
			for _, pe := range pg.Items {
				add(pe.GetDate().GetValue(), &v1.TimelineRecord{
					Record: &v1.TimelineRecord_PhaseEstimate{PhaseEstimate: pe},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// Sort most-recent-first (descending lexicographic order of YYYY-MM-DD keys).
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].key > entries[j].key
	})

	// Resolve offset from the opaque page token.
	offset := 0
	if page.PageToken != "" {
		if _, err := fmt.Sscanf(page.PageToken, "%d", &offset); err != nil {
			offset = 0
		}
	}

	pageSize := int(page.PageSize)
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	if offset >= len(entries) {
		return nil, "", nil
	}

	endIdx := offset + pageSize
	var nextToken string
	if endIdx < len(entries) {
		nextToken = fmt.Sprintf("%d", endIdx)
	}
	if endIdx > len(entries) {
		endIdx = len(entries)
	}

	records := make([]*v1.TimelineRecord, 0, endIdx-offset)
	for _, e := range entries[offset:endIdx] {
		records = append(records, e.record)
	}

	return records, nextToken, nil
}
