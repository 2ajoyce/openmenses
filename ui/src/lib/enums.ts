import {
  BleedingFlow,
  SymptomType,
  SymptomSeverity,
  MoodType,
  MoodIntensity,
  MedicationCategory,
  MedicationEventStatus,
} from "@gen/openmenses/v1/model_pb";

export const bleedingFlowLabels: Record<number, string> = {
  [BleedingFlow.SPOTTING]: "Spotting",
  [BleedingFlow.LIGHT]: "Light",
  [BleedingFlow.MEDIUM]: "Medium",
  [BleedingFlow.HEAVY]: "Heavy",
};

export const bleedingFlowOptions = [
  { value: BleedingFlow.SPOTTING, label: "Spotting" },
  { value: BleedingFlow.LIGHT, label: "Light" },
  { value: BleedingFlow.MEDIUM, label: "Medium" },
  { value: BleedingFlow.HEAVY, label: "Heavy" },
];

export function bleedingFlowLabel(flow: BleedingFlow): string {
  return bleedingFlowLabels[flow] ?? "Unknown";
}

export const symptomTypeLabels: Record<number, string> = {
  [SymptomType.CRAMPS]: "Cramps",
  [SymptomType.HEADACHE]: "Headache",
  [SymptomType.MIGRAINE]: "Migraine",
  [SymptomType.BLOATING]: "Bloating",
  [SymptomType.BREAST_TENDERNESS]: "Breast Tenderness",
  [SymptomType.FATIGUE]: "Fatigue",
  [SymptomType.NAUSEA]: "Nausea",
  [SymptomType.ACNE]: "Acne",
  [SymptomType.BACK_PAIN]: "Back Pain",
  [SymptomType.OTHER]: "Other",
};

export const symptomTypeOptions = [
  { value: SymptomType.CRAMPS, label: "Cramps" },
  { value: SymptomType.HEADACHE, label: "Headache" },
  { value: SymptomType.MIGRAINE, label: "Migraine" },
  { value: SymptomType.BLOATING, label: "Bloating" },
  { value: SymptomType.BREAST_TENDERNESS, label: "Breast Tenderness" },
  { value: SymptomType.FATIGUE, label: "Fatigue" },
  { value: SymptomType.NAUSEA, label: "Nausea" },
  { value: SymptomType.ACNE, label: "Acne" },
  { value: SymptomType.BACK_PAIN, label: "Back Pain" },
  { value: SymptomType.OTHER, label: "Other" },
];

export function symptomTypeLabel(type: SymptomType): string {
  return symptomTypeLabels[type] ?? "Unknown";
}

export const symptomSeverityLabels: Record<number, string> = {
  [SymptomSeverity.MINIMAL]: "Minimal",
  [SymptomSeverity.MILD]: "Mild",
  [SymptomSeverity.MODERATE]: "Moderate",
  [SymptomSeverity.SEVERE]: "Severe",
};

export const symptomSeverityOptions = [
  { value: SymptomSeverity.MINIMAL, label: "Minimal" },
  { value: SymptomSeverity.MILD, label: "Mild" },
  { value: SymptomSeverity.MODERATE, label: "Moderate" },
  { value: SymptomSeverity.SEVERE, label: "Severe" },
];

export function symptomSeverityLabel(severity: SymptomSeverity): string {
  return symptomSeverityLabels[severity] ?? "Unknown";
}

export const moodTypeLabels: Record<number, string> = {
  [MoodType.CALM]: "Calm",
  [MoodType.HAPPY]: "Happy",
  [MoodType.IRRITABLE]: "Irritable",
  [MoodType.ANXIOUS]: "Anxious",
  [MoodType.SAD]: "Sad",
  [MoodType.EMOTIONALLY_FLAT]: "Emotionally Flat",
  [MoodType.OTHER]: "Other",
};

export const moodTypeOptions = [
  { value: MoodType.CALM, label: "Calm" },
  { value: MoodType.HAPPY, label: "Happy" },
  { value: MoodType.IRRITABLE, label: "Irritable" },
  { value: MoodType.ANXIOUS, label: "Anxious" },
  { value: MoodType.SAD, label: "Sad" },
  { value: MoodType.EMOTIONALLY_FLAT, label: "Emotionally Flat" },
  { value: MoodType.OTHER, label: "Other" },
];

export function moodTypeLabel(type: MoodType): string {
  return moodTypeLabels[type] ?? "Unknown";
}

export const moodIntensityLabels: Record<number, string> = {
  [MoodIntensity.LOW]: "Low",
  [MoodIntensity.MEDIUM]: "Medium",
  [MoodIntensity.HIGH]: "High",
};

export const moodIntensityOptions = [
  { value: MoodIntensity.LOW, label: "Low" },
  { value: MoodIntensity.MEDIUM, label: "Medium" },
  { value: MoodIntensity.HIGH, label: "High" },
];

export function moodIntensityLabel(intensity: MoodIntensity): string {
  return moodIntensityLabels[intensity] ?? "Unknown";
}

export const medicationCategoryLabels: Record<number, string> = {
  [MedicationCategory.BIRTH_CONTROL]: "Birth Control",
  [MedicationCategory.PAIN_RELIEF]: "Pain Relief",
  [MedicationCategory.MIGRAINE]: "Migraine",
  [MedicationCategory.HORMONE_THERAPY]: "Hormone Therapy",
  [MedicationCategory.SUPPLEMENT]: "Supplement",
  [MedicationCategory.OTHER]: "Other",
};

export const medicationCategoryOptions = [
  { value: MedicationCategory.BIRTH_CONTROL, label: "Birth Control" },
  { value: MedicationCategory.PAIN_RELIEF, label: "Pain Relief" },
  { value: MedicationCategory.MIGRAINE, label: "Migraine" },
  { value: MedicationCategory.HORMONE_THERAPY, label: "Hormone Therapy" },
  { value: MedicationCategory.SUPPLEMENT, label: "Supplement" },
  { value: MedicationCategory.OTHER, label: "Other" },
];

export function medicationCategoryLabel(
  category: MedicationCategory,
): string {
  return medicationCategoryLabels[category] ?? "Unknown";
}

export const medicationEventStatusLabels: Record<number, string> = {
  [MedicationEventStatus.TAKEN]: "Taken",
  [MedicationEventStatus.MISSED]: "Missed",
  [MedicationEventStatus.SKIPPED]: "Skipped",
};

export const medicationEventStatusOptions = [
  { value: MedicationEventStatus.TAKEN, label: "Taken" },
  { value: MedicationEventStatus.MISSED, label: "Missed" },
  { value: MedicationEventStatus.SKIPPED, label: "Skipped" },
];

export function medicationEventStatusLabel(
  status: MedicationEventStatus,
): string {
  return medicationEventStatusLabels[status] ?? "Unknown";
}
