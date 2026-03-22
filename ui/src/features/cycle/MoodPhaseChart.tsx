import type {
  Cycle,
  MoodObservation,
  PhaseEstimate,
} from "@gen/openmenses/v1/model_pb";
import { CyclePhase, MoodType } from "@gen/openmenses/v1/model_pb";
import React, { useEffect, useState } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { ChartContainer } from "../../components/ChartContainer";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { fromDateTime, fromLocalDate, toLocalDate } from "../../lib/dates";
import { cyclePhaseLabel, moodTypeLabel } from "../../lib/enums";

interface MoodPhaseData {
  phase: string;
  [moodTypeLabel: string]: string | number;
}

export const MoodPhaseChart: React.FC = () => {
  const [data, setData] = useState<MoodPhaseData[]>([]);
  const [activeMoodTypes, setActiveMoodTypes] = useState<string[]>([]);
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

        // Fetch all cycles to determine phase date ranges
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

        if (moods.length === 0 || cycles.length === 0) {
          setData([]);
          return;
        }

        // Build a date → CyclePhase map from backend phase estimates
        const phaseDateMap = new Map<string, CyclePhase>();
        const completedCycles = cycles.filter((c) => c.startDate && c.endDate);
        if (completedCycles.length > 0) {
          const startTimes = completedCycles.map((c) =>
            fromLocalDate(c.startDate!).getTime(),
          );
          const endTimes = completedCycles.map((c) =>
            fromLocalDate(c.endDate!).getTime(),
          );
          const rangeStart = new Date(Math.min(...startTimes));
          const rangeEnd = new Date(Math.max(...endTimes));
          let timelinePageToken = "";
          do {
            const timelineRes = await client.listTimeline({
              parent: DEFAULT_PARENT,
              range: {
                start: toLocalDate(rangeStart),
                end: toLocalDate(rangeEnd),
              },
              pagination: { pageSize: 500, pageToken: timelinePageToken },
            });
            for (const record of timelineRes.records ?? []) {
              if (record.record.case === "phaseEstimate") {
                const pe = record.record.value as PhaseEstimate;
                if (pe.date?.value) {
                  phaseDateMap.set(pe.date.value, pe.phase);
                }
              }
            }
            timelinePageToken = timelineRes.pagination?.nextPageToken ?? "";
          } while (timelinePageToken);
        }

        const ORDERED_PHASES: CyclePhase[] = [
          CyclePhase.MENSTRUATION,
          CyclePhase.FOLLICULAR,
          CyclePhase.OVULATION_WINDOW,
          CyclePhase.LUTEAL,
        ];

        const phaseCounters = new Map<CyclePhase, Map<number, number>>(
          ORDERED_PHASES.map((p) => [p, new Map()]),
        );

        moods.forEach((mood: MoodObservation) => {
          if (!mood.timestamp || mood.mood === undefined) return;

          const moodDate = fromDateTime(mood.timestamp);
          const dateStr = toLocalDate(moodDate).value;
          let assignedPhase = phaseDateMap.get(dateStr);

          // Fall back to cycle-day arithmetic when no backend estimate exists
          if (
            assignedPhase === undefined ||
            assignedPhase === CyclePhase.UNSPECIFIED
          ) {
            for (const cycle of cycles) {
              if (!cycle.startDate || !cycle.endDate) continue;
              const cycleStart = fromLocalDate(cycle.startDate);
              const cycleEnd = fromLocalDate(cycle.endDate);
              if (moodDate >= cycleStart && moodDate <= cycleEnd) {
                const dayInCycle = Math.floor(
                  (moodDate.getTime() - cycleStart.getTime()) /
                    (1000 * 60 * 60 * 24),
                );
                if (dayInCycle < 5) assignedPhase = CyclePhase.MENSTRUATION;
                else if (dayInCycle < 12) assignedPhase = CyclePhase.FOLLICULAR;
                else if (dayInCycle < 17)
                  assignedPhase = CyclePhase.OVULATION_WINDOW;
                else assignedPhase = CyclePhase.LUTEAL;
                break;
              }
            }
          }

          if (
            assignedPhase !== undefined &&
            ORDERED_PHASES.includes(assignedPhase)
          ) {
            const phaseMap = phaseCounters.get(assignedPhase)!;
            phaseMap.set(mood.mood, (phaseMap.get(mood.mood) ?? 0) + 1);
          }
        });

        // Derive active mood types (those with at least one observation)
        const seenMoodTypes = new Set<number>();
        for (const countsMap of phaseCounters.values()) {
          for (const [mt, count] of countsMap.entries()) {
            if (count > 0) seenMoodTypes.add(mt);
          }
        }
        const activeMoodTypeLabels = Array.from(seenMoodTypes)
          .sort((a, b) => a - b)
          .map((n) => moodTypeLabel(n as MoodType));

        // Transform into percentage-normalized chart data
        const chartData: MoodPhaseData[] = ORDERED_PHASES.map((phase) => {
          const countsMap = phaseCounters.get(phase)!;
          const total = Array.from(countsMap.values()).reduce(
            (s, c) => s + c,
            0,
          );
          const row: MoodPhaseData = { phase: cyclePhaseLabel(phase) };
          for (const moodNum of seenMoodTypes) {
            const label = moodTypeLabel(moodNum as MoodType);
            row[label] =
              total > 0
                ? parseFloat(
                    (((countsMap.get(moodNum) ?? 0) / total) * 100).toFixed(1),
                  )
                : 0;
          }
          return row;
        });

        setActiveMoodTypes(activeMoodTypeLabels);
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

  if (loading) return null;
  if (data.length === 0) return null;

  const summaryText = `Percentage-normalized bar chart showing mood type distribution across cycle phases.`;

  return (
    <ChartContainer
      data={data}
      title="Mood by Phase"
      description="Shows how often each mood appears within each cycle phase, as a percentage of all moods recorded in that phase."
      emptyMessage="Track moods across at least one full cycle to see mood patterns by phase."
    >
      <div
        role="img"
        aria-label="Grouped bar chart showing mood distribution across menstrual cycle phases"
      >
        <ResponsiveContainer width="100%" height={300}>
          <BarChart
            data={data}
            margin={{ top: 20, right: 30, left: 0, bottom: 5 }}
          >
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="var(--om-color-chart-grid)"
            />
            <XAxis dataKey="phase" />
            <YAxis tickFormatter={(v: number) => `${v}%`} domain={[0, 100]} />
            <Tooltip
              contentStyle={{
                backgroundColor: "var(--om-color-chart-tooltip-bg)",
                border: `1px solid var(--om-color-chart-tooltip-border)`,
              }}
              formatter={(value: number) => [`${value}%`]}
            />
            <Legend />
            {activeMoodTypes.map((moodLabel, index) => (
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
          {data
            .map(
              (row) =>
                `${row.phase}: ${Object.entries(row)
                  .filter(([k]) => k !== "phase")
                  .map(([k, v]) => `${k}: ${v}%`)
                  .join(", ")}`,
            )
            .join("; ")}
        </div>
      </div>
    </ChartContainer>
  );
};
