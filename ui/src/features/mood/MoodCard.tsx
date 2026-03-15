import React, { useState } from "react";
import { Card, CardContent, CardHeader, Link, f7 } from "framework7-react";
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
      f7.dialog.alert(
        err instanceof Error ? err.message : "Failed to delete observation",
        "Error",
      );
    }
    setConfirmDelete(false);
  }

  return (
    <>
      <div className="mood-card">
        <Card>
          <CardHeader>
            <span className="om-card-title">
              {moodTypeLabel(observation.mood)} —{" "}
              {moodIntensityLabel(observation.intensity)}
            </span>
            <div className="om-row">
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
              <p className="om-card-timestamp">
                {formatDateTime(observation.timestamp)}
              </p>
            )}
            {observation.note && (
              <p className="om-card-notes om-muted om-truncate-2">
                {observation.note}
              </p>
            )}
          </CardContent>
        </Card>
      </div>
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
