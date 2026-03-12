package memory

import (
	"sort"

	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

func sortByTimestamp(items []*v1.BleedingObservation) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetTimestamp().GetValue() < items[j].GetTimestamp().GetValue()
	})
}

func sortSymptomsByTimestamp(items []*v1.SymptomObservation) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetTimestamp().GetValue() < items[j].GetTimestamp().GetValue()
	})
}

func sortMoodsByTimestamp(items []*v1.MoodObservation) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetTimestamp().GetValue() < items[j].GetTimestamp().GetValue()
	})
}

func sortMedicationsByID(items []*v1.Medication) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}

func sortMedicationEventsByTimestamp(items []*v1.MedicationEvent) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetTimestamp().GetValue() < items[j].GetTimestamp().GetValue()
	})
}

func sortCyclesByStartDate(items []*v1.Cycle) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetStartDate().GetValue() < items[j].GetStartDate().GetValue()
	})
}

func sortPhaseEstimatesByDate(items []*v1.PhaseEstimate) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetDate().GetValue() < items[j].GetDate().GetValue()
	})
}

func sortPredictionsByStartDate(items []*v1.Prediction) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetPredictedStartDate().GetValue() < items[j].GetPredictedStartDate().GetValue()
	})
}

func sortInsightsByID(items []*v1.Insight) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
}
