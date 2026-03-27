import { Code, ConnectError } from "@connectrpc/connect";
import { Block, BlockTitle, Button, List, ListItem } from "framework7-react";
import React, { useState } from "react";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { personas, type Persona } from "./personas";

const DevToolsSection: React.FC = () => {
  const [confirmClear, setConfirmClear] = useState(false);
  const [selectedPersona, setSelectedPersona] = useState<Persona | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleClearData = async () => {
    try {
      setLoading(true);
      setError(null);
      try {
        await client.deleteUserProfile({ name: DEFAULT_PARENT });
      } catch (err) {
        // Ignore CodeNotFound — profile may not exist yet
        if (!(err instanceof ConnectError && err.code === Code.NotFound)) {
          throw err;
        }
      }
      // Reload the page to clear all UI state
      window.location.reload();
    } catch (err) {
      setLoading(false);
      if (err instanceof Error) {
        setError(`Failed to clear data: ${err.message}`);
      } else {
        setError("Failed to clear data");
      }
    }
  };

  const handleLoadPersona = async (persona: Persona) => {
    try {
      setLoading(true);
      setError(null);

      // Step 1: Delete existing profile (if it exists)
      try {
        await client.deleteUserProfile({ name: DEFAULT_PARENT });
      } catch (err) {
        // Ignore CodeNotFound errors - user may not have data yet
        if (!(err instanceof ConnectError && err.code === Code.NotFound)) {
          throw err;
        }
      }

      // Step 2: Fetch fixture data
      const resp = await fetch(persona.fixturePath);
      if (!resp.ok) {
        throw new Error(
          `Failed to fetch fixture: ${resp.status} ${resp.statusText}`,
        );
      }
      const data = await resp.arrayBuffer();

      // Step 3: Import the fixture data
      await client.createDataImport({
        parent: DEFAULT_PARENT,
        data: new Uint8Array(data),
      });

      // Reload the page to display all imported data
      window.location.reload();
    } catch (err) {
      setLoading(false);
      if (err instanceof Error) {
        setError(`Failed to load persona data: ${err.message}`);
      } else {
        setError("Failed to load persona data");
      }
    }
  };

  return (
    <>
      <BlockTitle>Dev Tools</BlockTitle>
      <Block inset>
        {error && <p className="form-error">{error}</p>}
        <Button
          onClick={() => setConfirmClear(true)}
          disabled={loading}
          color="red"
          outline
        >
          Clear All Data
        </Button>
      </Block>

      <BlockTitle>Simulate Persona</BlockTitle>
      <List inset>
        {personas.map((p) => (
          <ListItem
            key={p.id}
            title={p.name}
            after={p.subtitle}
            onClick={() => setSelectedPersona(p)}
            disabled={loading}
            link="#"
          />
        ))}
      </List>

      <ConfirmDialog
        open={confirmClear}
        title="Clear All Data"
        message="This will permanently delete your profile and all observations, cycles, predictions, and insights. This cannot be undone."
        onConfirm={() => {
          setConfirmClear(false);
          handleClearData();
        }}
        onCancel={() => setConfirmClear(false)}
      />

      <ConfirmDialog
        open={selectedPersona !== null}
        title="Load Sample Data"
        message={`This will replace all your data with sample data for ${selectedPersona?.name} (${selectedPersona?.subtitle}). Existing data will be deleted.`}
        onConfirm={() => {
          const p = selectedPersona;
          setSelectedPersona(null);
          if (p) handleLoadPersona(p);
        }}
        onCancel={() => setSelectedPersona(null)}
      />
    </>
  );
};

export default DevToolsSection;
