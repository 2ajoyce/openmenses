import React from "react";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import { BleedingCard } from "../bleeding/BleedingCard";
import { SymptomCard } from "../symptom/SymptomCard";
import { MoodCard } from "../mood/MoodCard";
import { MedicationCard } from "../medication/MedicationCard";
import { MedicationEventCard } from "../medication/MedicationEventCard";

interface TimelineItemProps {
  record: TimelineRecord;
  onNavigateEdit: (path: string) => void;
  onDeleted: () => void;
}

export const TimelineItem: React.FC<TimelineItemProps> = ({
  record,
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
          onEdit={(name) =>
            onNavigateEdit(
              `/log/medication/?name=${encodeURIComponent(name)}`,
            )
          }
          onDeleted={onDeleted}
        />
      );
    default:
      return null;
  }
};
