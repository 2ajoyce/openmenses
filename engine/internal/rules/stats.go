package rules

import (
	"math"
	"sort"
	"time"

	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// CycleStats holds statistical measures over a set of completed cycles.
type CycleStats struct {
	// Count is the number of completed cycles analysed.
	Count int
	// Average is the mean cycle length in days.
	Average float64
	// Median is the median cycle length in days.
	Median float64
	// Min is the length of the shortest cycle in days.
	Min int
	// Max is the length of the longest cycle in days.
	Max int
	// StdDev is the population standard deviation of cycle lengths in days.
	StdDev float64
}

// Stats computes statistics over the given cycles. Open-ended cycles (those
// without an end_date) are excluded from the computation. Returns a zero-value
// CycleStats when no completed cycles are present.
func Stats(cycles []*v1.Cycle) CycleStats {
	lengths := completedLengths(cycles)
	if len(lengths) == 0 {
		return CycleStats{}
	}
	return buildStats(lengths)
}

// WindowStats computes statistics over the last n completed cycles ordered by
// start_date. Open-ended cycles are excluded. If fewer than n completed cycles
// exist, all completed cycles are used.
func WindowStats(cycles []*v1.Cycle, n int) CycleStats {
	completed := completedOnly(cycles)
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].GetStartDate().GetValue() < completed[j].GetStartDate().GetValue()
	})
	if len(completed) > n {
		completed = completed[len(completed)-n:]
	}
	return Stats(completed)
}

// completedOnly returns cycles that have a non-empty end_date.
func completedOnly(cycles []*v1.Cycle) []*v1.Cycle {
	out := make([]*v1.Cycle, 0, len(cycles))
	for _, c := range cycles {
		if c.GetEndDate().GetValue() != "" {
			out = append(out, c)
		}
	}
	return out
}

// completedLengths returns the length in days for each completed cycle,
// excluding outlier-length cycles (outside 15–90 days) per domain rules §1.2.
func completedLengths(cycles []*v1.Cycle) []int {
	out := make([]int, 0, len(cycles))
	for _, c := range cycles {
		if l := CycleLength(c); l > 0 && !IsOutlierLength(c) {
			out = append(out, l)
		}
	}
	return out
}

// CycleLength returns the inclusive number of days from start_date to end_date.
// Returns 0 for open-ended cycles or cycles with unparseable dates.
func CycleLength(c *v1.Cycle) int {
	start := c.GetStartDate().GetValue()
	end := c.GetEndDate().GetValue()
	if start == "" || end == "" {
		return 0
	}
	st, err1 := time.Parse("2006-01-02", start)
	en, err2 := time.Parse("2006-01-02", end)
	if err1 != nil || err2 != nil {
		return 0
	}
	days := int(en.Sub(st).Hours()/24) + 1
	if days <= 0 {
		return 0
	}
	return days
}

// buildStats computes statistical measures from a non-empty slice of lengths.
func buildStats(lengths []int) CycleStats {
	n := len(lengths)
	mn, mx, sum := lengths[0], lengths[0], 0
	for _, l := range lengths {
		sum += l
		if l < mn {
			mn = l
		}
		if l > mx {
			mx = l
		}
	}
	avg := float64(sum) / float64(n)

	// Median — sort a copy.
	sorted := make([]int, n)
	copy(sorted, lengths)
	sort.Ints(sorted)
	var median float64
	if n%2 == 1 {
		median = float64(sorted[n/2])
	} else {
		median = float64(sorted[n/2-1]+sorted[n/2]) / 2.0
	}

	// Population standard deviation.
	variance := 0.0
	for _, l := range lengths {
		diff := float64(l) - avg
		variance += diff * diff
	}
	variance /= float64(n)

	return CycleStats{
		Count:   n,
		Average: avg,
		Median:  median,
		Min:     mn,
		Max:     mx,
		StdDev:  math.Sqrt(variance),
	}
}
