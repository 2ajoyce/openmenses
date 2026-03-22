import type {
  BleedingObservation,
  MedicationEvent,
  MoodObservation,
  PhaseEstimate,
  SymptomObservation,
} from "@gen/openmenses/v1/model_pb";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import React, { useCallback, useEffect, useState } from "react";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { fromDateTime, toLocalDate } from "../../lib/dates";
import {
  bleedingFlowLabel,
  moodIntensityLabel,
  symptomTypeLabel,
} from "../../lib/enums";

interface ObservationsByDate {
  [date: string]: {
    bleeding?: BleedingObservation[];
    symptoms?: SymptomObservation[];
    moods?: MoodObservation[];
    medications?: MedicationEvent[];
    phase?: PhaseEstimate;
  };
}

export const CalendarHeatmap: React.FC = () => {
  const [currentMonth, setCurrentMonth] = useState(new Date());
  const [observations, setObservations] = useState<ObservationsByDate>({});
  const [loading, setLoading] = useState(false);

  // Fetch timeline data for the current month
  const fetchMonthObservations = useCallback(async (date: Date) => {
    setLoading(true);
    try {
      const startOfMonth = new Date(date.getFullYear(), date.getMonth(), 1);
      const endOfMonth = new Date(date.getFullYear(), date.getMonth() + 1, 0);

      const allRecords: TimelineRecord[] = [];
      let pageToken = "";
      do {
        const res = await client.listTimeline({
          parent: DEFAULT_PARENT,
          range: {
            start: toLocalDate(startOfMonth),
            end: toLocalDate(endOfMonth),
          },
          pagination: { pageSize: 500, pageToken },
        });
        allRecords.push(...res.records);
        pageToken = res.pagination?.nextPageToken ?? "";
      } while (pageToken);

      // Group observations by date
      const byDate: ObservationsByDate = {};
      const getEntry = (dateStr: string) => {
        if (!byDate[dateStr]) byDate[dateStr] = {};
        return byDate[dateStr]!;
      };
      allRecords.forEach((record: TimelineRecord) => {
        const { record: data } = record;

        switch (data.case) {
          case "bleedingObservation": {
            if (data.value.timestamp) {
              const entry = getEntry(
                toLocalDate(fromDateTime(data.value.timestamp)).value,
              );
              if (!entry.bleeding) entry.bleeding = [];
              entry.bleeding.push(data.value);
            }
            break;
          }
          case "symptomObservation": {
            if (data.value.timestamp) {
              const entry = getEntry(
                toLocalDate(fromDateTime(data.value.timestamp)).value,
              );
              if (!entry.symptoms) entry.symptoms = [];
              entry.symptoms.push(data.value);
            }
            break;
          }
          case "moodObservation": {
            if (data.value.timestamp) {
              const entry = getEntry(
                toLocalDate(fromDateTime(data.value.timestamp)).value,
              );
              if (!entry.moods) entry.moods = [];
              entry.moods.push(data.value);
            }
            break;
          }
          case "medicationEvent": {
            if (data.value.timestamp) {
              const entry = getEntry(
                toLocalDate(fromDateTime(data.value.timestamp)).value,
              );
              if (!entry.medications) entry.medications = [];
              entry.medications.push(data.value);
            }
            break;
          }
          case "phaseEstimate": {
            const dateStr = data.value.date?.value ?? "";
            if (dateStr) {
              getEntry(dateStr).phase = data.value;
            }
            break;
          }
        }
      });

      setObservations(byDate);
    } catch (err) {
      console.error("Failed to fetch month observations:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMonthObservations(currentMonth);
  }, [currentMonth, fetchMonthObservations]);

  const handlePrevMonth = () => {
    setCurrentMonth(
      (prev) => new Date(prev.getFullYear(), prev.getMonth() - 1, 1),
    );
  };

  const handleNextMonth = () => {
    setCurrentMonth(
      (prev) => new Date(prev.getFullYear(), prev.getMonth() + 1, 1),
    );
  };

  // Generate calendar grid
  const generateCalendarDays = (date: Date) => {
    const year = date.getFullYear();
    const month = date.getMonth();
    const firstDay = new Date(year, month, 1).getDay(); // 0 = Sunday
    const daysInMonth = new Date(year, month + 1, 0).getDate();

    const days: (number | null)[] = [];

    // Fill empty cells before month starts
    for (let i = 0; i < firstDay; i++) {
      days.push(null);
    }

    // Fill days of month
    for (let i = 1; i <= daysInMonth; i++) {
      days.push(i);
    }

    return days;
  };

  const getObservationSummary = (dateStr: string) => {
    const obs = observations[dateStr];
    if (!obs) return "";

    const parts: string[] = [];
    if (obs.bleeding && obs.bleeding.length > 0) {
      const flows = obs.bleeding
        .map((b) => bleedingFlowLabel(b.flow))
        .join(", ");
      parts.push(`bleeding (${flows})`);
    }
    if (obs.symptoms && obs.symptoms.length > 0) {
      const types = obs.symptoms
        .map((s) => symptomTypeLabel(s.symptom))
        .join(", ");
      parts.push(`symptoms: ${types}`);
    }
    if (obs.moods && obs.moods.length > 0) {
      const intensities = obs.moods
        .map((m) => moodIntensityLabel(m.intensity))
        .join(", ");
      parts.push(`mood: ${intensities}`);
    }
    if (obs.medications && obs.medications.length > 0) {
      parts.push(`medication (${obs.medications.length})`);
    }

    return parts.join("; ");
  };

  const getObservationIndicators = (dateStr: string) => {
    const obs = observations[dateStr];
    if (!obs) return { hasObservations: false, types: new Set<string>() };

    const types = new Set<string>();
    if (obs.bleeding && obs.bleeding.length > 0) types.add("bleeding");
    if (obs.symptoms && obs.symptoms.length > 0) types.add("symptom");
    if (obs.moods && obs.moods.length > 0) types.add("mood");
    if (obs.medications && obs.medications.length > 0) types.add("medication");

    return { hasObservations: types.size > 0, types };
  };

  const getPhaseColor = (dateStr: string) => {
    const obs = observations[dateStr];
    if (!obs?.phase) return "";

    const phaseEstimate = obs.phase as Record<string, unknown>;
    const phase = phaseEstimate.phase as number | undefined;
    switch (phase) {
      case 1:
        return "menstruation";
      case 2:
        return "follicular";
      case 3:
        return "ovulation";
      case 4:
        return "luteal";
      default:
        return "";
    }
  };

  const days = generateCalendarDays(currentMonth);
  const dayLabels = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const monthYear = currentMonth.toLocaleDateString("en-US", {
    month: "long",
    year: "numeric",
  });

  return (
    <div className="om-heatmap-container">
      <div className="om-heatmap-nav">
        <button
          className="om-heatmap-nav-button"
          onClick={handlePrevMonth}
          aria-label="Previous month"
        >
          ← Prev
        </button>
        <h3 className="om-heatmap-month">{monthYear}</h3>
        <button
          className="om-heatmap-nav-button"
          onClick={handleNextMonth}
          aria-label="Next month"
        >
          Next →
        </button>
      </div>

      {loading && <p className="om-loading-text">Loading...</p>}

      <div className="om-heatmap-grid" role="presentation">
        {/* Day labels */}
        {dayLabels.map((label) => (
          <div key={`label-${label}`} className="om-heatmap-label">
            {label}
          </div>
        ))}

        {/* Calendar cells */}
        {days.map((day, index) => {
          if (day === null) {
            return (
              <div
                key={`empty-${index}`}
                className="om-heatmap-cell om-heatmap-empty"
              />
            );
          }

          const dateStr = `${currentMonth.getFullYear()}-${String(currentMonth.getMonth() + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
          const { hasObservations, types } = getObservationIndicators(dateStr);
          const phaseColor = getPhaseColor(dateStr);
          const summary = getObservationSummary(dateStr);
          const cellDate = new Date(
            currentMonth.getFullYear(),
            currentMonth.getMonth(),
            day,
          );
          const ariaLabel = `${cellDate.toLocaleDateString("en-US", { month: "long", day: "numeric" })}: ${summary || "no observations"}`;

          return (
            <div
              key={`day-${day}`}
              className={`om-heatmap-cell ${hasObservations ? "om-heatmap-cell-active" : ""} ${phaseColor ? `om-heatmap-phase-${phaseColor}` : ""}`}
              aria-label={ariaLabel}
              title={summary}
            >
              <div className="om-heatmap-day-number">{day}</div>
              {hasObservations && (
                <div className="om-heatmap-indicators">
                  {types.has("bleeding") && (
                    <div className="om-heatmap-indicator om-heatmap-indicator-bleeding" />
                  )}
                  {types.has("symptom") && (
                    <div className="om-heatmap-indicator om-heatmap-indicator-symptom" />
                  )}
                  {types.has("mood") && (
                    <div className="om-heatmap-indicator om-heatmap-indicator-mood" />
                  )}
                  {types.has("medication") && (
                    <div className="om-heatmap-indicator om-heatmap-indicator-medication" />
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="om-heatmap-legend" aria-label="Calendar legend">
        <div className="om-heatmap-legend-section">
          <span className="om-heatmap-legend-title">Observations</span>
          <div className="om-heatmap-legend-items">
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-indicator om-heatmap-indicator-bleeding" />
              <span>Bleeding</span>
            </div>
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-indicator om-heatmap-indicator-symptom" />
              <span>Symptom</span>
            </div>
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-indicator om-heatmap-indicator-mood" />
              <span>Mood</span>
            </div>
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-indicator om-heatmap-indicator-medication" />
              <span>Medication</span>
            </div>
          </div>
        </div>
        <div className="om-heatmap-legend-section">
          <span className="om-heatmap-legend-title">Phases</span>
          <div className="om-heatmap-legend-items">
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-legend-swatch om-heatmap-phase-menstruation" />
              <span>Menstruation</span>
            </div>
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-legend-swatch om-heatmap-phase-follicular" />
              <span>Follicular</span>
            </div>
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-legend-swatch om-heatmap-phase-ovulation" />
              <span>Ovulation</span>
            </div>
            <div className="om-heatmap-legend-item">
              <div className="om-heatmap-legend-swatch om-heatmap-phase-luteal" />
              <span>Luteal</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
