import { describe, it, expect } from "vitest";
import {
  BleedingFlow,
  SymptomType,
  SymptomSeverity,
  MoodType,
  MoodIntensity,
  MedicationCategory,
  MedicationEventStatus,
  CyclePhase,
  ConfidenceLevel,
  CycleSource,
  BiologicalCycleModel,
  CycleRegularity,
  TrackingFocus,
  PredictionType,
  InsightType,
} from "@gen/openmenses/v1/model_pb";
import {
  bleedingFlowLabel,
  bleedingFlowOptions,
  symptomTypeLabel,
  symptomTypeOptions,
  symptomSeverityLabel,
  symptomSeverityOptions,
  moodTypeLabel,
  moodTypeOptions,
  moodIntensityLabel,
  moodIntensityOptions,
  medicationCategoryLabel,
  medicationCategoryOptions,
  medicationEventStatusLabel,
  medicationEventStatusOptions,
  cyclePhaseLabel,
  cyclePhaseOptions,
  suppressedCyclePhaseLabel,
  confidenceLevelLabel,
  confidenceLevelOptions,
  cycleSourceLabel,
  cycleSourceOptions,
  biologicalCycleModelLabel,
  biologicalCycleModelOptions,
  cycleRegularityLabel,
  cycleRegularityOptions,
  trackingFocusLabel,
  trackingFocusOptions,
  predictionTypeLabel,
  insightTypeLabel,
} from "../enums";

describe("bleedingFlowLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(bleedingFlowLabel(BleedingFlow.SPOTTING)).toBe("Spotting");
    expect(bleedingFlowLabel(BleedingFlow.LIGHT)).toBe("Light");
    expect(bleedingFlowLabel(BleedingFlow.MEDIUM)).toBe("Medium");
    expect(bleedingFlowLabel(BleedingFlow.HEAVY)).toBe("Heavy");
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(bleedingFlowLabel(BleedingFlow.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("bleedingFlowOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(bleedingFlowOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 4 options", () => {
    expect(bleedingFlowOptions).toHaveLength(4);
  });
});

describe("symptomTypeLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(symptomTypeLabel(SymptomType.CRAMPS)).toBe("Cramps");
    expect(symptomTypeLabel(SymptomType.HEADACHE)).toBe("Headache");
    expect(symptomTypeLabel(SymptomType.OTHER)).toBe("Other");
  });
});

describe("symptomTypeOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(symptomTypeOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 10 options", () => {
    expect(symptomTypeOptions).toHaveLength(10);
  });
});

describe("symptomSeverityLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(symptomSeverityLabel(SymptomSeverity.MINIMAL)).toBe("Minimal");
    expect(symptomSeverityLabel(SymptomSeverity.MILD)).toBe("Mild");
    expect(symptomSeverityLabel(SymptomSeverity.MODERATE)).toBe("Moderate");
    expect(symptomSeverityLabel(SymptomSeverity.SEVERE)).toBe("Severe");
  });
});

describe("symptomSeverityOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(symptomSeverityOptions.every((o) => o.value !== 0)).toBe(true);
  });
});

describe("moodTypeLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(moodTypeLabel(MoodType.CALM)).toBe("Calm");
    expect(moodTypeLabel(MoodType.HAPPY)).toBe("Happy");
    expect(moodTypeLabel(MoodType.SAD)).toBe("Sad");
  });
});

describe("moodTypeOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(moodTypeOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 7 options", () => {
    expect(moodTypeOptions).toHaveLength(7);
  });
});

describe("moodIntensityLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(moodIntensityLabel(MoodIntensity.LOW)).toBe("Low");
    expect(moodIntensityLabel(MoodIntensity.MEDIUM)).toBe("Medium");
    expect(moodIntensityLabel(MoodIntensity.HIGH)).toBe("High");
  });
});

describe("moodIntensityOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(moodIntensityOptions.every((o) => o.value !== 0)).toBe(true);
  });
});

describe("medicationCategoryLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(medicationCategoryLabel(MedicationCategory.BIRTH_CONTROL)).toBe(
      "Birth Control",
    );
    expect(medicationCategoryLabel(MedicationCategory.PAIN_RELIEF)).toBe(
      "Pain Relief",
    );
    expect(medicationCategoryLabel(MedicationCategory.OTHER)).toBe("Other");
  });
});

describe("medicationCategoryOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(medicationCategoryOptions.every((o) => o.value !== 0)).toBe(true);
  });
});

describe("medicationEventStatusLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(medicationEventStatusLabel(MedicationEventStatus.TAKEN)).toBe(
      "Taken",
    );
    expect(medicationEventStatusLabel(MedicationEventStatus.MISSED)).toBe(
      "Missed",
    );
    expect(medicationEventStatusLabel(MedicationEventStatus.SKIPPED)).toBe(
      "Skipped",
    );
  });
});

describe("medicationEventStatusOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(medicationEventStatusOptions.every((o) => o.value !== 0)).toBe(
      true,
    );
  });
});

describe("cyclePhaseLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(cyclePhaseLabel(CyclePhase.MENSTRUATION)).toBe("Menstruation");
    expect(cyclePhaseLabel(CyclePhase.FOLLICULAR)).toBe("Follicular");
    expect(cyclePhaseLabel(CyclePhase.OVULATION_WINDOW)).toBe(
      "Ovulation Window",
    );
    expect(cyclePhaseLabel(CyclePhase.LUTEAL)).toBe("Luteal");
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(cyclePhaseLabel(CyclePhase.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("cyclePhaseOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(cyclePhaseOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 5 options", () => {
    expect(cyclePhaseOptions).toHaveLength(5);
  });
});

describe("suppressedCyclePhaseLabel", () => {
  it("returns normal labels for menstruation, ovulation window, and luteal", () => {
    expect(suppressedCyclePhaseLabel(CyclePhase.MENSTRUATION)).toBe(
      "Menstruation",
    );
    expect(suppressedCyclePhaseLabel(CyclePhase.OVULATION_WINDOW)).toBe(
      "Ovulation Window",
    );
    expect(suppressedCyclePhaseLabel(CyclePhase.LUTEAL)).toBe("Luteal");
  });

  it("returns suppressed label for follicular phase", () => {
    expect(suppressedCyclePhaseLabel(CyclePhase.FOLLICULAR)).toBe(
      "Pill-free / Active pill days",
    );
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(suppressedCyclePhaseLabel(CyclePhase.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("confidenceLevelLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(confidenceLevelLabel(ConfidenceLevel.LOW)).toBe("Low");
    expect(confidenceLevelLabel(ConfidenceLevel.MEDIUM)).toBe("Medium");
    expect(confidenceLevelLabel(ConfidenceLevel.HIGH)).toBe("High");
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(confidenceLevelLabel(ConfidenceLevel.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("confidenceLevelOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(confidenceLevelOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 3 options", () => {
    expect(confidenceLevelOptions).toHaveLength(3);
  });
});

describe("cycleSourceLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(cycleSourceLabel(CycleSource.DERIVED_FROM_BLEEDING)).toBe(
      "Derived from bleeding",
    );
    expect(cycleSourceLabel(CycleSource.USER_CONFIRMED)).toBe("User confirmed");
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(cycleSourceLabel(CycleSource.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("cycleSourceOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(cycleSourceOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 2 options", () => {
    expect(cycleSourceOptions).toHaveLength(2);
  });
});

describe("biologicalCycleModelLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(biologicalCycleModelLabel(BiologicalCycleModel.OVULATORY)).toBe(
      "Ovulatory",
    );
    expect(
      biologicalCycleModelLabel(BiologicalCycleModel.HORMONALLY_SUPPRESSED),
    ).toBe("Hormonally Suppressed");
    expect(biologicalCycleModelLabel(BiologicalCycleModel.IRREGULAR)).toBe(
      "Irregular",
    );
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(biologicalCycleModelLabel(BiologicalCycleModel.UNSPECIFIED)).toBe(
      "Unknown",
    );
  });
});

describe("biologicalCycleModelOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(biologicalCycleModelOptions.every((o) => o.value !== 0)).toBe(
      true,
    );
  });

  it("has 3 options", () => {
    expect(biologicalCycleModelOptions).toHaveLength(3);
  });
});

describe("cycleRegularityLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(cycleRegularityLabel(CycleRegularity.REGULAR)).toBe("Regular");
    expect(cycleRegularityLabel(CycleRegularity.SOMEWHAT_IRREGULAR)).toBe(
      "Somewhat Irregular",
    );
    expect(cycleRegularityLabel(CycleRegularity.VERY_IRREGULAR)).toBe(
      "Very Irregular",
    );
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(cycleRegularityLabel(CycleRegularity.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("cycleRegularityOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(cycleRegularityOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 4 options", () => {
    expect(cycleRegularityOptions).toHaveLength(4);
  });
});

describe("trackingFocusLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(trackingFocusLabel(TrackingFocus.BLEEDING)).toBe("Bleeding");
    expect(trackingFocusLabel(TrackingFocus.SYMPTOMS)).toBe("Symptoms");
    expect(trackingFocusLabel(TrackingFocus.MOOD)).toBe("Mood");
    expect(trackingFocusLabel(TrackingFocus.MEDICATION)).toBe("Medication");
    expect(trackingFocusLabel(TrackingFocus.CYCLE_PREDICTION)).toBe(
      "Cycle Prediction",
    );
    expect(trackingFocusLabel(TrackingFocus.PATTERN_ANALYSIS)).toBe(
      "Pattern Analysis",
    );
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(trackingFocusLabel(TrackingFocus.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("trackingFocusOptions", () => {
  it("excludes UNSPECIFIED", () => {
    expect(trackingFocusOptions.every((o) => o.value !== 0)).toBe(true);
  });

  it("has 6 options", () => {
    expect(trackingFocusOptions).toHaveLength(6);
  });
});

describe("predictionTypeLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(predictionTypeLabel(PredictionType.NEXT_BLEED)).toBe("Next Period");
    expect(predictionTypeLabel(PredictionType.PMS_WINDOW)).toBe("PMS Window");
    expect(predictionTypeLabel(PredictionType.OVULATION_WINDOW)).toBe("Ovulation Window");
    expect(predictionTypeLabel(PredictionType.SYMPTOM_WINDOW)).toBe("Symptom Window");
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(predictionTypeLabel(PredictionType.UNSPECIFIED)).toBe("Unknown");
  });
});

describe("insightTypeLabel", () => {
  it("returns label for each non-UNSPECIFIED value", () => {
    expect(insightTypeLabel(InsightType.CYCLE_LENGTH_PATTERN)).toBe("Cycle Length Trend");
    expect(insightTypeLabel(InsightType.SYMPTOM_PATTERN)).toBe("Symptom Pattern");
    expect(insightTypeLabel(InsightType.MEDICATION_ADHERENCE_PATTERN)).toBe("Medication Adherence");
    expect(insightTypeLabel(InsightType.BLEEDING_PATTERN)).toBe("Bleeding Pattern");
  });

  it("returns Unknown for UNSPECIFIED", () => {
    expect(insightTypeLabel(InsightType.UNSPECIFIED)).toBe("Unknown");
  });
});
