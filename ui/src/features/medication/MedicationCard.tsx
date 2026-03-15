import React from "react";
import { Card, CardContent, CardHeader } from "framework7-react";
import type { Medication } from "@gen/openmenses/v1/model_pb";
import { medicationCategoryLabel } from "../../lib/enums";

interface MedicationCardProps {
  medication: Medication;
}

export const MedicationCard: React.FC<MedicationCardProps> = ({
  medication,
}) => {
  return (
    <div className="medication-card">
      <Card>
        <CardHeader>
          <span className="om-card-title">
            Medication — {medication.displayName}
          </span>
        </CardHeader>
        <CardContent>
          <p className="om-card-timestamp">
            {medicationCategoryLabel(medication.category)}
          </p>
          <p
            className={
              medication.active ? "medication-status-active" : "om-muted"
            }
          >
            {medication.active ? "Active" : "Inactive"}
          </p>
          {medication.note && (
            <p className="om-card-notes om-muted om-truncate-2">
              {medication.note}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
