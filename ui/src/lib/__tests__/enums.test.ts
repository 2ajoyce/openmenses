import { describe, it, expect } from "vitest";
import {
  BleedingFlow,
  SymptomType,
  SymptomSeverity,
  MoodType,
  MoodIntensity,
  MedicationCategory,
  MedicationEventStatus,
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
