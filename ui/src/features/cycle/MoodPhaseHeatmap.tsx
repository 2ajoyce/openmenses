import type {
  Cycle,
  MoodObservation,
  PhaseEstimate,
} from "@gen/openmenses/v1/model_pb";
import {
  CyclePhase,
  MoodIntensity,
  MoodType,
} from "@gen/openmenses/v1/model_pb";
import React, { useEffect, useState } from "react";
import { ChartContainer } from "../../components/ChartContainer";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { fromDateTime, fromLocalDate, toLocalDate } from "../../lib/dates";
import { cyclePhaseLabel, moodTypeLabel } from "../../lib/enums";

const ORDERED_PHASES: CyclePhase[] = [
  CyclePhase.MENSTRUATION,
  CyclePhase.FOLLICULAR,
  CyclePhase.OVULATION_WINDOW,
  CyclePhase.LUTEAL,
];

const PHASE_SHORT_LABELS: Record<CyclePhase, string> = {
  [CyclePhase.MENSTRUATION]: "Mens.",
  [CyclePhase.FOLLICULAR]: "Foll.",
  [CyclePhase.OVULATION_WINDOW]: "Ovul.",
  [CyclePhase.LUTEAL]: "Luteal",
  [CyclePhase.UNSPECIFIED]: "",
  [CyclePhase.UNKNOWN]: "",
};

interface CellData {
  avgIntensity: number;
  count: number;
}

type HeatmapData = Map<MoodType, Map<CyclePhase, CellData>>;

function intensityOpacity(avg: number): number {
  if (avg < 1.5) return 0.2;
  if (avg < 2.5) return 0.5;
  return 0.85;
}

function intensityLabel(avg: number): string {
  if (avg < 1.5) return "Low";
  if (avg < 2.5) return "Med";
  return "High";
}

export const MoodPhaseHeatmap: React.FC = () => {
  const [heatmapData, setHeatmapData] = useState<HeatmapData>(new Map());
  const [activeMoodTypes, setActiveMoodTypes] = useState<MoodType[]>([]);
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

        if (moods.length === 0 || cycles.length === 0) {
          setHeatmapData(new Map());
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

        // Accumulate intensity sums and counts per (MoodType, CyclePhase)
        const accumulator = new Map<
          MoodType,
          Map<CyclePhase, { sum: number; count: number }>
        >();

        moods.forEach((mood: MoodObservation) => {
          if (
            !mood.timestamp ||
            mood.mood === undefined ||
            mood.intensity === undefined ||
            mood.intensity === MoodIntensity.UNSPECIFIED
          )
            return;

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
            assignedPhase === undefined ||
            !ORDERED_PHASES.includes(assignedPhase)
          )
            return;

          const moodType = mood.mood;
          if (!accumulator.has(moodType)) {
            accumulator.set(moodType, new Map());
          }
          const phaseMap = accumulator.get(moodType)!;
          if (!phaseMap.has(assignedPhase)) {
            phaseMap.set(assignedPhase, { sum: 0, count: 0 });
          }
          const cell = phaseMap.get(assignedPhase)!;
          cell.sum += mood.intensity;
          cell.count += 1;
        });

        // Build final CellData map
        const result: HeatmapData = new Map();
        for (const [moodType, phaseMap] of accumulator.entries()) {
          const cellMap = new Map<CyclePhase, CellData>();
          for (const [phase, { sum, count }] of phaseMap.entries()) {
            cellMap.set(phase, { avgIntensity: sum / count, count });
          }
          result.set(moodType, cellMap);
        }

        const activeMoods = Array.from(result.keys()).sort((a, b) => a - b);
        setHeatmapData(result);
        setActiveMoodTypes(activeMoods);
      } catch (err) {
        console.error("Failed to fetch mood phase heatmap data:", err);
        setHeatmapData(new Map());
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) return null;
  if (heatmapData.size === 0) return null;

  return (
    <ChartContainer
      data={activeMoodTypes}
      title="Mood Intensity by Phase"
      description="Shows average mood intensity across each cycle phase. Darker cells indicate higher average intensity."
      emptyMessage="Track moods across at least one full cycle to see mood intensity patterns by phase."
    >
      <div
        role="img"
        aria-label="Heatmap grid showing average mood intensity across cycle phases"
      >
        <div className="om-mood-heatmap">
          {/* Corner + phase column headers */}
          <div className="om-mood-heatmap-corner" />
          {ORDERED_PHASES.map((phase) => (
            <div key={phase} className="om-mood-heatmap-header">
              {PHASE_SHORT_LABELS[phase]}
            </div>
          ))}

          {/* One row per active mood type */}
          {activeMoodTypes.map((moodType, moodColorIndex) => {
            const phaseMap = heatmapData.get(moodType);
            return (
              <React.Fragment key={moodType}>
                <div className="om-mood-heatmap-row-label">
                  {moodTypeLabel(moodType)}
                </div>
                {ORDERED_PHASES.map((phase) => {
                  const cell = phaseMap?.get(phase);
                  if (!cell) {
                    return (
                      <div
                        key={phase}
                        className="om-mood-heatmap-cell om-mood-heatmap-cell--empty"
                        aria-label={`${moodTypeLabel(moodType)} in ${cyclePhaseLabel(phase)}: no data`}
                      />
                    );
                  }
                  return (
                    <div
                      key={phase}
                      className="om-mood-heatmap-cell"
                      style={
                        {
                          "--om-cell-rgb": `var(--om-color-mood-rgb-${moodColorIndex})`,
                          "--om-cell-opacity": String(
                            intensityOpacity(cell.avgIntensity),
                          ),
                        } as React.CSSProperties
                      }
                      aria-label={`${moodTypeLabel(moodType)} in ${cyclePhaseLabel(phase)}: ${intensityLabel(cell.avgIntensity)}`}
                      title={`${cell.count} observation${cell.count !== 1 ? "s" : ""}, avg ${cell.avgIntensity.toFixed(1)}`}
                    >
                      {intensityLabel(cell.avgIntensity)}
                    </div>
                  );
                })}
              </React.Fragment>
            );
          })}
        </div>

        {/* Intensity legend */}
        <div className="om-mood-heatmap-intensity-legend">
          <span className="om-mood-heatmap-intensity-legend-label">
            Intensity:
          </span>
          {(
            [
              { label: "Low", opacity: "0.2" },
              { label: "Med", opacity: "0.5" },
              { label: "High", opacity: "0.85" },
            ] as const
          ).map(({ label, opacity }) => (
            <div key={label} className="om-mood-heatmap-intensity-legend-item">
              <div
                className="om-mood-heatmap-intensity-swatch"
                style={
                  {
                    "--om-cell-rgb": "var(--om-color-mood-rgb-0)",
                    "--om-cell-opacity": opacity,
                  } as React.CSSProperties
                }
              />
              <span>{label}</span>
            </div>
          ))}
        </div>

        <div className="om-sr-only">
          Grid showing average mood intensity by cycle phase. Rows are mood
          types, columns are cycle phases.
          {activeMoodTypes
            .map((mt) => {
              const phaseMap = heatmapData.get(mt);
              return `${moodTypeLabel(mt)}: ${ORDERED_PHASES.map((phase) => {
                const cell = phaseMap?.get(phase);
                return `${cyclePhaseLabel(phase)}: ${cell ? intensityLabel(cell.avgIntensity) : "no data"}`;
              }).join(", ")}`;
            })
            .join("; ")}
        </div>
      </div>
    </ChartContainer>
  );
};
