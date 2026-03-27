package main

import (
	v1 "github.com/2ajoyce/openmenses/gen/go/openmenses/v1"
)

// regularScenario returns a scenario with 12 stable cycles, consistent symptom patterns,
// and high medication adherence. Produces stable CYCLE_LENGTH_PATTERN, SYMPTOM_PATTERN,
// and MEDICATION_ADHERENCE_PATTERN insights.
// Persona: Emily (Ovulatory, Regular).
func regularScenario() *Scenario {
	return &Scenario{
		HumanName:           "Emily",
		Name:                "regular-12",
		Description:         "12 stable cycles (28 days mean), consistent headache on day 12, high medication adherence",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
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
// Persona: Hannah (Irregular cycle model, Regular regularity).
func irregularScenario() *Scenario {
	return &Scenario{
		HumanName:           "Hannah",
		Name:                "irregular",
		Description:         "8 cycles with variable lengths (32 days mean, 7 day stddev) and variable bleed durations",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
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

// jessicaScenario returns a scenario with 10 cycles of somewhat irregular length.
// Produces medium confidence phase estimation.
// Persona: Jessica (Ovulatory, Somewhat Irregular).
func beatrizScenario() *Scenario {
	return &Scenario{
		HumanName:           "Jessica",
		Name:                "ovulatory-somewhat-irregular",
		Description:         "10 cycles (30 days mean, 3 day stddev), somewhat irregular ovulatory",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR,
		CycleCount:          10,
		CycleLengthMean:     30.0,
		CycleLengthStdDev:   3.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 1.0,
		FlowPattern:         []FlowIntensity{FlowLight, FlowHeavy, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{"Cramps": {1, 2}},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         true,
	}
}

// ashleyScenario returns a scenario with 8 cycles of highly variable length.
// Produces low confidence phase estimation and no ovulation predictions.
// Persona: Ashley (Ovulatory, Very Irregular).
func chiomaScenario() *Scenario {
	return &Scenario{
		HumanName:           "Ashley",
		Name:                "ovulatory-very-irregular",
		Description:         "8 cycles (29 days mean, 6 day stddev), very irregular ovulatory",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR,
		CycleCount:          8,
		CycleLengthMean:     29.0,
		CycleLengthStdDev:   6.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 1.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowModerate, FlowHeavy, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         false,
	}
}

// sophieScenario returns a scenario with 6 cycles of stable length but unknown regularity
// (typical of a new user without enough history to detect patterns).
// Produces high confidence phase estimation but no ovulation predictions.
// Persona: Sophie (Ovulatory, Unknown regularity).
func diyaScenario() *Scenario {
	return &Scenario{
		HumanName:           "Sophie",
		Name:                "ovulatory-unknown",
		Description:         "6 cycles (28 days mean), ovulatory with unknown regularity (new user)",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_OVULATORY,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_UNKNOWN,
		CycleCount:          6,
		CycleLengthMean:     28.0,
		CycleLengthStdDev:   1.5,
		BleedDurationMean:   4.0,
		BleedDurationStdDev: 0.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowModerate, FlowHeavy, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         false,
	}
}

// lauraScenario returns a scenario with 10 cycles of stable length on hormonal suppression.
// Produces high confidence phase estimation but no ovulation predictions.
// Persona: Laura (Hormonally Suppressed, Regular).
func elenaScenario() *Scenario {
	return &Scenario{
		HumanName:           "Laura",
		Name:                "hormonal-regular",
		Description:         "10 cycles (28 days mean), hormonally suppressed (pill), regular withdrawal bleeds",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_REGULAR,
		CycleCount:          10,
		CycleLengthMean:     28.0,
		CycleLengthStdDev:   0.5,
		BleedDurationMean:   4.0,
		BleedDurationStdDev: 0.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowModerate, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{"Oral Contraceptive"},
		MedicationAdherence: map[string]float64{"Oral Contraceptive": 0.98},
		IncludeMood:         true,
	}
}

// emmaScenario returns a scenario with 8 cycles on hormonal suppression with occasional gaps.
// Produces medium confidence phase estimation and no ovulation predictions.
// Persona: Emma (Hormonally Suppressed, Somewhat Irregular).
func fatouScenario() *Scenario {
	return &Scenario{
		HumanName:           "Emma",
		Name:                "hormonal-somewhat-irregular",
		Description:         "8 cycles (28 days mean, 2 day stddev), hormonal with occasional missed pills",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_SOMEWHAT_IRREGULAR,
		CycleCount:          8,
		CycleLengthMean:     28.0,
		CycleLengthStdDev:   2.0,
		BleedDurationMean:   4.0,
		BleedDurationStdDev: 1.0,
		FlowPattern:         []FlowIntensity{FlowLight, FlowModerate, FlowLight},
		SymptomPatterns:     map[string][]int{"Headache": {22}},
		MedicationNames:     []string{"Oral Contraceptive"},
		MedicationAdherence: map[string]float64{"Oral Contraceptive": 0.85},
		IncludeMood:         false,
	}
}

// camilleScenario returns a scenario with 6 cycles on hormonal suppression with breakthrough bleeding.
// Produces low confidence phase estimation and no ovulation predictions.
// Persona: Camille (Hormonally Suppressed, Very Irregular).
func gretaScenario() *Scenario {
	return &Scenario{
		HumanName:           "Camille",
		Name:                "hormonal-very-irregular",
		Description:         "6 cycles (35 days mean, 8 day stddev), hormonal implant with breakthrough bleeding",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_HORMONALLY_SUPPRESSED,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR,
		CycleCount:          6,
		CycleLengthMean:     35.0,
		CycleLengthStdDev:   8.0,
		BleedDurationMean:   3.0,
		BleedDurationStdDev: 1.5,
		FlowPattern:         []FlowIntensity{FlowLight, FlowLight, FlowModerate},
		SymptomPatterns:     map[string][]int{},
		MedicationNames:     []string{},
		MedicationAdherence: map[string]float64{},
		IncludeMood:         false,
	}
}

// priyaScenario returns a scenario with 6 cycles of highly variable length on the irregular model.
// Produces low confidence phase estimation and no ovulation predictions.
// Persona: Priya (Irregular, Very Irregular).
func ingridScenario() *Scenario {
	return &Scenario{
		HumanName:           "Priya",
		Name:                "irregular-very-irregular",
		Description:         "6 cycles (34 days mean, 9 day stddev), irregular cycle model with very irregular regularity",
		BiologicalCycle:     v1.BiologicalCycleModel_BIOLOGICAL_CYCLE_MODEL_IRREGULAR,
		CycleRegularity:     v1.CycleRegularity_CYCLE_REGULARITY_VERY_IRREGULAR,
		CycleCount:          6,
		CycleLengthMean:     34.0,
		CycleLengthStdDev:   9.0,
		BleedDurationMean:   5.0,
		BleedDurationStdDev: 2.0,
		FlowPattern:         []FlowIntensity{FlowLight, FlowModerate, FlowHeavy, FlowHeavy, FlowModerate, FlowLight},
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
