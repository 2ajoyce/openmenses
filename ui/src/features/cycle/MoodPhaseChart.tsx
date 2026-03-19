import React, { useEffect, useState } from "react";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from "recharts";
import type { MoodObservation } from "@gen/openmenses/v1/model_pb";
import { MoodType } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { fromLocalDate, fromDateTime } from "../../lib/dates";
import { moodTypeLabel } from "../../lib/enums";
import { ChartContainer } from "../../components/ChartContainer";

interface MoodPhaseData {
  phase: string;
  [moodTypeLabel: string]: string | number;
}

export const MoodPhaseChart: React.FC = () => {
  const [data, setData] = useState<MoodPhaseData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        // Fetch all mood observations
        const moodRes = await client.listMoodObservations({
          parent: DEFAULT_PARENT,
          pagination: { pageSize: 1000, pageToken: "" },
        });
        const moods = moodRes.observations || [];

        // Fetch all cycles to determine phase date ranges
        const cycleRes = await client.listCycles({
          parent: DEFAULT_PARENT,
          pagination: { pageSize: 100, pageToken: "" },
        });
        const cycles = cycleRes.cycles || [];

        // If no moods or no cycles with phase estimates, return empty
        if (moods.length === 0 || cycles.length === 0) {
          setData([]);
          return;
        }

        // Map moods to phases
        const phaseCounters: Record<string, Record<number, number>> = {
          "Menstruation": {},
          "Follicular": {},
          "Ovulation Window": {},
          "Luteal": {},
        };

        // Initialize counters for all phases and mood types
        Object.keys(phaseCounters).forEach((phase) => {
          Object.values(MoodType).forEach((moodType) => {
            if (typeof moodType === "number") {
              phaseCounters[phase]![moodType] = 0;
            }
          });
        });

        moods.forEach((mood: MoodObservation) => {
          if (!mood.timestamp) return;

          const moodDate = fromDateTime(mood.timestamp);
          let assignedPhase: string | null = null;

          // Find the cycle this mood falls into and determine its phase
          for (const cycle of cycles) {
            if (!cycle.startDate || !cycle.endDate) continue;

            const cycleStart = fromLocalDate(cycle.startDate);
            const cycleEnd = fromLocalDate(cycle.endDate);

            // Check if mood is within this cycle
            if (moodDate >= cycleStart && moodDate <= cycleEnd) {
              // Calculate which phase day this is in the cycle
              const dayInCycle = Math.floor((moodDate.getTime() - cycleStart.getTime()) / (1000 * 60 * 60 * 24));

              // Simple phase estimation (can be enhanced with biological model)
              // Menstruation: days 1-5
              // Follicular: days 6-12
              // Ovulation Window: days 13-17
              // Luteal: days 18+
              if (dayInCycle < 5) {
                assignedPhase = "Menstruation";
              } else if (dayInCycle < 12) {
                assignedPhase = "Follicular";
              } else if (dayInCycle < 17) {
                assignedPhase = "Ovulation Window";
              } else {
                assignedPhase = "Luteal";
              }
              break;
            }
          }

          // Count mood for its assigned phase
          if (assignedPhase && mood.mood !== undefined) {
            phaseCounters[assignedPhase]![mood.mood] =
              (phaseCounters[assignedPhase]![mood.mood] || 0) + 1;
          }
        });

        // Transform counters into chart data
        const chartData: MoodPhaseData[] = Object.entries(phaseCounters).map(
          ([phase, moodCounts]) => {
            const row: MoodPhaseData = { phase };
            Object.entries(moodCounts).forEach(([moodTypeNum, count]) => {
              const moodLabel = moodTypeLabel(parseInt(moodTypeNum) as MoodType);
              row[moodLabel] = count;
            });
            return row;
          },
        );

        setData(chartData);
      } catch (err) {
        console.error("Failed to fetch mood phase data:", err);
        setData([]);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading || data.length === 0) {
    return null;
  }

  // Text summary for screen readers
  const summaryText = `Grouped bar chart showing mood type distribution across cycle phases. `;
  const moodTypes = Object.keys(MoodType)
    .filter((k) => !isNaN(Number(k)))
    .map((k) => moodTypeLabel(parseInt(k) as MoodType));

  return (
    <ChartContainer data={data} title="Mood by Phase">
      <div role="img" aria-label="Grouped bar chart showing mood distribution across menstrual cycle phases">
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={data} margin={{ top: 20, right: 30, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--om-color-chart-grid)" />
            <XAxis dataKey="phase" />
            <YAxis />
            <Tooltip
              contentStyle={{
                backgroundColor: "var(--om-color-chart-tooltip-bg)",
                border: `1px solid var(--om-color-chart-tooltip-border)`,
              }}
            />
            <Legend />
            {moodTypes.map((moodLabel, index) => (
              <Bar
                key={moodLabel}
                dataKey={moodLabel}
                fill={`var(--om-color-mood-${index})`}
              />
            ))}
          </BarChart>
        </ResponsiveContainer>
        <div className="om-sr-only">
          {summaryText}
          Distribution across phases:
          {data.map((row) => `${row.phase}: ${Object.entries(row).filter(([k]) => k !== "phase").map(([k, v]) => `${k}: ${v}`).join(", ")}`).join("; ")}
        </div>
      </div>
    </ChartContainer>
  );
};
