import {
  BiologicalCycleModel,
  BleedingFlow,
  ConfidenceLevel,
  ContraceptionType,
  CyclePhase,
  CycleRegularity,
  CycleSource,
  InsightType,
  MedicationCategory,
  MedicationEventStatus,
  MoodIntensity,
  MoodType,
  PredictionType,
  ReproductiveGoal,
  SymptomSeverity,
  SymptomType,
  TrackingFocus,
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

export function medicationCategoryLabel(category: MedicationCategory): string {
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

export const cyclePhaseLabels: Record<number, string> = {
  [CyclePhase.MENSTRUATION]: "Menstruation",
  [CyclePhase.FOLLICULAR]: "Follicular",
  [CyclePhase.OVULATION_WINDOW]: "Ovulation Window",
  [CyclePhase.LUTEAL]: "Luteal",
  [CyclePhase.UNKNOWN]: "Unknown",
};

export const cyclePhaseOptions = [
  { value: CyclePhase.MENSTRUATION, label: "Menstruation" },
  { value: CyclePhase.FOLLICULAR, label: "Follicular" },
  { value: CyclePhase.OVULATION_WINDOW, label: "Ovulation Window" },
  { value: CyclePhase.LUTEAL, label: "Luteal" },
  { value: CyclePhase.UNKNOWN, label: "Unknown" },
];

export function cyclePhaseLabel(phase: CyclePhase): string {
  return cyclePhaseLabels[phase] ?? "Unknown";
}

export const suppressedCyclePhaseLabels: Record<number, string> = {
  [CyclePhase.MENSTRUATION]: "Menstruation",
  [CyclePhase.FOLLICULAR]: "Pill-free / Active pill days",
  [CyclePhase.OVULATION_WINDOW]: "Ovulation Window",
  [CyclePhase.LUTEAL]: "Luteal",
  [CyclePhase.UNKNOWN]: "Unknown",
};

export function suppressedCyclePhaseLabel(phase: CyclePhase): string {
  return suppressedCyclePhaseLabels[phase] ?? "Unknown";
}

export const confidenceLevelLabels: Record<number, string> = {
  [ConfidenceLevel.LOW]: "Low",
  [ConfidenceLevel.MEDIUM]: "Medium",
  [ConfidenceLevel.HIGH]: "High",
};

export const confidenceLevelOptions = [
  { value: ConfidenceLevel.LOW, label: "Low" },
  { value: ConfidenceLevel.MEDIUM, label: "Medium" },
  { value: ConfidenceLevel.HIGH, label: "High" },
];

export function confidenceLevelLabel(level: ConfidenceLevel): string {
  return confidenceLevelLabels[level] ?? "Unknown";
}

export const cycleSourceLabels: Record<number, string> = {
  [CycleSource.DERIVED_FROM_BLEEDING]: "Derived from bleeding",
  [CycleSource.USER_CONFIRMED]: "User confirmed",
};

export const cycleSourceOptions = [
  { value: CycleSource.DERIVED_FROM_BLEEDING, label: "Derived from bleeding" },
  { value: CycleSource.USER_CONFIRMED, label: "User confirmed" },
];

export function cycleSourceLabel(source: CycleSource): string {
  return cycleSourceLabels[source] ?? "Unknown";
}

export const contraceptionTypeLabels: Record<number, string> = {
  [ContraceptionType.NONE]: "None",
  [ContraceptionType.BARRIER]: "Barrier",
  [ContraceptionType.HORMONAL_PILL]: "Hormonal Pill",
  [ContraceptionType.HORMONAL_PATCH]: "Hormonal Patch",
  [ContraceptionType.HORMONAL_RING]: "Hormonal Ring",
  [ContraceptionType.HORMONAL_SHOT]: "Hormonal Shot",
  [ContraceptionType.HORMONAL_IMPLANT]: "Hormonal Implant",
  [ContraceptionType.HORMONAL_IUD]: "Hormonal IUD",
  [ContraceptionType.COPPER_IUD]: "Copper IUD",
  [ContraceptionType.OTHER]: "Other",
};

export const contraceptionTypeOptions = [
  { value: ContraceptionType.NONE, label: "None" },
  { value: ContraceptionType.BARRIER, label: "Barrier" },
  { value: ContraceptionType.HORMONAL_PILL, label: "Hormonal Pill" },
  { value: ContraceptionType.HORMONAL_PATCH, label: "Hormonal Patch" },
  { value: ContraceptionType.HORMONAL_RING, label: "Hormonal Ring" },
  { value: ContraceptionType.HORMONAL_SHOT, label: "Hormonal Shot" },
  { value: ContraceptionType.HORMONAL_IMPLANT, label: "Hormonal Implant" },
  { value: ContraceptionType.HORMONAL_IUD, label: "Hormonal IUD" },
  { value: ContraceptionType.COPPER_IUD, label: "Copper IUD" },
  { value: ContraceptionType.OTHER, label: "Other" },
];

export function contraceptionTypeLabel(type: ContraceptionType): string {
  return contraceptionTypeLabels[type] ?? "Unknown";
}

export const reproductiveGoalLabels: Record<number, string> = {
  [ReproductiveGoal.TRYING_TO_CONCEIVE]: "Trying to Conceive",
  [ReproductiveGoal.AVOID_PREGNANCY]: "Avoid Pregnancy",
  [ReproductiveGoal.PREGNANCY_IRRELEVANT]: "Pregnancy Irrelevant",
  [ReproductiveGoal.NOT_TRACKING_FERTILITY]: "Not Tracking Fertility",
};

export const reproductiveGoalOptions = [
  { value: ReproductiveGoal.TRYING_TO_CONCEIVE, label: "Trying to Conceive" },
  { value: ReproductiveGoal.AVOID_PREGNANCY, label: "Avoid Pregnancy" },
  {
    value: ReproductiveGoal.PREGNANCY_IRRELEVANT,
    label: "Pregnancy Irrelevant",
  },
  {
    value: ReproductiveGoal.NOT_TRACKING_FERTILITY,
    label: "Not Tracking Fertility",
  },
];

export function reproductiveGoalLabel(goal: ReproductiveGoal): string {
  return reproductiveGoalLabels[goal] ?? "Unknown";
}

export const biologicalCycleModelLabels: Record<number, string> = {
  [BiologicalCycleModel.OVULATORY]: "Ovulatory",
  [BiologicalCycleModel.HORMONALLY_SUPPRESSED]: "Hormonally Suppressed",
  [BiologicalCycleModel.IRREGULAR]: "Irregular",
};

export const biologicalCycleModelOptions = [
  { value: BiologicalCycleModel.OVULATORY, label: "Ovulatory" },
  {
    value: BiologicalCycleModel.HORMONALLY_SUPPRESSED,
    label: "Hormonally Suppressed",
  },
  { value: BiologicalCycleModel.IRREGULAR, label: "Irregular" },
];

export function biologicalCycleModelLabel(model: BiologicalCycleModel): string {
  return biologicalCycleModelLabels[model] ?? "Unknown";
}

export const cycleRegularityLabels: Record<number, string> = {
  [CycleRegularity.REGULAR]: "Regular",
  [CycleRegularity.SOMEWHAT_IRREGULAR]: "Somewhat Irregular",
  [CycleRegularity.VERY_IRREGULAR]: "Very Irregular",
  [CycleRegularity.UNKNOWN]: "Unknown",
};

export const cycleRegularityOptions = [
  { value: CycleRegularity.REGULAR, label: "Regular" },
  { value: CycleRegularity.SOMEWHAT_IRREGULAR, label: "Somewhat Irregular" },
  { value: CycleRegularity.VERY_IRREGULAR, label: "Very Irregular" },
  { value: CycleRegularity.UNKNOWN, label: "Unknown" },
];

export function cycleRegularityLabel(regularity: CycleRegularity): string {
  return cycleRegularityLabels[regularity] ?? "Unknown";
}

export const trackingFocusLabels: Record<number, string> = {
  [TrackingFocus.BLEEDING]: "Bleeding",
  [TrackingFocus.SYMPTOMS]: "Symptoms",
  [TrackingFocus.MOOD]: "Mood",
  [TrackingFocus.MEDICATION]: "Medication",
  [TrackingFocus.CYCLE_PREDICTION]: "Cycle Prediction",
  [TrackingFocus.PATTERN_ANALYSIS]: "Pattern Analysis",
};

export const trackingFocusOptions = [
  { value: TrackingFocus.BLEEDING, label: "Bleeding" },
  { value: TrackingFocus.SYMPTOMS, label: "Symptoms" },
  { value: TrackingFocus.MOOD, label: "Mood" },
  { value: TrackingFocus.MEDICATION, label: "Medication" },
  { value: TrackingFocus.CYCLE_PREDICTION, label: "Cycle Prediction" },
  { value: TrackingFocus.PATTERN_ANALYSIS, label: "Pattern Analysis" },
];

export function trackingFocusLabel(focus: TrackingFocus): string {
  return trackingFocusLabels[focus] ?? "Unknown";
}

export const predictionTypeLabels: Record<number, string> = {
  [PredictionType.NEXT_BLEED]: "Next Period",
  [PredictionType.PMS_WINDOW]: "PMS Window",
  [PredictionType.OVULATION_WINDOW]: "Ovulation Window",
  [PredictionType.SYMPTOM_WINDOW]: "Symptom Window",
};

export function predictionTypeLabel(type: PredictionType): string {
  return predictionTypeLabels[type] ?? "Unknown";
}

export const insightTypeLabels: Record<number, string> = {
  [InsightType.CYCLE_LENGTH_PATTERN]: "Cycle Length Trend",
  [InsightType.SYMPTOM_PATTERN]: "Symptom Pattern",
  [InsightType.MEDICATION_ADHERENCE_PATTERN]: "Medication Adherence",
  [InsightType.BLEEDING_PATTERN]: "Bleeding Pattern",
};

export function insightTypeLabel(type: InsightType): string {
  return insightTypeLabels[type] ?? "Unknown";
}
