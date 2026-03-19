import { describe, it, expect } from "vitest";
import { exportPayloadToCSV } from "./csvConverter";
import type { ExportPayload } from "./csvConverter";

describe("csvConverter", () => {
  it("should convert bleeding observations to CSV", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      bleeding_observations: [
        {
          timestamp: { date: "2026-03-01", time: "09:00:00" },
          flow: 3, // MEDIUM
          note: "Normal",
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.bleeding).toContain("Date,Time,Flow,Notes");
    expect(csv.bleeding).toContain("2026-03-01");
    expect(csv.bleeding).toContain("Medium");
    expect(csv.bleeding).toContain("Normal");
  });

  it("should convert symptom observations to CSV", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      symptom_observations: [
        {
          timestamp: { date: "2026-03-02", time: "10:30:00" },
          symptom: 1, // CRAMPS
          severity: 2, // MILD
          note: "Mild pain",
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.symptoms).toContain("Date,Time,Symptom,Severity,Notes");
    expect(csv.symptoms).toContain("2026-03-02");
    expect(csv.symptoms).toContain("Cramps");
    expect(csv.symptoms).toContain("Mild");
    expect(csv.symptoms).toContain("Mild pain");
  });

  it("should convert mood observations to CSV", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      mood_observations: [
        {
          timestamp: { date: "2026-03-03", time: "14:00:00" },
          mood: 4, // ANXIOUS
          intensity: 2, // MEDIUM
          note: "Feeling stressed",
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.moods).toContain("Date,Time,Mood,Intensity,Notes");
    expect(csv.moods).toContain("2026-03-03");
    expect(csv.moods).toContain("Anxious");
    expect(csv.moods).toContain("Medium");
    expect(csv.moods).toContain("Feeling stressed");
  });

  it("should convert medications to CSV", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      medications: [
        {
          display_name: "Ibuprofen",
          category: 2, // PAIN_RELIEF
          active: true,
          note: "As needed",
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.medications).toContain("Name,Category,Active,Notes");
    expect(csv.medications).toContain("Ibuprofen");
    expect(csv.medications).toContain("Pain Relief");
    expect(csv.medications).toContain("Yes");
    expect(csv.medications).toContain("As needed");
  });

  it("should convert medication events to CSV", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      medication_events: [
        {
          timestamp: { date: "2026-03-04", time: "09:00:00" },
          medication_id: "med-123",
          status: 1, // TAKEN
          dose: "200mg",
          note: "Taken with food",
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.medicationEvents).toContain(
      "Date,Time,Medication ID,Status,Dose,Notes"
    );
    expect(csv.medicationEvents).toContain("2026-03-04");
    expect(csv.medicationEvents).toContain("med-123");
    expect(csv.medicationEvents).toContain("Taken");
    expect(csv.medicationEvents).toContain("200mg");
    expect(csv.medicationEvents).toContain("Taken with food");
  });

  it("should convert cycles to CSV", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      cycles: [
        {
          start_date: "2026-03-01",
          end_date: "2026-03-28",
          source: 2, // USER_CONFIRMED
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.cycles).toContain("Start Date,End Date,Length (days),Source");
    expect(csv.cycles).toContain("2026-03-01");
    expect(csv.cycles).toContain("2026-03-28");
    expect(csv.cycles).toContain("User confirmed");
  });

  it("should escape CSV special characters", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      symptom_observations: [
        {
          timestamp: { date: "2026-03-05", time: "10:00:00" },
          symptom: 1,
          severity: 1,
          note: 'Note with "quotes" and, commas',
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.symptoms).toContain('"Note with ""quotes"" and, commas"');
  });

  it("should handle empty optional fields", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      bleeding_observations: [
        {
          timestamp: { date: "2026-03-06", time: "09:00:00" },
          flow: 1,
          // note is missing/undefined
        },
      ],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.bleeding).toContain("2026-03-06,09:00:00,Spotting,");
  });

  it("should handle empty arrays", () => {
    const payload: ExportPayload = {
      version: "1",
      user_id: "users/default",
      bleeding_observations: [],
    };

    const csv = exportPayloadToCSV(payload);
    expect(csv.bleeding).toContain("Date,Time,Flow,Notes");
    expect(csv.bleeding.split("\n").length).toBe(1); // Only header
  });
});
