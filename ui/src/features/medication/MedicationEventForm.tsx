import { create } from "@bufbuild/protobuf";
import type { Medication } from "@gen/openmenses/v1/model_pb";
import {
  MedicationEventSchema,
  MedicationEventStatus,
} from "@gen/openmenses/v1/model_pb";
import {
  BlockTitle,
  Button,
  f7,
  List,
  ListInput,
  Navbar,
  Page,
} from "framework7-react";
import type { Router } from "framework7/types";
import React, { useEffect, useState } from "react";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EnumSelector } from "../../components/EnumSelector";
import { NotesField } from "../../components/NotesField";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toDateTime } from "../../lib/dates";
import { medicationEventStatusOptions } from "../../lib/enums";

interface MedicationEventFormProps {
  f7router: Router.Router;
  name?: string;
}

const MedicationEventForm: React.FC<MedicationEventFormProps> = ({
  f7router,
  name,
}) => {
  const [medications, setMedications] = useState<Medication[]>([]);
  const [medicationId, setMedicationId] = useState("");
  const [timestamp, setTimestamp] = useState(new Date());
  const [status, setStatus] = useState<number>(MedicationEventStatus.TAKEN);
  const [dose, setDose] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [dataLoaded, setDataLoaded] = useState(!name);
  const isEdit = Boolean(name);

  useEffect(() => {
    client
      .listMedications({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      })
      .then((res) => {
        const active = res.medications.filter((m) => m.active);
        setMedications(active);
        if (active.length > 0 && !name && !medicationId) {
          setMedicationId(active[0]!.name);
        }
      })
      .catch((err) => {
        console.error("Failed to fetch medications:", err);
        f7.dialog.alert(
          err instanceof Error ? err.message : "Failed to load medications",
          "Error",
        );
      });
  }, []);

  useEffect(() => {
    if (name) {
      client
        .getMedicationEvent({ name })
        .then((res) => {
          const evt = res.event;
          if (evt) {
            setMedicationId(evt.medicationId);
            if (evt.timestamp) setTimestamp(new Date(evt.timestamp.value));
            setStatus(evt.status);
            setDose(evt.dose);
            setNote(evt.note);
            setDataLoaded(true);
          }
        })
        .catch((err) => {
          console.error("Failed to fetch medication event:", err);
          f7.dialog.alert(
            err instanceof Error ? err.message : "Failed to load event",
            "Error",
          );
        });
    }
  }, [name]);

  async function handleSubmit() {
    if (!medicationId) return;
    setSubmitting(true);
    try {
      const event = create(MedicationEventSchema, {
        medicationId,
        timestamp: toDateTime(timestamp),
        status: status as MedicationEventStatus,
        dose,
        note,
      });

      if (isEdit && name) {
        event.name = name;
        event.userId = DEFAULT_PARENT;
        await client.updateMedicationEvent({
          event,
          updateMask: {
            paths: ["timestamp", "status", "dose", "note"],
          },
        });
      } else {
        await client.createMedicationEvent({
          parent: DEFAULT_PARENT,
          event,
        });
      }

      f7router.back();
    } catch (err) {
      console.error("Failed to save medication event:", err);
      f7.dialog.alert(
        err instanceof Error ? err.message : "An unexpected error occurred",
        "Error",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Page>
      <Navbar
        title={isEdit ? "Edit Medication Event" : "Log Medication"}
        backLink="Back"
      />

      <List inset>
        <ListInput
          label="Medication"
          type="select"
          value={medicationId}
          disabled={isEdit}
          onInput={(e: React.ChangeEvent<HTMLSelectElement>) =>
            setMedicationId(e.target.value)
          }
        >
          {medications.map((med) => (
            <option key={med.name} value={med.name}>
              {med.displayName}
            </option>
          ))}
          {medications.length === 0 && (
            <option value="">No active medications</option>
          )}
        </ListInput>
      </List>

      <div style={{ padding: "0 16px 8px" }}>
        <DateTimePicker value={timestamp} onChange={setTimestamp} />
      </div>

      <BlockTitle>Status</BlockTitle>
      <EnumSelector
        options={medicationEventStatusOptions}
        selected={status}
        onChange={setStatus}
      />

      <List inset key={dataLoaded ? "ready" : "loading"}>
        <ListInput
          label="Dose"
          type="text"
          placeholder="e.g., 200mg"
          value={dose}
          onInput={(e: React.ChangeEvent<HTMLInputElement>) =>
            setDose(e.target.value)
          }
        />
        <NotesField value={note} onChange={setNote} />
      </List>

      <div style={{ padding: "0 16px" }}>
        <Button
          fill
          round
          large
          onClick={handleSubmit}
          disabled={submitting || !medicationId}
        >
          {submitting ? "Saving..." : isEdit ? "Update" : "Save"}
        </Button>
      </div>
    </Page>
  );
};

export default MedicationEventForm;
