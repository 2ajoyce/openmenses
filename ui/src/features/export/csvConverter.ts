import {
  bleedingFlowLabels,
  symptomTypeLabels,
  symptomSeverityLabels,
  moodTypeLabels,
  moodIntensityLabels,
  medicationCategoryLabels,
  medicationEventStatusLabels,
  cycleSourceLabels,
} from "../../lib/enums";

// Exported JSON structure from CreateDataExport RPC
export interface ExportPayload {
  version: string;
  user_id: string;
  profile?: {
    [key: string]: unknown;
  };
  bleeding_observations?: Record<string, unknown>[];
  symptom_observations?: Record<string, unknown>[];
  mood_observations?: Record<string, unknown>[];
  medications?: Record<string, unknown>[];
  medication_events?: Record<string, unknown>[];
  cycles?: Record<string, unknown>[];
}

/**
 * Converts a CreateDataExport JSON payload to CSV format.
 * Returns an object with multiple CSV strings, one per record type.
 */
export interface CSVOutput {
  bleeding: string;
  symptoms: string;
  moods: string;
  medications: string;
  medicationEvents: string;
  cycles: string;
}

/**
 * Convert exported JSON data to multiple CSV strings (one per record type).
 * Each CSV includes headers and rows with human-readable enum labels.
 * Timestamps are in ISO 8601 format. Empty optional fields are represented as empty strings.
 */
export function exportPayloadToCSV(payload: ExportPayload): CSVOutput {
  return {
    bleeding: generateBleedingCSV(payload.bleeding_observations || []),
    symptoms: generateSymptomCSV(payload.symptom_observations || []),
    moods: generateMoodCSV(payload.mood_observations || []),
    medications: generateMedicationCSV(payload.medications || []),
    medicationEvents: generateMedicationEventCSV(
      payload.medication_events || []
    ),
    cycles: generateCycleCSV(payload.cycles || []),
  };
}

function escapeCSV(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  const str = String(value);
  if (str.includes(",") || str.includes('"') || str.includes("\n")) {
    return `"${str.replace(/"/g, '""')}"`;
  }
  return str;
}

function generateBleedingCSV(records: Record<string, unknown>[]): string {
  const headers = ["Date", "Time", "Flow", "Notes"];
  const rows = records.map((rec) => {
    const timestamp = rec.timestamp as Record<string, unknown> | undefined;
    const date =
      timestamp?.date_string || timestamp?.date || "";
    const time =
      timestamp?.time_string || timestamp?.time || "";
    const flow = bleedingFlowLabels[rec.flow as number] || "";
    const notes = rec.note || "";
    return [date, time, flow, notes].map(escapeCSV);
  });
  return [headers, ...rows].map((row) => row.join(",")).join("\n");
}

function generateSymptomCSV(records: Record<string, unknown>[]): string {
  const headers = ["Date", "Time", "Symptom", "Severity", "Notes"];
  const rows = records.map((rec) => {
    const timestamp = rec.timestamp as Record<string, unknown> | undefined;
    const date =
      timestamp?.date_string || timestamp?.date || "";
    const time =
      timestamp?.time_string || timestamp?.time || "";
    const symptom = symptomTypeLabels[rec.symptom as number] || "";
    const severity =
      symptomSeverityLabels[rec.severity as number] || "";
    const notes = rec.note || "";
    return [date, time, symptom, severity, notes].map(escapeCSV);
  });
  return [headers, ...rows].map((row) => row.join(",")).join("\n");
}

function generateMoodCSV(records: Record<string, unknown>[]): string {
  const headers = ["Date", "Time", "Mood", "Intensity", "Notes"];
  const rows = records.map((rec) => {
    const timestamp = rec.timestamp as Record<string, unknown> | undefined;
    const date =
      timestamp?.date_string || timestamp?.date || "";
    const time =
      timestamp?.time_string || timestamp?.time || "";
    const mood = moodTypeLabels[rec.mood as number] || "";
    const intensity =
      moodIntensityLabels[rec.intensity as number] || "";
    const notes = rec.note || "";
    return [date, time, mood, intensity, notes].map(escapeCSV);
  });
  return [headers, ...rows].map((row) => row.join(",")).join("\n");
}

function generateMedicationCSV(records: Record<string, unknown>[]): string {
  const headers = ["Name", "Category", "Active", "Notes"];
  const rows = records.map((rec) => {
    const name = rec.display_name || "";
    const category =
      medicationCategoryLabels[rec.category as number] || "";
    const active = rec.active ? "Yes" : "No";
    const notes = rec.note || "";
    return [name, category, active, notes].map(escapeCSV);
  });
  return [headers, ...rows].map((row) => row.join(",")).join("\n");
}

function generateMedicationEventCSV(records: Record<string, unknown>[]): string {
  const headers = ["Date", "Time", "Medication ID", "Status", "Dose", "Notes"];
  const rows = records.map((rec) => {
    const timestamp = rec.timestamp as Record<string, unknown> | undefined;
    const date =
      timestamp?.date_string || timestamp?.date || "";
    const time =
      timestamp?.time_string || timestamp?.time || "";
    const medicationId = rec.medication_id || "";
    const status =
      medicationEventStatusLabels[rec.status as number] || "";
    const dose = rec.dose || "";
    const notes = rec.note || "";
    return [date, time, medicationId, status, dose, notes].map(escapeCSV);
  });
  return [headers, ...rows].map((row) => row.join(",")).join("\n");
}

function generateCycleCSV(records: Record<string, unknown>[]): string {
  const headers = ["Start Date", "End Date", "Length (days)", "Source"];
  const rows = records.map((rec) => {
    const startDate = rec.start_date || "";
    const endDate = rec.end_date || "";
    let length = "";
    if (startDate && endDate) {
      const start = new Date(String(startDate));
      const end = new Date(String(endDate));
      const days = Math.round(
        (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)
      ) + 1;
      length = String(days);
    }
    const source = cycleSourceLabels[rec.source as number] || "";
    return [startDate, endDate, length, source].map(escapeCSV);
  });
  return [headers, ...rows].map((row) => row.join(",")).join("\n");
}
