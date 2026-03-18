package main

// regularScenario returns a scenario with 12 stable cycles, consistent symptom patterns,
// and high medication adherence. Produces stable CYCLE_LENGTH_PATTERN, SYMPTOM_PATTERN,
// and MEDICATION_ADHERENCE_PATTERN insights.
func regularScenario() *Scenario {
	return &Scenario{
		Name:                "regular-12",
		Description:         "12 stable cycles (28 days mean), consistent headache on day 12, high medication adherence",
		CycleCount:          12,
		CycleLengthMean:     28.0,
		CycleLengthStdDev:   1.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 0.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowHeavy, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{"Headache": {12}},
		MedicationNames:     []string{"Ibuprofen"},
		MedicationAdherence: map[string]float64{"Ibuprofen": 0.95},
		IncludeMood:         true,
	}
}

// irregularScenario returns a scenario with 8 cycles of highly variable lengths,
// producing an IRREGULAR classification in CYCLE_LENGTH_PATTERN.
func irregularScenario() *Scenario {
	return &Scenario{
		Name:                "irregular",
		Description:         "8 cycles with variable lengths (32 days mean, 7 day stddev) and variable bleed durations",
		CycleCount:          8,
		CycleLengthMean:     32.0,
		CycleLengthStdDev:   7.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 2.0,
		FlowPattern:         []FlowIntensity{FlowLight, FlowModerate, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         false,
	}
}

// shorteningScenario returns a scenario with 10 cycles whose lengths trend downward
// from 32 to 26 days, producing a SHORTENING CYCLE_LENGTH_PATTERN insight.
func shorteningScenario() *Scenario {
	return &Scenario{
		Name:                "shortening",
		Description:         "10 cycles trending from 32 → 26 days, produces SHORTENING pattern",
		CycleCount:          10,
		CycleLengthMean:     32.0,
		CycleLengthStdDev:   0.5,
		CycleLengthTrend:    -0.667,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 0.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowHeavy, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         false,
	}
}

// medicationGapsScenario returns a scenario with 6 cycles and a medication tracked
// at ~60% adherence, producing a LOW MEDICATION_ADHERENCE_PATTERN insight.
func medicationGapsScenario() *Scenario {
	return &Scenario{
		Name:                "medication-gaps",
		Description:         "6 cycles with medication at ~60% adherence, produces LOW adherence pattern",
		CycleCount:          6,
		CycleLengthMean:     28.0,
		CycleLengthStdDev:   1.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 0.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowHeavy, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{"Aspirin"},
		MedicationAdherence: map[string]float64{"Aspirin": 0.60},
		IncludeMood:         false,
	}
}

// minimalScenario returns a scenario with only 3 cycles and minimal data,
// useful for testing threshold behavior and data validation.
func minimalScenario() *Scenario {
	return &Scenario{
		Name:                "minimal",
		Description:         "3 cycles, minimal data for testing threshold behavior",
		CycleCount:          3,
		CycleLengthMean:     28.0,
		CycleLengthStdDev:   0.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 0.0,
		FlowPattern:         []FlowIntensity{FlowLight, FlowHeavy, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         false,
	}
}
