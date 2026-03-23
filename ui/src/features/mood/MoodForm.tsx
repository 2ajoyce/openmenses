import { create } from "@bufbuild/protobuf";
import {
  MoodIntensity,
  MoodObservationSchema,
  MoodType,
} from "@gen/openmenses/v1/model_pb";
import { BlockTitle, Button, f7, List, Navbar, Page } from "framework7-react";
import type { Router } from "framework7/types";
import React, { useEffect, useState } from "react";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EnumSelector } from "../../components/EnumSelector";
import { NotesField } from "../../components/NotesField";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toDateTime } from "../../lib/dates";
import { moodIntensityOptions, moodTypeOptions } from "../../lib/enums";

interface MoodFormProps {
  f7router: Router.Router;
  f7route?: Router.Route;
  name?: string;
}

const MoodForm: React.FC<MoodFormProps> = ({
  f7router,
  f7route,
  name: nameProp,
}) => {
  const name = nameProp ?? f7route?.query?.name;
  const [timestamp, setTimestamp] = useState(new Date());
  const [mood, setMood] = useState<number>(MoodType.CALM);
  const [intensity, setIntensity] = useState<number>(MoodIntensity.MEDIUM);
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const isEdit = Boolean(name);

  useEffect(() => {
    if (name) {
      client
        .getMoodObservation({ name })
        .then((res) => {
          const obs = res.observation;
          if (obs) {
            if (obs.timestamp) setTimestamp(new Date(obs.timestamp.value));
            setMood(obs.mood);
            setIntensity(obs.intensity);
            setNote(obs.note);
          }
        })
        .catch((err) => {
          console.error("Failed to fetch mood observation:", err);
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
      const observation = create(MoodObservationSchema, {
        timestamp: toDateTime(timestamp),
        mood: mood as MoodType,
        intensity: intensity as MoodIntensity,
        note,
      });

      if (isEdit && name) {
        observation.name = name;
        observation.userId = DEFAULT_PARENT;
        await client.updateMoodObservation({
          observation,
          updateMask: { paths: ["timestamp", "mood", "intensity", "note"] },
        });
      } else {
        await client.createMoodObservation({
          parent: DEFAULT_PARENT,
          observation,
        });
      }

      if (f7router.view?.main) {
        f7router.navigate("/", { clearPreviousHistory: true });
      } else {
        f7router.back();
      }
    } catch (err) {
      console.error("Failed to save mood observation:", err);
      f7.dialog.alert(
        err instanceof Error ? err.message : "An unexpected error occurred",
        "Error",
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Page pageContent={false}>
      <div className="page-content">
        <Navbar title={isEdit ? "Edit Mood" : "Log Mood"} backLink="Back" />

        <div style={{ padding: "0 16px 8px" }}>
          <DateTimePicker value={timestamp} onChange={setTimestamp} />
        </div>

        <BlockTitle>Mood</BlockTitle>
        <EnumSelector
          options={moodTypeOptions}
          selected={mood}
          onChange={setMood}
        />

        <BlockTitle>Intensity</BlockTitle>
        <EnumSelector
          options={moodIntensityOptions}
          selected={intensity}
          onChange={setIntensity}
        />

        <List inset>
          <NotesField value={note} onChange={setNote} />
        </List>

        <div style={{ padding: "0 16px" }}>
          <Button fill round large onClick={handleSubmit} disabled={submitting}>
            {submitting ? "Saving..." : isEdit ? "Update" : "Save"}
          </Button>
        </div>
      </div>
    </Page>
  );
};

export default MoodForm;
