// Package rules implements the core domain logic for openmenses: cycle
// detection, cycle statistics, and phase estimation.
package rules

import (
	"context"
	"sort"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/2ajoyce/openmenses/engine/internal/storage"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// minNonBleedingGap is the number of consecutive non-bleeding days required
// before a new cycle begins. A date gap strictly greater than this value
// (i.e., ≥ 4 calendar days between consecutive bleeding dates) separates
// two distinct episodes per domain rules §1.1.
const minNonBleedingGap = 3

// MinCycleLength is the minimum valid cycle length in days per domain rules
// §1.2. Cycles shorter than this are considered outliers.
const MinCycleLength = 15

// MaxCycleLength is the maximum valid cycle length in days per domain rules
// §1.2. Cycles longer than this are considered outliers.
const MaxCycleLength = 90

// IsOutlierLength reports whether a completed cycle's length falls outside
// the valid bounds (15–90 days) per domain rules §1.2. Open-ended cycles
// (no end_date) are never considered outliers.
func IsOutlierLength(c *v1.Cycle) bool {
	l := CycleLength(c)
	if l <= 0 {
		return false // open-ended or unparseable
	}
	return l < MinCycleLength || l > MaxCycleLength
}

// DetectCycles derives cycles from bleeding observations stored for userID.
// USER_CONFIRMED cycles are fetched from storage and preserved unchanged.
// DERIVED cycles are recomputed from scratch; the caller is responsible for
// reconciling the returned list with any previously stored derived cycles.
// The returned slice is sorted by start_date ascending.
func DetectCycles(ctx context.Context, userID string, store storage.Repository) ([]*v1.Cycle, error) {
	obs, err := allBleedingObs(ctx, userID, store)
	if err != nil {
		return nil, err
	}

	confirmed, err := confirmedCycles(ctx, userID, store)
	if err != nil {
		return nil, err
	}

	derived := computeDerivedCycles(userID, obs)

	// Remove derived cycles whose date range overlaps a confirmed cycle.
	final := make([]*v1.Cycle, 0, len(confirmed)+len(derived))
	final = append(final, confirmed...)
	for _, d := range derived {
		if !overlapsAny(d, confirmed) {
			final = append(final, d)
		}
	}

	sort.Slice(final, func(i, j int) bool {
		return final[i].GetStartDate().GetValue() < final[j].GetStartDate().GetValue()
	})

	return final, nil
}

// allBleedingObs returns all BleedingObservation records for userID, paginating
// across all pages.
func allBleedingObs(ctx context.Context, userID string, store storage.Repository) ([]*v1.BleedingObservation, error) {
	var all []*v1.BleedingObservation
	req := storage.PageRequest{PageSize: 500}
	for {
		page, err := store.BleedingObservations().ListByUserAndDateRange(
			ctx, userID, "0001-01-01", "9999-12-31", req,
		)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		if page.NextPageToken == "" {
			break
		}
		req.PageToken = page.NextPageToken
	}
	return all, nil
}

// confirmedCycles returns all CYCLE_SOURCE_USER_CONFIRMED Cycle records for
// userID, paginating across all pages.
func confirmedCycles(ctx context.Context, userID string, store storage.Repository) ([]*v1.Cycle, error) {
	var all []*v1.Cycle
	req := storage.PageRequest{PageSize: 500}
	for {
		page, err := store.Cycles().ListByUser(ctx, userID, req)
		if err != nil {
			return nil, err
		}
		for _, c := range page.Items {
			if c.GetSource() == v1.CycleSource_CYCLE_SOURCE_USER_CONFIRMED {
				all = append(all, c)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		req.PageToken = page.NextPageToken
	}
	return all, nil
}

// computeDerivedCycles computes DERIVED_FROM_BLEEDING cycles from observations.
// Each returned cycle has a freshly generated ULID.
func computeDerivedCycles(userID string, obs []*v1.BleedingObservation) []*v1.Cycle {
	if len(obs) == 0 {
		return nil
	}

	// Build date → set-of-flows map.
	dayMap := make(map[string]map[v1.BleedingFlow]struct{})
	for _, o := range obs {
		d := dateFromTimestamp(o.GetTimestamp().GetValue())
		if d == "" {
			continue
		}
		if dayMap[d] == nil {
			dayMap[d] = make(map[v1.BleedingFlow]struct{})
		}
		dayMap[d][o.GetFlow()] = struct{}{}
	}
	if len(dayMap) == 0 {
		return nil
	}

	// Produce sorted unique date strings.
	dates := make([]string, 0, len(dayMap))
	for d := range dayMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	episodeStarts := findEpisodeStarts(dates, dayMap)
	if len(episodeStarts) == 0 {
		return nil
	}

	entropy := ulid.DefaultEntropy()
	derived := make([]*v1.Cycle, 0, len(episodeStarts))
	for i, start := range episodeStarts {
		// end_date = the day before the next episode's start.
		var endDate string
		if i+1 < len(episodeStarts) {
			nt, err := time.Parse("2006-01-02", episodeStarts[i+1])
			if err == nil {
				endDate = nt.AddDate(0, 0, -1).Format("2006-01-02")
			}
		}

		c := &v1.Cycle{
			Name:      ulid.MustNew(ulid.Now(), entropy).String(),
			UserId:    userID,
			StartDate: &v1.LocalDate{Value: start},
			Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
		}
		if endDate != "" {
			c.EndDate = &v1.LocalDate{Value: endDate}
		}
		derived = append(derived, c)
	}
	return derived
}

// findEpisodeStarts returns the start date of each valid bleeding episode.
// A new episode begins when the gap between two consecutive bleeding dates
// is greater than minNonBleedingGap calendar days. Spotting-only episode
// starts are dropped unless heavier flow occurs within 2 days (§1.1).
func findEpisodeStarts(dates []string, dayMap map[string]map[v1.BleedingFlow]struct{}) []string {
	var starts []string
	for i, d := range dates {
		isStart := i == 0
		if !isStart {
			prev, err1 := time.Parse("2006-01-02", dates[i-1])
			cur, err2 := time.Parse("2006-01-02", d)
			if err1 != nil || err2 != nil {
				continue
			}
			gapDays := int(cur.Sub(prev).Hours() / 24)
			if gapDays > minNonBleedingGap {
				isStart = true
			}
		}
		if !isStart {
			continue
		}

		// Spotting-only start: require heavier flow within 2 calendar days.
		if spottingOnly(dayMap[d]) && !hasHeavierWithin2Days(d, dayMap) {
			continue
		}

		starts = append(starts, d)
	}
	return starts
}

// spottingOnly returns true when every flow in the set is BLEEDING_FLOW_SPOTTING.
func spottingOnly(flows map[v1.BleedingFlow]struct{}) bool {
	for f := range flows {
		if f != v1.BleedingFlow_BLEEDING_FLOW_SPOTTING {
			return false
		}
	}
	return true
}

// hasHeavierWithin2Days returns true when any date within 2 calendar days of d
// has a non-spotting bleeding observation.
func hasHeavierWithin2Days(d string, dayMap map[string]map[v1.BleedingFlow]struct{}) bool {
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return false
	}
	for delta := 1; delta <= 2; delta++ {
		next := t.AddDate(0, 0, delta).Format("2006-01-02")
		if flows, ok := dayMap[next]; ok && !spottingOnly(flows) {
			return true
		}
	}
	return false
}

// dateFromTimestamp extracts the YYYY-MM-DD prefix from an RFC 3339 timestamp.
func dateFromTimestamp(ts string) string {
	if len(ts) < 10 {
		return ""
	}
	return ts[:10]
}

// overlapsAny reports whether cycle c's date range overlaps any cycle in the
// given slice.
func overlapsAny(c *v1.Cycle, cycles []*v1.Cycle) bool {
	for _, other := range cycles {
		if cyclesOverlap(c, other) {
			return true
		}
	}
	return false
}

// cyclesOverlap reports whether two cycles have overlapping date ranges.
// Open-ended cycles (empty end_date) are treated as extending to 9999-12-31.
func cyclesOverlap(a, b *v1.Cycle) bool {
	aStart := a.GetStartDate().GetValue()
	aEnd := openEnd(a.GetEndDate().GetValue())
	bStart := b.GetStartDate().GetValue()
	bEnd := openEnd(b.GetEndDate().GetValue())
	return aStart <= bEnd && bStart <= aEnd
}

// openEnd returns end if non-empty, otherwise the far-future sentinel used for
// open-ended cycle comparisons.
func openEnd(end string) string {
	if end == "" {
		return "9999-12-31"
	}
	return end
}
