package rules_test

import (
	"fmt"
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

// ============================================================================
// TestStats — Table-driven tests for cycle statistics
// ============================================================================

type statsTestCase struct {
	name        string
	cycles      []*v1.Cycle
	wantCount   int
	wantAvg     float64
	wantMedian  float64
	wantMin     int
	wantMax     int
	wantStdDev  float64
	checkStdDev bool
}

// makeCycles is a helper to create numbered cycles for table-driven tests.
func makeCyclesFromDays(userID string, days []int) []*v1.Cycle {
	cycles := make([]*v1.Cycle, 0, len(days))
	day := 1
	month := 1
	year := 2026

	for i, d := range days {
		startDay := day
		startMonth := month
		startYear := year

		// Calculate end date
		day += d - 1
		if day > daysInMonth(month, year) {
			day -= daysInMonth(month, year)
			month++
			if month > 12 {
				month = 1
				year++
			}
		}

		start := fmt.Sprintf("%04d-%02d-%02d", startYear, startMonth, startDay)
		end := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		cycles = append(cycles, makeCycle(fmt.Sprintf("c%d", i+1), userID, start, end))

		// Move to next day
		day++
		if day > daysInMonth(month, year) {
			day = 1
			month++
			if month > 12 {
				month = 1
				year++
			}
		}
	}
	return cycles
}

// daysInMonth returns the number of days in a given month/year.
func daysInMonth(month, year int) int {
	if month == 2 {
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}

func TestStats(t *testing.T) {
	tests := []statsTestCase{
		{
			name:      "Empty",
			cycles:    nil,
			wantCount: 0,
		},
		{
			name:      "AllOpenEnded",
			cycles:    []*v1.Cycle{makeCycle("c1", "u1", "2026-01-01", "")},
			wantCount: 0,
		},
		{
			name:        "SingleCycle28Days",
			cycles:      makeCyclesFromDays("u1", []int{28}),
			wantCount:   1,
			wantAvg:     28.0,
			wantMedian:  28.0,
			wantMin:     28,
			wantMax:     28,
			wantStdDev:  0.0,
			checkStdDev: true,
		},
		{
			name:        "ThreeCycles_28_30_26",
			cycles:      makeCyclesFromDays("u1", []int{28, 30, 26}),
			wantCount:   3,
			wantAvg:     28.0,
			wantMedian:  28.0,
			wantMin:     26,
			wantMax:     30,
			wantStdDev:  math.Sqrt(8.0 / 3.0),
			checkStdDev: true,
		},
		{
			name:       "EvenCount_28_30",
			cycles:     makeCyclesFromDays("u1", []int{28, 30}),
			wantCount:  2,
			wantMedian: 29.0,
		},
		{
			name: "ExcludesOutlierShortCycle_8days",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2026-01-01", "2026-01-28"), // 28 days
				makeCycle("c2", "u1", "2026-01-29", "2026-02-05"), // 8 days — outlier
				makeCycle("c3", "u1", "2026-02-06", "2026-03-07"), // 30 days
			},
			wantCount: 2,
			wantAvg:   29.0,
		},
		{
			name: "ExcludesOutlierLongCycle_99days",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2026-01-01", "2026-01-28"), // 28 days
				makeCycle("c2", "u1", "2026-01-29", "2026-05-07"), // 99 days — outlier
				makeCycle("c3", "u1", "2026-05-08", "2026-06-06"), // 30 days
			},
			wantCount: 2,
			wantAvg:   29.0,
		},
		{
			name: "AllOutliers_EmptyResult",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2026-01-01", "2026-01-05"), // 5 days
				makeCycle("c2", "u1", "2026-01-06", "2026-05-15"), // 130 days
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := rules.Stats(tt.cycles)

			if s.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", s.Count, tt.wantCount)
			}

			if tt.wantAvg > 0 && !approxEqual(s.Average, tt.wantAvg, 0.001) {
				t.Errorf("Average = %.4f, want %.4f", s.Average, tt.wantAvg)
			}

			if tt.wantMedian > 0 && !approxEqual(s.Median, tt.wantMedian, 0.001) {
				t.Errorf("Median = %.4f, want %.4f", s.Median, tt.wantMedian)
			}

			if tt.wantMin > 0 && s.Min != tt.wantMin {
				t.Errorf("Min = %d, want %d", s.Min, tt.wantMin)
			}

			if tt.wantMax > 0 && s.Max != tt.wantMax {
				t.Errorf("Max = %d, want %d", s.Max, tt.wantMax)
			}

			if tt.checkStdDev && !approxEqual(s.StdDev, tt.wantStdDev, 0.001) {
				t.Errorf("StdDev = %.4f, want %.4f", s.StdDev, tt.wantStdDev)
			}
		})
	}
}

// ============================================================================
// TestWindowStats — Table-driven tests for windowed statistics
// ============================================================================

type windowStatsTestCase struct {
	name       string
	cycles     []*v1.Cycle
	windowSize int
	wantCount  int
	wantAvg    float64
}

func TestWindowStats(t *testing.T) {
	tests := []windowStatsTestCase{
		{
			name: "LastN_3cycles_take2",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2025-01-01", "2025-01-20"), // 20 days — oldest
				makeCycle("c2", "u1", "2026-01-01", "2026-01-28"), // 28 days
				makeCycle("c3", "u1", "2026-01-29", "2026-02-27"), // 30 days — newest
			},
			windowSize: 2,
			wantCount:  2,
			wantAvg:    29.0, // Last 2: 28 and 30
		},
		{
			name: "FewerThanN_1cycle_requestwindow5",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
			},
			windowSize: 5,
			wantCount:  1,
		},
		{
			name: "IgnoresOpenEnded",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2026-01-01", "2026-01-28"),
				makeCycle("c2", "u1", "2026-01-29", ""), // open-ended
			},
			windowSize: 10,
			wantCount:  1,
		},
		{
			name: "ExcludesOutliers_lastwindow3_outlier6days",
			cycles: []*v1.Cycle{
				makeCycle("c1", "u1", "2025-01-01", "2025-01-28"), // 28 days
				makeCycle("c2", "u1", "2025-01-29", "2025-02-03"), // 6 days — outlier
				makeCycle("c3", "u1", "2026-01-01", "2026-01-30"), // 30 days
			},
			windowSize: 3,
			wantCount:  2, // Only 28 and 30 (outlier excluded)
			wantAvg:    29.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := rules.WindowStats(tt.cycles, tt.windowSize)

			if s.Count != tt.wantCount {
				t.Errorf("Count = %d, want %d", s.Count, tt.wantCount)
			}

			if tt.wantAvg > 0 && !approxEqual(s.Average, tt.wantAvg, 0.001) {
				t.Errorf("Average = %.4f, want %.4f", s.Average, tt.wantAvg)
			}
		})
	}
}
