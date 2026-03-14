import React, { useState, useEffect } from "react";
import { Page, Navbar, List, Button, BlockTitle } from "framework7-react";
import type { Router } from "framework7/types";
import { create } from "@bufbuild/protobuf";
import { MoodType, MoodIntensity } from "@gen/openmenses/v1/model_pb";
import { MoodObservationSchema } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toDateTime } from "../../lib/dates";
import { moodTypeOptions, moodIntensityOptions } from "../../lib/enums";
import { DateTimePicker } from "../../components/DateTimePicker";
import { EnumSelector } from "../../components/EnumSelector";
import { NotesField } from "../../components/NotesField";

interface MoodFormProps {
  f7router: Router.Router;
  name?: string;
}

const MoodForm: React.FC<MoodFormProps> = ({ f7router, name }) => {
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
        .catch(console.error);
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

      f7router.back();
    } catch (err) {
      console.error("Failed to save mood observation:", err);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Page>
      <Navbar title={isEdit ? "Edit Mood" : "Log Mood"} backLink="Back" />

      <List inset>
        <DateTimePicker value={timestamp} onChange={setTimestamp} />
      </List>

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
    </Page>
  );
};

export default MoodForm;
