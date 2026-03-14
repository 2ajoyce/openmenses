import React, { useState } from "react";
import { Card, CardContent, CardHeader, Link } from "framework7-react";
import type { MoodObservation } from "@gen/openmenses/v1/model_pb";
import { moodTypeLabel, moodIntensityLabel } from "../../lib/enums";
import { formatDateTime } from "../../lib/dates";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { client } from "../../lib/client";

interface MoodCardProps {
  observation: MoodObservation;
  onEdit?: (name: string) => void;
  onDeleted?: () => void;
}

export const MoodCard: React.FC<MoodCardProps> = ({
  observation,
  onEdit,
  onDeleted,
}) => {
  const [confirmDelete, setConfirmDelete] = useState(false);

  async function handleDelete() {
    try {
      await client.deleteMoodObservation({ name: observation.name });
      onDeleted?.();
    } catch (err) {
      console.error("Failed to delete mood observation:", err);
    }
    setConfirmDelete(false);
  }

  return (
    <>
      <Card>
        <CardHeader>
          <span>
            {moodTypeLabel(observation.mood)} —{" "}
            {moodIntensityLabel(observation.intensity)}
          </span>
          <div style={{ display: "flex", gap: "8px" }}>
            {onEdit && (
              <Link onClick={() => onEdit(observation.name)}>Edit</Link>
            )}
            <Link color="red" onClick={() => setConfirmDelete(true)}>
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
        message="Are you sure you want to delete this mood observation?"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(false)}
      />
    </>
  );
};
