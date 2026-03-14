import React, { useState, useEffect } from "react";
import {
  Page,
  Navbar,
  List,
  Button,
  BlockTitle,
} from "framework7-react";
import type { Router } from "framework7/types";
import { create } from "@bufbuild/protobuf";
import { BleedingFlow } from "@gen/openmenses/v1/model_pb";
import { BleedingObservationSchema } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toDateTime } from "../../lib/dates";
import { bleedingFlowOptions } from "../../lib/enums";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EnumSelector } from "../../components/EnumSelector";
import { NotesField } from "../../components/NotesField";

interface BleedingFormProps {
  f7router: Router.Router;
  name?: string;
}

const BleedingForm: React.FC<BleedingFormProps> = ({ f7router, name }) => {
  const [timestamp, setTimestamp] = useState(new Date());
  const [flow, setFlow] = useState<number>(BleedingFlow.MEDIUM);
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const isEdit = Boolean(name);

  useEffect(() => {
    if (name) {
      client
        .getBleedingObservation({ name })
        .then((res) => {
          const obs = res.observation;
          if (obs) {
            if (obs.timestamp) setTimestamp(new Date(obs.timestamp.value));
            setFlow(obs.flow);
            setNote(obs.note);
          }
        })
        .catch(console.error);
    }
  }, [name]);

  async function handleSubmit() {
    setSubmitting(true);
    try {
      const observation = create(BleedingObservationSchema, {
        timestamp: toDateTime(timestamp),
        flow: flow as BleedingFlow,
        note,
      });

      if (isEdit && name) {
        observation.name = name;
        await client.updateBleedingObservation({
          observation,
          updateMask: { paths: ["timestamp", "flow", "note"] },
        });
      } else {
        await client.createBleedingObservation({
          parent: DEFAULT_PARENT,
          observation,
        });
      }

      f7router.back();
    } catch (err) {
      console.error("Failed to save bleeding observation:", err);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Page>
      <Navbar
        title={isEdit ? "Edit Bleeding" : "Log Bleeding"}
        backLink="Back"
      />

      <List inset>
        <DateTimePicker value={timestamp} onChange={setTimestamp} />
      </List>

      <BlockTitle>Flow</BlockTitle>
      <EnumSelector
        options={bleedingFlowOptions}
        selected={flow}
        onChange={setFlow}
      />

      <List inset>
        <NotesField value={note} onChange={setNote} />
      </List>

      <div style={{ padding: "0 16px" }}>
        <Button fill round large onClick={handleSubmit} disabled={submitting}>
          {submitting ? "Saving..." : isEdit ? "Update" : "Save"}
        </Button>
      </div>
    </Page>
  );
};

export default BleedingForm;
