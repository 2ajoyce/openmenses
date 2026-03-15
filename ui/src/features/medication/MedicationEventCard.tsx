import React, { useState } from "react";
import { Card, CardContent, CardHeader, Link, f7 } from "framework7-react";
import type { MedicationEvent } from "@gen/openmenses/v1/model_pb";
import { medicationEventStatusLabel } from "../../lib/enums";
import { formatDateTime } from "../../lib/dates";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { client } from "../../lib/client";

interface MedicationEventCardProps {
  event: MedicationEvent;
  medicationName?: string;
  onEdit?: (name: string) => void;
  onDeleted?: () => void;
}

export const MedicationEventCard: React.FC<MedicationEventCardProps> = ({
  event,
  medicationName,
  onEdit,
  onDeleted,
}) => {
  const [confirmDelete, setConfirmDelete] = useState(false);

  async function handleDelete() {
    try {
      await client.deleteMedicationEvent({ name: event.name });
      onDeleted?.();
    } catch (err) {
      console.error("Failed to delete medication event:", err);
      f7.dialog.alert(
        err instanceof Error ? err.message : "Failed to delete event",
        "Error",
      );
    }
    setConfirmDelete(false);
  }

  const displayName = medicationName ?? "Medication";

  return (
    <>
      <div className="medication-event-card">
        <Card>
          <CardHeader>
            <span className="om-card-title">
              {displayName} — {medicationEventStatusLabel(event.status)}
            </span>
            <div className="om-row">
              {onEdit && (
                <Link onClick={() => onEdit(event.name)}>Edit</Link>
              )}
              <Link color="red" onClick={() => setConfirmDelete(true)}>
                Delete
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {event.dose && <p className="om-card-timestamp">Dose: {event.dose}</p>}
            {event.timestamp && (
              <p className="om-card-timestamp">
                {formatDateTime(event.timestamp)}
              </p>
            )}
            {event.note && (
              <p className="om-card-notes om-muted om-truncate-2">
                {event.note}
              </p>
            )}
          </CardContent>
        </Card>
      </div>
      <ConfirmDialog
        open={confirmDelete}
        title="Delete Event"
        message="Are you sure you want to delete this medication event?"
        onConfirm={handleDelete}
        onCancel={() => setConfirmDelete(false)}
      />
    </>
  );
};
