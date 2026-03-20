// Package timeline provides BuildTimeline, which queries all observable record
// types for a user within a date range, merges them into a single
// chronologically ordered slice, and applies offset-based pagination.
package timeline

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/oklog/ulid/v2"

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
	// key is the full timestamp or date string used for lexicographic
	// descending sort (most-recent-first). Using the full timestamp preserves
	// intra-day chronological ordering for records that carry a time component.
	key    string
	record *v1.TimelineRecord
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
		entries = append(entries, entry{key: key, record: rec})
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

	// ── predictions ────────────────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.Predictions().ListByUser(ctx, userID, req)
			if err != nil {
				return nil, "", fmt.Errorf("list predictions: %w", err)
			}
			for _, pred := range pg.Items {
				// Filter in-memory: include predictions where predicted_start_date <= end
				// AND (predicted_end_date >= start OR predicted_end_date is empty).
				predStart := pred.GetPredictedStartDate().GetValue()
				predEnd := pred.GetPredictedEndDate().GetValue()

				if predStart > end {
					continue
				}
				if predEnd != "" && predEnd < start {
					continue
				}

				add(predStart, &v1.TimelineRecord{
					Record: &v1.TimelineRecord_Prediction{Prediction: pred},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// ── insights ────────────────────────────────────────────────────────────────
	{
		req := storage.PageRequest{PageSize: fetchPageSize}
		for {
			pg, err := store.Insights().ListByUser(ctx, userID, req)
			if err != nil {
				return nil, "", fmt.Errorf("list insights: %w", err)
			}
			for _, ins := range pg.Items {
				// Decode the ULID creation timestamp so insights sort inline with
				// other records from the same moment rather than always after all
				// date strings (ULID "01J..." < date "2026-..." lexicographically).
				id, err := ulid.Parse(ins.GetName())
				if err != nil {
					// Name is not a bare ULID; fall back to including it unfiltered.
					add(ins.GetName(), &v1.TimelineRecord{
						Record: &v1.TimelineRecord_Insight{Insight: ins},
					})
					continue
				}
				key := time.UnixMilli(int64(id.Time())).UTC().Format(time.RFC3339Nano)

				// Filter in-memory: only include insights whose creation timestamp
				// falls within [start, end] (same contract as other date-filtered
				// record types). This prevents insights from appearing beyond the
				// first page when a large date range is selected.
				creationDate := time.UnixMilli(int64(id.Time())).UTC().Format("2006-01-02")
				if creationDate > end || creationDate < start {
					continue
				}

				add(key, &v1.TimelineRecord{
					Record: &v1.TimelineRecord_Insight{Insight: ins},
				})
			}
			if pg.NextPageToken == "" {
				break
			}
			req.PageToken = pg.NextPageToken
		}
	}

	// Sort most-recent-first (descending lexicographic order of full timestamp/date keys).
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
