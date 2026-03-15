import { create } from "@bufbuild/protobuf";
import {
  SymptomObservationSchema,
  SymptomSeverity,
  SymptomType,
} from "@gen/openmenses/v1/model_pb";
import { BlockTitle, Button, f7, List, Navbar, Page } from "framework7-react";
import type { Router } from "framework7/types";
import React, { useEffect, useState } from "react";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EnumSelector } from "../../components/EnumSelector";
import { NotesField } from "../../components/NotesField";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toDateTime } from "../../lib/dates";
import { symptomSeverityOptions, symptomTypeOptions } from "../../lib/enums";

interface SymptomFormProps {
  f7router: Router.Router;
  f7route?: Router.Route;
  name?: string;
}

const SymptomForm: React.FC<SymptomFormProps> = ({ f7router, f7route, name: nameProp }) => {
  const name = nameProp ?? f7route?.query?.name;
  const [timestamp, setTimestamp] = useState(new Date());
  const [symptom, setSymptom] = useState<number>(SymptomType.CRAMPS);
  const [severity, setSeverity] = useState<number>(SymptomSeverity.MODERATE);
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const isEdit = Boolean(name);

  useEffect(() => {
    if (name) {
      client
        .getSymptomObservation({ name })
        .then((res) => {
          const obs = res.observation;
          if (obs) {
            if (obs.timestamp) setTimestamp(new Date(obs.timestamp.value));
            setSymptom(obs.symptom);
            setSeverity(obs.severity);
            setNote(obs.note);
          }
        })
        .catch((err) => {
          console.error("Failed to fetch symptom observation:", err);
          f7.dialog.alert(
            err instanceof Error ? err.message : "Failed to load observation",
            "Error",
          );
        });
    }
  }, [name]);

  async function handleSubmit() {
    setSubmitting(true);
    try {
      const observation = create(SymptomObservationSchema, {
        timestamp: toDateTime(timestamp),
        symptom: symptom as SymptomType,
        severity: severity as SymptomSeverity,
        note,
      });

      if (isEdit && name) {
        observation.name = name;
        observation.userId = DEFAULT_PARENT;
        await client.updateSymptomObservation({
          observation,
          updateMask: { paths: ["timestamp", "symptom", "severity", "note"] },
        });
      } else {
        await client.createSymptomObservation({
          parent: DEFAULT_PARENT,
          observation,
        });
      }

      f7router.back();
    } catch (err) {
      console.error("Failed to save symptom observation:", err);
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
      <Navbar title={isEdit ? "Edit Symptom" : "Log Symptom"} backLink="Back" />

      <div style={{ padding: "0 16px 8px" }}>
        <DateTimePicker value={timestamp} onChange={setTimestamp} />
      </div>

      <BlockTitle>Symptom</BlockTitle>
      <EnumSelector
        options={symptomTypeOptions}
        selected={symptom}
        onChange={setSymptom}
      />

      <BlockTitle>Severity</BlockTitle>
      <EnumSelector
        options={symptomSeverityOptions}
        selected={severity}
        onChange={setSeverity}
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

export default SymptomForm;
