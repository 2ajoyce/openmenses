import React from "react";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import type { BiologicalCycleModel, PhaseEstimate } from "@gen/openmenses/v1/model_pb";
import { BleedingCard } from "../bleeding/BleedingCard";
import { SymptomCard } from "../symptom/SymptomCard";
import { MoodCard } from "../mood/MoodCard";
import { MedicationCard } from "../medication/MedicationCard";
import { MedicationEventCard } from "../medication/MedicationEventCard";
import { CycleCard } from "../cycle/CycleCard";
import { PhaseEstimateCard } from "../cycle/PhaseEstimateCard";
import { PredictionCard } from "../prediction/PredictionCard";
import { InsightCard } from "../insight/InsightCard";

interface TimelineItemProps {
  record: TimelineRecord;
  medicationNames?: Record<string, string>;
  biologicalCycleModel?: BiologicalCycleModel;
  groupedPhaseEstimates?: PhaseEstimate[];
  recordLookup?: Record<string, TimelineRecord>;
  onNavigateEdit: (path: string) => void;
  onDeleted: () => void;
}

export const TimelineItem: React.FC<TimelineItemProps> = ({
  record,
  medicationNames,
  biologicalCycleModel,
  groupedPhaseEstimates,
  recordLookup,
  onNavigateEdit,
  onDeleted,
}) => {
  switch (record.record.case) {
    case "bleedingObservation":
      return (
        <BleedingCard
          observation={record.record.value}
          onEdit={(name) =>
            onNavigateEdit(`/log/bleeding/?name=${encodeURIComponent(name)}`)
          }
          onDeleted={onDeleted}
        />
      );
    case "symptomObservation":
      return (
        <SymptomCard
          observation={record.record.value}
          onEdit={(name) =>
            onNavigateEdit(`/log/symptom/?name=${encodeURIComponent(name)}`)
          }
          onDeleted={onDeleted}
        />
      );
    case "moodObservation":
      return (
        <MoodCard
          observation={record.record.value}
          onEdit={(name) =>
            onNavigateEdit(`/log/mood/?name=${encodeURIComponent(name)}`)
          }
          onDeleted={onDeleted}
        />
      );
    case "medication":
      return <MedicationCard medication={record.record.value} />;
    case "medicationEvent":
      return (
        <MedicationEventCard
          event={record.record.value}
          {...(medicationNames?.[record.record.value.medicationId] != null
            ? { medicationName: medicationNames[record.record.value.medicationId] }
            : {})}
          onEdit={(name) =>
            onNavigateEdit(
              `/log/medication/?name=${encodeURIComponent(name)}`,
            )
          }
          onDeleted={onDeleted}
        />
      );
    case "cycle":
      return <CycleCard cycle={record.record.value} />;
    case "phaseEstimate":
      return (
        <PhaseEstimateCard
          estimates={groupedPhaseEstimates || [record.record.value]}
          {...(biologicalCycleModel != null && {
            biologicalCycleModel,
          })}
        />
      );
    case "prediction":
      return <PredictionCard prediction={record.record.value} />;
    case "insight":
      return <InsightCard insight={record.record.value} {...(recordLookup != null && { recordLookup })} />;
    default:
      return null;
  }
};
