package rules_test

import (
	"math"
	"testing"

	"github.com/2ajoyce/openmenses/engine/internal/rules"
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// makeCycle creates a Cycle with YYYY-MM-DD start and end dates.
// Pass empty string for end to create an open-ended cycle.
func makeCycle(id, uid, start, end string) *v1.Cycle {
	c := &v1.Cycle{
		Name:      id,
		UserId:    uid,
		StartDate: &v1.LocalDate{Value: start},
		Source:    v1.CycleSource_CYCLE_SOURCE_DERIVED_FROM_BLEEDING,
	}
	if end != "" {
		c.EndDate = &v1.LocalDate{Value: end}
	}
	return c
}

// approxEqual returns true when a and b differ by less than eps.
func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

// ---- Stats: empty and open-ended ----------------------------------------- //

func TestStats_Empty(t *testing.T) {
	s := rules.Stats(nil)
	if s.Count != 0 {
		t.Errorf("Count = %d, want 0", s.Count)
	}
}

func TestStats_AllOpenEnded(t *testing.T) {
	cycles := []*v1.Cycle{makeCycle("c1", "u1", "2026-01-01", "")}
	s := rules.Stats(cycles)
	if s.Count != 0 {
		t.Errorf("Count = %d, want 0 (open-ended excluded)", s.Count)
	}
}

// ---- Stats: single cycle -------------------------------------------------- //

func TestStats_SingleCycle28Days(t *testing.T) {
	// Jan 1 – Jan 28 = 28 days inclusive.
	cycles := []*v1.Cycle{makeCycle("c1", "u1", "2026-01-01", "2026-01-28")}
	s := rules.Stats(cycles)
	if s.Count != 1 {
		t.Fatalf("Count = %d, want 1", s.Count)
	}
	if s.Average != 28 {
		t.Errorf("Average = %.2f, want 28", s.Average)
	}
	if s.Median != 28 {
		t.Errorf("Median = %.2f, want 28", s.Median)
	}
	if s.Min != 28 || s.Max != 28 {
		t.Errorf("Min=%d Max=%d, want 28 28", s.Min, s.Max)
	}
	if s.StdDev != 0 {
		t.Errorf("StdDev = %.4f, want 0 for single cycle", s.StdDev)
	}
}

// ---- Stats: known dataset ------------------------------------------------- //

// Three cycles: 28, 30, 26 days.
// Average  = 84/3 = 28.
// Sorted   = [26, 28, 30] → Median = 28.
// Variance = ((28-28)²+(30-28)²+(26-28)²)/3 = (0+4+4)/3 = 8/3.
// StdDev   ≈ 1.6330.
func TestStats_ThreeCycles(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"), // 28 days
		makeCycle("c2", "u1", "2026-01-29", "2026-02-27"), // 30 days
		makeCycle("c3", "u1", "2026-02-28", "2026-03-25"), // 26 days
	}
	s := rules.Stats(cycles)
	if s.Count != 3 {
		t.Fatalf("Count = %d, want 3", s.Count)
	}
	if !approxEqual(s.Average, 28.0, 0.001) {
		t.Errorf("Average = %.4f, want 28", s.Average)
	}
	if !approxEqual(s.Median, 28.0, 0.001) {
		t.Errorf("Median = %.4f, want 28", s.Median)
	}
	if s.Min != 26 {
		t.Errorf("Min = %d, want 26", s.Min)
	}
	if s.Max != 30 {
		t.Errorf("Max = %d, want 30", s.Max)
	}
	wantStdDev := math.Sqrt(8.0 / 3.0)
	if !approxEqual(s.StdDev, wantStdDev, 0.001) {
		t.Errorf("StdDev = %.4f, want %.4f", s.StdDev, wantStdDev)
	}
}

// ---- Stats: even count median -------------------------------------------- //

// Two cycles: 28 and 30 days → median = 29.
func TestStats_EvenCount_Median(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-01-29", "2026-02-27"),
	}
	s := rules.Stats(cycles)
	if !approxEqual(s.Median, 29.0, 0.001) {
		t.Errorf("Median = %.4f, want 29 for cycles of 28 and 30 days", s.Median)
	}
}

// ---- WindowStats ---------------------------------------------------------- //

func TestWindowStats_LastN(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2025-01-01", "2025-01-20"), // 20 days — oldest
		makeCycle("c2", "u1", "2026-01-01", "2026-01-28"), // 28 days
		makeCycle("c3", "u1", "2026-01-29", "2026-02-27"), // 30 days — newest
	}
	// Last 2 cycles: 28 and 30 → average 29.
	s := rules.WindowStats(cycles, 2)
	if s.Count != 2 {
		t.Fatalf("Count = %d, want 2", s.Count)
	}
	if !approxEqual(s.Average, 29.0, 0.001) {
		t.Errorf("Average = %.4f, want 29 for last 2 cycles", s.Average)
	}
}

func TestWindowStats_FewerThanN(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
	}
	// Requesting window of 5 but only 1 available.
	s := rules.WindowStats(cycles, 5)
	if s.Count != 1 {
		t.Fatalf("Count = %d, want 1", s.Count)
	}
}

func TestWindowStats_IgnoresOpenEnded(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
		makeCycle("c2", "u1", "2026-01-29", ""), // open-ended, excluded
	}
	s := rules.WindowStats(cycles, 10)
	if s.Count != 1 {
		t.Fatalf("Count = %d, want 1 (open-ended excluded)", s.Count)
	}
}

// ---- Stats: outlier filtering --------------------------------------------- //

func TestStats_ExcludesOutlierShortCycle(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"), // 28 days — normal
		makeCycle("c2", "u1", "2026-01-29", "2026-02-05"), // 8 days — outlier (< 15)
		makeCycle("c3", "u1", "2026-02-06", "2026-03-07"), // 30 days — normal
	}
	s := rules.Stats(cycles)
	// Only 28 and 30 should be counted.
	if s.Count != 2 {
		t.Fatalf("Count = %d, want 2 (outlier excluded)", s.Count)
	}
	if !approxEqual(s.Average, 29.0, 0.001) {
		t.Errorf("Average = %.4f, want 29 (only 28 and 30)", s.Average)
	}
}

func TestStats_ExcludesOutlierLongCycle(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-28"), // 28 days — normal
		makeCycle("c2", "u1", "2026-01-29", "2026-05-07"), // 99 days — outlier (> 90)
		makeCycle("c3", "u1", "2026-05-08", "2026-06-06"), // 30 days — normal
	}
	s := rules.Stats(cycles)
	if s.Count != 2 {
		t.Fatalf("Count = %d, want 2 (outlier excluded)", s.Count)
	}
	if !approxEqual(s.Average, 29.0, 0.001) {
		t.Errorf("Average = %.4f, want 29 (only 28 and 30)", s.Average)
	}
}

func TestStats_AllOutliers_EmptyResult(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2026-01-01", "2026-01-05"), // 5 days
		makeCycle("c2", "u1", "2026-01-06", "2026-05-15"), // 130 days
	}
	s := rules.Stats(cycles)
	if s.Count != 0 {
		t.Fatalf("Count = %d, want 0 (all outliers)", s.Count)
	}
}

func TestWindowStats_ExcludesOutliers(t *testing.T) {
	cycles := []*v1.Cycle{
		makeCycle("c1", "u1", "2025-01-01", "2025-01-28"), // 28 days
		makeCycle("c2", "u1", "2025-01-29", "2025-02-03"), // 6 days — outlier
		makeCycle("c3", "u1", "2026-01-01", "2026-01-30"), // 30 days
	}
	// Last 3 cycles include all 3, but outlier is filtered from stats.
	s := rules.WindowStats(cycles, 3)
	if s.Count != 2 {
		t.Fatalf("Count = %d, want 2 (outlier excluded)", s.Count)
	}
	if !approxEqual(s.Average, 29.0, 0.001) {
		t.Errorf("Average = %.4f, want 29 (only 28 and 30)", s.Average)
	}
}
