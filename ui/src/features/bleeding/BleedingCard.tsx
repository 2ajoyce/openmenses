import React, { useState } from "react";
import { Card, CardContent, CardHeader, Link, f7 } from "framework7-react";
import type { BleedingObservation } from "@gen/openmenses/v1/model_pb";
import { bleedingFlowLabel } from "../../lib/enums";
import { formatDateTime } from "../../lib/dates";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { client } from "../../lib/client";

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
      f7.dialog.alert(
        err instanceof Error ? err.message : "Failed to delete observation",
        "Error",
      );
    }
    setConfirmDelete(false);
  }

  return (
    <>
      <div className="bleeding-card">
        <Card>
          <CardHeader>
            <div className="om-row">
              <span
                className="om-dot"
                data-flow={String(observation.flow)}
              />
              <span className="om-card-title">
                Bleeding — {bleedingFlowLabel(observation.flow)}
              </span>
            </div>
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
        message="Are you sure you want to delete this bleeding observation?"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(false)}
      />
    </>
  );
};
