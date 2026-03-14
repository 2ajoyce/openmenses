import React, { useState } from "react";
import { Card, CardContent, CardHeader, Link } from "framework7-react";
import type { BleedingObservation } from "@gen/openmenses/v1/model_pb";
import { bleedingFlowLabel } from "../../lib/enums";
import { formatDateTime } from "../../lib/dates";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { client } from "../../lib/client";

const flowColors: Record<number, string> = {
  1: "#ffb6c1",
  2: "#ff8da1",
  3: "#e74c6f",
  4: "#c62828",
};

interface BleedingCardProps {
  observation: BleedingObservation;
  onEdit?: (name: string) => void;
  onDeleted?: () => void;
}

export const BleedingCard: React.FC<BleedingCardProps> = ({
  observation,
  onEdit,
  onDeleted,
}) => {
  const [confirmDelete, setConfirmDelete] = useState(false);

  async function handleDelete() {
    try {
      await client.deleteBleedingObservation({ name: observation.name });
      onDeleted?.();
    } catch (err) {
      console.error("Failed to delete bleeding observation:", err);
    }
    setConfirmDelete(false);
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
            <span
              style={{
                width: 12,
                height: 12,
                borderRadius: "50%",
                backgroundColor: flowColors[observation.flow] ?? "#ccc",
                display: "inline-block",
              }}
            />
            <span>Bleeding — {bleedingFlowLabel(observation.flow)}</span>
          </div>
          <div style={{ display: "flex", gap: "8px" }}>
            {onEdit && (
              <Link onClick={() => onEdit(observation.name)}>Edit</Link>
            )}
            <Link
              color="red"
              onClick={() => setConfirmDelete(true)}
            >
              Delete
            </Link>
          </div>
        </CardHeader>
        <CardContent>
          {observation.timestamp && (
            <p>{formatDateTime(observation.timestamp)}</p>
          )}
          {observation.note && (
            <p style={{ opacity: 0.7 }}>{observation.note}</p>
          )}
        </CardContent>
      </Card>
      <ConfirmDialog
        open={confirmDelete}
        title="Delete Observation"
        message="Are you sure you want to delete this bleeding observation?"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(false)}
      />
    </>
  );
};
