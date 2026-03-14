import React, { useState, useEffect } from "react";
import { Page, Navbar, List, ListInput, Button, BlockTitle } from "framework7-react";
import type { Router } from "framework7/types";
import { create } from "@bufbuild/protobuf";
import { MedicationCategory, MedicationSchema } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { medicationCategoryOptions } from "../../lib/enums";
import { EnumSelector } from "../../components/EnumSelector";
import { NotesField } from "../../components/NotesField";

interface MedicationFormProps {
  f7router: Router.Router;
  medicationName?: string;
}

const MedicationForm: React.FC<MedicationFormProps> = ({
  f7router,
  medicationName,
}) => {
  const [displayName, setDisplayName] = useState("");
  const [category, setCategory] = useState<number>(
    MedicationCategory.PAIN_RELIEF,
  );
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const isEdit = Boolean(medicationName);

  useEffect(() => {
    if (medicationName) {
      client
        .getMedication({ name: medicationName })
        .then((res) => {
          const med = res.medication;
          if (med) {
            setDisplayName(med.displayName);
            setCategory(med.category);
            setNote(med.note);
          }
        })
        .catch(console.error);
    }
  }, [medicationName]);

  async function handleSubmit() {
    if (!displayName.trim()) return;
    setSubmitting(true);
    try {
      if (isEdit && medicationName) {
        const medication = create(MedicationSchema, {
          name: medicationName,
          displayName,
          category: category as MedicationCategory,
          note,
          active: true,
        });
        await client.updateMedication({
          medication,
          updateMask: { paths: ["display_name", "category", "note"] },
        });
      } else {
        const medication = create(MedicationSchema, {
          displayName,
          category: category as MedicationCategory,
          note,
          active: true,
        });
        await client.createMedication({
          parent: DEFAULT_PARENT,
          medication,
        });
      }

      f7router.back();
    } catch (err) {
      console.error("Failed to save medication:", err);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Page>
      <Navbar
        title={isEdit ? "Edit Medication" : "Add Medication"}
        backLink="Back"
      />

      <List inset>
        <ListInput
          label="Name"
          type="text"
          placeholder="Medication name"
          value={displayName}
          onInput={(e: React.ChangeEvent<HTMLInputElement>) =>
            setDisplayName(e.target.value)
          }
          required
        />
      </List>

      <BlockTitle>Category</BlockTitle>
      <EnumSelector
        options={medicationCategoryOptions}
        selected={category}
        onChange={setCategory}
      />

      <List inset>
        <NotesField value={note} onChange={setNote} />
      </List>

      <div style={{ padding: "0 16px" }}>
        <Button
          fill
          round
          large
          onClick={handleSubmit}
          disabled={submitting || !displayName.trim()}
        >
          {submitting ? "Saving..." : isEdit ? "Update" : "Add"}
        </Button>
      </div>
    </Page>
  );
};

export default MedicationForm;
