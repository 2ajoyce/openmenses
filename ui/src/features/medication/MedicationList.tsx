import React, { useState, useEffect, useCallback } from "react";
import {
  Page,
  Navbar,
  List,
  ListItem,
  Button,
  SwipeoutActions,
  SwipeoutButton,
} from "framework7-react";
import type { Router } from "framework7/types";
import type { Medication } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { medicationCategoryLabel } from "../../lib/enums";
import { EmptyState } from "../../components/EmptyState";
import { ConfirmDialog } from "../../components/ConfirmDialog";

interface MedicationListProps {
  f7router: Router.Router;
}

const MedicationList: React.FC<MedicationListProps> = ({ f7router }) => {
  const [medications, setMedications] = useState<Medication[]>([]);
  const [loading, setLoading] = useState(true);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const fetchMedications = useCallback(async () => {
    try {
      const res = await client.listMedications({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      });
      setMedications(res.medications);
    } catch (err) {
      console.error("Failed to fetch medications:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMedications();
  }, [fetchMedications]);

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await client.deleteMedication({ name: deleteTarget });
      await fetchMedications();
    } catch (err) {
      console.error("Failed to delete medication:", err);
    }
    setDeleteTarget(null);
  }

  async function handleToggleActive(med: Medication) {
    try {
      await client.updateMedication({
        medication: { ...med, active: !med.active },
        updateMask: { paths: ["active"] },
      });
      await fetchMedications();
    } catch (err) {
      console.error("Failed to update medication:", err);
    }
  }

  if (!loading && medications.length === 0) {
    return (
      <Page>
        <Navbar title="Medications" />
        <EmptyState
          message="No medications added yet"
          actionLabel="Add Medication"
          onAction={() => f7router.navigate("/medications/new/")}
        />
      </Page>
    );
  }

  return (
    <Page>
      <Navbar title="Medications" />

      <List mediaList inset>
        {medications.map((med) => (
          <ListItem
            key={med.name}
            title={med.displayName}
            subtitle={medicationCategoryLabel(med.category)}
            after={med.active ? "Active" : "Inactive"}
            swipeout
            onClick={() =>
              f7router.navigate(`/medications/edit/`, {
                props: { medicationName: med.name },
              })
            }
          >
            <SwipeoutActions right>
              <SwipeoutButton
                color="orange"
                onClick={() => handleToggleActive(med)}
              >
                {med.active ? "Deactivate" : "Activate"}
              </SwipeoutButton>
              <SwipeoutButton
                delete
                confirmText="Delete this medication?"
                onClick={() => {
                  client
                    .deleteMedication({ name: med.name })
                    .then(fetchMedications)
                    .catch(console.error);
                }}
              >
                Delete
              </SwipeoutButton>
            </SwipeoutActions>
          </ListItem>
        ))}
      </List>

      <div style={{ padding: "0 16px" }}>
        <Button
          fill
          round
          large
          onClick={() => f7router.navigate("/medications/new/")}
        >
          Add Medication
        </Button>
      </div>

      <ConfirmDialog
        open={deleteTarget !== null}
        title="Delete Medication"
        message="Are you sure? This will also remove all related events."
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </Page>
  );
};

export default MedicationList;
