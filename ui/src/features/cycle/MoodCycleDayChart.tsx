import type { Cycle, MoodObservation } from "@gen/openmenses/v1/model_pb";
import { MoodType } from "@gen/openmenses/v1/model_pb";
import React, { useEffect, useState } from "react";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ReferenceArea,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { ChartContainer } from "../../components/ChartContainer";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { fromDateTime, fromLocalDate } from "../../lib/dates";
import { moodTypeLabel } from "../../lib/enums";

interface MoodCycleDayData {
  cycleDay: number;
  [moodTypeLabel: string]: number;
}

export const MoodCycleDayChart: React.FC = () => {
  const [data, setData] = useState<MoodCycleDayData[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        // Fetch all mood observations
        const moods: MoodObservation[] = [];
        let moodPageToken = "";
        do {
          const moodRes = await client.listMoodObservations({
            parent: DEFAULT_PARENT,
            pagination: { pageSize: 500, pageToken: moodPageToken },
          });
          moods.push(...(moodRes.observations || []));
          moodPageToken = moodRes.pagination?.nextPageToken ?? "";
        } while (moodPageToken);

        // Fetch all cycles
        const cycles: Cycle[] = [];
        let cyclePageToken = "";
        do {
          const cycleRes = await client.listCycles({
            parent: DEFAULT_PARENT,
            pagination: { pageSize: 500, pageToken: cyclePageToken },
          });
          cycles.push(...(cycleRes.cycles || []));
          cyclePageToken = cycleRes.pagination?.nextPageToken ?? "";
        } while (cyclePageToken);

        // Filter to completed cycles only
        const completedCycles = cycles.filter((c) => c.endDate);

        // If insufficient data, return empty
        if (moods.length === 0 || completedCycles.length < 3) {
          setData([]);
          return;
        }

        // Map moods to cycle days and aggregate by mood type
        const cycleDayCounters: Record<
          number,
          Record<number, { sum: number; count: number }>
        > = {};

        moods.forEach((mood: MoodObservation) => {
          if (
            !mood.timestamp ||
            mood.mood === undefined ||
            mood.intensity === undefined
          )
            return;

          const moodDate = fromDateTime(mood.timestamp);
          const moodType = mood.mood; // MoodType enum
          const moodIntensity = mood.intensity; // MoodIntensity (1-3)

          // Find the cycle this mood falls into
          for (const cycle of completedCycles) {
            if (!cycle.startDate || !cycle.endDate) continue;

            const cycleStart = fromLocalDate(cycle.startDate);
            const cycleEnd = fromLocalDate(cycle.endDate);

            // Check if mood is within this cycle
            if (moodDate >= cycleStart && moodDate <= cycleEnd) {
              // Calculate cycle day (1-indexed)
              const dayInCycle =
                Math.floor(
                  (moodDate.getTime() - cycleStart.getTime()) /
                    (1000 * 60 * 60 * 24),
                ) + 1;

              if (dayInCycle > 0 && dayInCycle <= 35) {
                // Initialize counters for this cycle day if not exists
                if (!cycleDayCounters[dayInCycle]) {
                  cycleDayCounters[dayInCycle] = {};
                }

                // Initialize mood type counter if not exists
                if (!cycleDayCounters[dayInCycle]![moodType]) {
                  cycleDayCounters[dayInCycle]![moodType] = {
                    sum: 0,
                    count: 0,
                  };
                }

                // Accumulate intensity values for this mood type
                cycleDayCounters[dayInCycle]![moodType]!.sum += moodIntensity;
                cycleDayCounters[dayInCycle]![moodType]!.count += 1;
              }
              break;
            }
          }
        });

        // Transform counters into chart data
        const chartData: MoodCycleDayData[] = Object.entries(cycleDayCounters)
          .sort((a, b) => parseInt(a[0]) - parseInt(b[0]))
          .map(([dayStr, moodCounts]) => {
            const row: MoodCycleDayData = { cycleDay: parseInt(dayStr) };

            // For each mood type, compute average intensity across all observations for this cycle day
            Object.entries(moodCounts).forEach(
              ([moodTypeNum, intensityData]) => {
                const moodLabel = moodTypeLabel(
                  parseInt(moodTypeNum) as MoodType,
                );
                const average = intensityData.sum / intensityData.count;
                row[moodLabel] = parseFloat(average.toFixed(2));
              },
            );

            return row;
          });

        setData(chartData);
      } catch (err) {
        console.error("Failed to fetch mood cycle day data:", err);
        setData([]);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) return null;
  if (data.length === 0) return null;

  // Derive active mood types from data (only those with at least one observation)
  const moodTypes = Array.from(
    new Set(
      data.flatMap((row) => Object.keys(row).filter((k) => k !== "cycleDay")),
    ),
  ).sort();

  // Text summary for screen readers
  const summaryText = `Line chart showing average mood intensity by cycle day. X-axis represents cycle day (1-35), Y-axis represents average mood intensity (1-3).`;

  return (
    <ChartContainer
      data={data}
      title="Mood Intensity by Cycle Day"
      emptyMessage="Track moods across at least 3 cycles to see mood intensity patterns by cycle day."
    >
      <div
        role="img"
        aria-label="Line chart showing average mood intensity across cycle days"
      >
        <ResponsiveContainer width="100%" height={300}>
          <LineChart
            data={data}
            margin={{ top: 5, right: 10, left: 5, bottom: 20 }}
          >
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="var(--om-color-chart-grid)"
            />
            <XAxis
              dataKey="cycleDay"
              type="number"
              label={{
                value: "Cycle Day",
                position: "insideBottom",
                offset: -5,
              }}
            />
            <YAxis
              domain={[1, 3]}
              label={{
                value: "Mood Intensity",
                angle: -90,
                position: "center",
                dx: -10,
              }}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "var(--om-color-chart-tooltip-bg)",
                border: `1px solid var(--om-color-chart-tooltip-border)`,
              }}
              labelFormatter={(value: number) => `Day ${value}`}
              formatter={(value: number) => value.toFixed(2)}
            />
            <Legend />
            <ReferenceArea
              x1={1}
              x2={5}
              fill="var(--om-color-phase-menstruation)"
              fillOpacity={0.08}
            />
            <ReferenceArea
              x1={6}
              x2={12}
              fill="var(--om-color-phase-follicular)"
              fillOpacity={0.08}
            />
            <ReferenceArea
              x1={13}
              x2={17}
              fill="var(--om-color-phase-ovulation)"
              fillOpacity={0.08}
            />
            <ReferenceArea
              x1={18}
              x2={35}
              fill="var(--om-color-phase-luteal)"
              fillOpacity={0.08}
            />
            {moodTypes.map((moodLabel, index) => (
              <Line
                key={moodLabel}
                type="monotone"
                dataKey={moodLabel}
                stroke={`var(--om-color-mood-${index})`}
                dot={false}
                isAnimationActive={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
        <div className="om-sr-only">
          {summaryText}
          Average mood intensities by cycle day:
          {data
            .map(
              (row) =>
                `Day ${row.cycleDay}: ${Object.entries(row)
                  .filter(([k]) => k !== "cycleDay")
                  .map(([k, v]) => `${k}: ${v}`)
                  .join(", ")}`,
            )
            .join("; ")}
        </div>
      </div>
    </ChartContainer>
  );
};
