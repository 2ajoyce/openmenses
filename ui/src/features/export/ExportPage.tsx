import React, { useState } from "react";
import { Page, Navbar, Block, BlockTitle, Button, Icon } from "framework7-react";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { exportPayloadToCSV } from "./csvConverter";

const ExportPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleExportJSON = async () => {
    try {
      setLoading(true);
      setError(null);
      setSuccess(false);

      const response = await client.createDataExport({
        parent: DEFAULT_PARENT,
      });

      const jsonData = new Uint8Array(response.data);
      const blob = new Blob([jsonData], {
        type: "application/json;charset=utf-8",
      });
      downloadFile(blob, "openmenses-export.json");

      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to export JSON data"
      );
    } finally {
      setLoading(false);
    }
  };

  const handleExportCSV = async () => {
    try {
      setLoading(true);
      setError(null);
      setSuccess(false);

      const response = await client.createDataExport({
        parent: DEFAULT_PARENT,
      });

      // Parse the JSON response
      const jsonData = response.data;
      const text = new TextDecoder().decode(new Uint8Array(jsonData));
      const payload = JSON.parse(text);

      // Convert to CSV
      const csvOutput = exportPayloadToCSV(payload);

      // Combine all CSVs into one file with section headers
      const combinedCSV = [
        "# Bleeding Observations\n",
        csvOutput.bleeding,
        "\n\n# Symptom Observations\n",
        csvOutput.symptoms,
        "\n\n# Mood Observations\n",
        csvOutput.moods,
        "\n\n# Medications\n",
        csvOutput.medications,
        "\n\n# Medication Events\n",
        csvOutput.medicationEvents,
        "\n\n# Cycles\n",
        csvOutput.cycles,
      ].join("\n");

      const blob = new Blob([combinedCSV], {
        type: "text/csv;charset=utf-8",
      });
      downloadFile(blob, "openmenses-export.csv");

      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to export CSV data");
    } finally {
      setLoading(false);
    }
  };

  const downloadFile = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <Page>
      <Navbar title="Export Data" />

      <BlockTitle>Data Export</BlockTitle>
      <Block inset>
        <p className="om-muted">
          Download your cycle tracking data in JSON or CSV format. This includes all your observations, medications, and cycles.
        </p>

        <div className="export-buttons">
          <Button
            onClick={handleExportJSON}
            disabled={loading}
            fill
            large
            className="export-button"
          >
            {loading ? "Exporting..." : "Export as JSON"}
          </Button>

          <Button
            onClick={handleExportCSV}
            disabled={loading}
            fill
            large
            className="export-button"
          >
            {loading ? "Exporting..." : "Export as CSV"}
          </Button>
        </div>

        {error && (
          <div className="form-error">
            <Icon ios="f7:exclamationmark_circle_fill" md="material:error" />
            {error}
          </div>
        )}

        {success && (
          <div className="form-success">
            <Icon ios="f7:checkmark_circle_fill" md="material:check_circle" />
            Export successful
          </div>
        )}
      </Block>

      <BlockTitle>About Your Data</BlockTitle>
      <Block inset>
        <p className="om-muted">
          Your data is stored locally on this device and never sent to any server. Exporting your data allows you to:
        </p>
        <ul className="om-list">
          <li>Back up your tracking history</li>
          <li>Transfer data between devices</li>
          <li>Analyze your data in external tools</li>
        </ul>
      </Block>
    </Page>
  );
};

export default ExportPage;
