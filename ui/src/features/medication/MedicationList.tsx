import React, { useState, useEffect, useCallback } from "react";
import {
  Page,
  Navbar,
  List,
  ListItem,
  Button,
  SwipeoutActions,
  SwipeoutButton,
  f7,
} from "framework7-react";
import type { Router } from "framework7/types";
import type { Medication } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { medicationCategoryLabel } from "../../lib/enums";
import { EmptyState } from "../../components/EmptyState";

interface MedicationListProps {
  f7router: Router.Router;
}

const MedicationList: React.FC<MedicationListProps> = ({ f7router }) => {
  const [medications, setMedications] = useState<Medication[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchMedications = useCallback(async () => {
    try {
      const res = await client.listMedications({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      });
      setMedications(res.medications);
    } catch (err) {
      console.error("Failed to fetch medications:", err);
      f7.dialog.alert(
        err instanceof Error ? err.message : "Failed to load medications",
        "Error",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMedications();
  }, [fetchMedications]);

  async function handleToggleActive(med: Medication) {
    try {
      await client.updateMedication({
        medication: { ...med, active: !med.active },
        updateMask: { paths: ["active"] },
      });
      await fetchMedications();
    } catch (err) {
      console.error("Failed to update medication:", err);
      f7.dialog.alert(
        err instanceof Error ? err.message : "Failed to update medication",
        "Error",
      );
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
                    .catch((err) => {
                      console.error("Failed to delete medication:", err);
                      f7.dialog.alert(
                        err instanceof Error
                          ? err.message
                          : "Failed to delete medication",
                        "Error",
                      );
                      fetchMedications();
                    });
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
    </Page>
  );
};

export default MedicationList;
