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
    <Card>
      <CardHeader>
        <span>Medication — {medication.displayName}</span>
      </CardHeader>
      <CardContent>
        <p>{medicationCategoryLabel(medication.category)}</p>
        <p
          style={{
            color: medication.active ? "#4caf50" : "#9e9e9e",
            fontWeight: 500,
          }}
        >
          {medication.active ? "Active" : "Inactive"}
        </p>
        {medication.note && (
          <p
            style={{
              opacity: 0.7,
              overflow: "hidden",
              textOverflow: "ellipsis",
              display: "-webkit-box",
              WebkitLineClamp: 2,
              WebkitBoxOrient: "vertical",
            }}
          >
            {medication.note}
          </p>
        )}
      </CardContent>
    </Card>
  );
};
