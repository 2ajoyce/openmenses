import React, { useCallback, useEffect, useRef, useState } from "react";
import { Block, Button, Card, CardContent, CardHeader, Navbar, Page, Segmented } from "framework7-react";
import type { Router } from "framework7/types";
import type { Cycle, CycleStatistics, PhaseEstimate, BiologicalCycleModel, Prediction, Insight } from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import { toLocalDate, fromLocalDate, formatDate } from "../../lib/dates";
import { EmptyState } from "../../components/EmptyState";
import { CycleCard } from "./CycleCard";
import { PhaseEstimateCard } from "./PhaseEstimateCard";
import { CycleLengthChart } from "./CycleLengthChart";
import { CalendarHeatmap } from "./CalendarHeatmap";
import { MoodPhaseChart } from "./MoodPhaseChart";
import { MoodCycleDayChart } from "./MoodCycleDayChart";
import { PredictionCard } from "../prediction/PredictionCard";
import { InsightCard } from "../insight/InsightCard";

interface CyclesPageProps {
  f7router: Router.Router;
}

type WindowSize = "all" | "6" | "12";

const CyclesPage: React.FC<CyclesPageProps> = ({ f7router }) => {
  const [loading, setLoading] = useState(true);
  const [cycles, setCycles] = useState<Cycle[]>([]);
  const [statistics, setStatistics] = useState<CycleStatistics | null>(null);
  const [biologicalCycleModel, setBiologicalCycleModel] = useState<
    BiologicalCycleModel | undefined
  >();
  const [windowSize, setWindowSize] = useState<WindowSize>("all");
  const [todayPhaseEstimate, setTodayPhaseEstimate] = useState<
    PhaseEstimate | undefined
  >();
  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [insights, setInsights] = useState<Insight[]>([]);
  const [profileComplete, setProfileComplete] = useState(true);
  const loadingRef = useRef(false);

  const fetchUserProfile = useCallback(async () => {
    try {
      const res = await client.getUserProfile({ name: DEFAULT_PARENT });
      if (res.profile) {
        setBiologicalCycleModel(res.profile.biologicalCycle);
        // Check if profile has all required fields for phase estimates
        const hasAllFields =
          res.profile.biologicalCycle &&
          res.profile.cycleRegularity &&
          res.profile.trackingFocus &&
          res.profile.trackingFocus.length > 0;
        setProfileComplete(Boolean(hasAllFields));
      }
    } catch {
      // Non-critical: biologicalCycleModel just won't be set
      setProfileComplete(false);
    }
  }, []);

  const fetchCycles = useCallback(async () => {
    try {
      const res = await client.listCycles({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      });
      // Sort cycles by start date, most recent first
      const sorted = [...(res.cycles || [])].sort((a, b) => {
        const aStart = a.startDate ? fromLocalDate(a.startDate).getTime() : 0;
        const bStart = b.startDate ? fromLocalDate(b.startDate).getTime() : 0;
        return bStart - aStart;
      });
      setCycles(sorted);
    } catch (err) {
      console.error("Failed to fetch cycles:", err);
    }
  }, []);

  const fetchStatistics = useCallback(
    async (size: WindowSize = windowSize) => {
      try {
        const windowSizeNum =
          size === "all" ? 0 : size === "6" ? 6 : size === "12" ? 12 : 0;
        const res = await client.getCycleStatistics({
          parent: DEFAULT_PARENT,
          windowSize: windowSizeNum,
        });
        setStatistics(res.statistics || null);
      } catch (err) {
        console.error("Failed to fetch cycle statistics:", err);
        setStatistics(null);
      }
    },
    [windowSize],
  );

  const fetchTodayPhaseEstimate = useCallback(async () => {
    try {
      const today = new Date();
      const res = await client.listTimeline({
        parent: DEFAULT_PARENT,
        range: {
          start: toLocalDate(today),
          end: toLocalDate(today),
        },
        pagination: { pageSize: 10, pageToken: "" },
      });

      // Find the phase estimate record for today
      const phaseRecord = res.records.find(
        (r) => r.record.case === "phaseEstimate",
      );
      if (phaseRecord && phaseRecord.record.case === "phaseEstimate") {
        setTodayPhaseEstimate(phaseRecord.record.value as PhaseEstimate);
      } else {
        setTodayPhaseEstimate(undefined);
      }
    } catch (err) {
      console.error("Failed to fetch today's phase estimate:", err);
    }
  }, []);

  const fetchPredictions = useCallback(async () => {
    try {
      const res = await client.listPredictions({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      });
      setPredictions(res.predictions || []);
    } catch (err) {
      console.error("Failed to fetch predictions:", err);
      setPredictions([]);
    }
  }, []);

  const fetchInsights = useCallback(async () => {
    try {
      const res = await client.listInsights({
        parent: DEFAULT_PARENT,
        pagination: { pageSize: 100, pageToken: "" },
      });
      setInsights(res.insights || []);
    } catch (err) {
      console.error("Failed to fetch insights:", err);
      setInsights([]);
    }
  }, []);

  const handleFetch = useCallback(async () => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setLoading(true);

    try {
      await Promise.all([
        fetchUserProfile(),
        fetchCycles(),
        fetchStatistics(),
        fetchTodayPhaseEstimate(),
        fetchPredictions(),
        fetchInsights(),
      ]);
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  }, [
    fetchUserProfile,
    fetchCycles,
    fetchStatistics,
    fetchTodayPhaseEstimate,
    fetchPredictions,
    fetchInsights,
  ]);

  useEffect(() => {
    handleFetch();
  }, [handleFetch]);

  const handleWindowSizeChange = useCallback(
    async (size: string) => {
      const newSize = size as WindowSize;
      setWindowSize(newSize);
      try {
        const windowSizeNum =
          newSize === "all" ? 0 : newSize === "6" ? 6 : newSize === "12" ? 12 : 0;
        const res = await client.getCycleStatistics({
          parent: DEFAULT_PARENT,
          windowSize: windowSizeNum,
        });
        setStatistics(res.statistics || null);
      } catch (err) {
        console.error("Failed to fetch statistics:", err);
      }
    },
    [],
  );

  function handleRefresh(done: () => void) {
    handleFetch().finally(done);
  }

  // Find current (open-ended) cycle
  const currentCycle = cycles.find((c) => !c.endDate);
  const completedCycles = cycles.filter((c) => c.endDate);

  // Compute day count for current cycle
  const getCurrentCycleDayCount = (cycle: Cycle): number | null => {
    if (!cycle.startDate) return null;
    const start = fromLocalDate(cycle.startDate);
    const today = new Date();
    return Math.floor((today.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1;
  };

  const hasStatistics = statistics && statistics.count > 0;

  return (
    <Page ptr onPtrRefresh={handleRefresh} onPageBeforeIn={handleFetch}>
      <Navbar title="Cycles" />

      {/* Loading state */}
      {loading && cycles.length === 0 && (
        <Block className="om-loading-placeholder">
          <p>Loading cycles...</p>
        </Block>
      )}

      {/* Empty state */}
      {!loading && cycles.length === 0 && (
        <EmptyState
          message="No cycles detected yet"
          actionLabel="Log your first observation"
          onAction={() => f7router.navigate("/log/")}
        />
      )}

      {/* Profile incomplete warning */}
      {!loading && cycles.length > 0 && !profileComplete && (
        <Block className="om-banner-profile-incomplete">
          <Card>
            <CardContent>
              <p>
                Complete your profile to see phase estimates and predictions.
              </p>
              <Button
                small
                fill
                onClick={() => f7router.navigate("/settings/")}
              >
                Complete Profile
              </Button>
            </CardContent>
          </Card>
        </Block>
      )}

      {/* Statistics section */}
      {!loading && hasStatistics && (
        <div role="region" aria-labelledby="stats-heading">
          <Block strong>
          <div className="cycle-stats-card">
            <Card>
              <CardHeader>
                <span className="om-card-title" id="stats-heading">Cycle Statistics</span>
              </CardHeader>
              <CardContent>
                <div className="om-stats-grid">
                  <div className="om-stat-item">
                    <span className="om-stat-label">Average</span>
                    <span className="om-stat-value" aria-label={`Average: ${statistics.average?.toFixed(1) || "—"} days`}>
                      {statistics.average?.toFixed(1) || "—"} days
                    </span>
                  </div>
                  <div className="om-stat-item">
                    <span className="om-stat-label">Median</span>
                    <span className="om-stat-value" aria-label={`Median: ${statistics.median || "—"} days`}>
                      {statistics.median || "—"} days
                    </span>
                  </div>
                  <div className="om-stat-item">
                    <span className="om-stat-label">Min</span>
                    <span className="om-stat-value" aria-label={`Minimum: ${statistics.min || "—"} days`}>
                      {statistics.min || "—"} days
                    </span>
                  </div>
                  <div className="om-stat-item">
                    <span className="om-stat-label">Max</span>
                    <span className="om-stat-value" aria-label={`Maximum: ${statistics.max || "—"} days`}>
                      {statistics.max || "—"} days
                    </span>
                  </div>
                  <div className="om-stat-item">
                    <span className="om-stat-label">Std Dev</span>
                    <span className="om-stat-value" aria-label={`Standard deviation: ${statistics.stdDev?.toFixed(1) || "—"} days`}>
                      {statistics.stdDev?.toFixed(1) || "—"} days
                    </span>
                  </div>
                  <div className="om-stat-item">
                    <span className="om-stat-label">Count</span>
                    <span className="om-stat-value" aria-label={`Count: ${statistics.count}`}>{statistics.count}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Window size selector */}
          <div className="cycle-stats-window-selector">
            <label>Statistics Window:</label>
            <Segmented>
              <Button
                active={windowSize === "6"}
                onClick={() => handleWindowSizeChange("6")}
              >
                Last 6
              </Button>
              <Button
                active={windowSize === "12"}
                onClick={() => handleWindowSizeChange("12")}
              >
                Last 12
              </Button>
              <Button
                active={windowSize === "all"}
                onClick={() => handleWindowSizeChange("all")}
              >
                All
              </Button>
            </Segmented>
          </div>
          </Block>
        </div>
      )}

      {/* Cycle Length Trend Chart */}
      {!loading && completedCycles.length >= 2 && (
        <div role="region" aria-labelledby="chart-heading">
          <Block strong>
          <div className="cycle-length-chart-section">
            <h3 className="om-block-subtitle" id="chart-heading">Cycle Length Trend</h3>
            <CycleLengthChart cycles={cycles} />
          </div>
          </Block>
        </div>
      )}

      {/* Calendar Heatmap */}
      {!loading && cycles.length > 0 && (
        <div role="region" aria-labelledby="heatmap-heading">
          <Block strong>
          <div className="calendar-heatmap-section">
            <h3 className="om-block-subtitle" id="heatmap-heading">Observation Calendar</h3>
            <CalendarHeatmap />
          </div>
          </Block>
        </div>
      )}

      {/* Mood & Cycle Section */}
      {!loading && completedCycles.length >= 2 && (
        <div role="region" aria-labelledby="mood-cycle-heading">
          <Block strong>
          <div className="mood-cycle-section">
            <h3 className="om-block-subtitle" id="mood-cycle-heading">Mood & Cycle</h3>
            <MoodPhaseChart />
            <MoodCycleDayChart />
          </div>
          </Block>
        </div>
      )}

      {/* Current cycle section */}
      {!loading && currentCycle && (
        <div role="region" aria-labelledby="current-cycle-heading">
          <Block strong>
          <div className="current-cycle-section">
            <Card>
              <CardHeader>
                <span className="om-card-title" id="current-cycle-heading">Current Cycle</span>
              </CardHeader>
              <CardContent>
                {currentCycle.startDate && (
                  <>
                    <p className="om-card-timestamp">
                      Started: {formatDate(currentCycle.startDate)}
                    </p>
                    {(() => {
                      const dayCount = getCurrentCycleDayCount(currentCycle);
                      return dayCount ? (
                        <p className="om-card-notes">Day {dayCount}</p>
                      ) : null;
                    })()}
                  </>
                )}
                {profileComplete && todayPhaseEstimate && (
                  <div className="current-phase">
                    <PhaseEstimateCard
                      estimates={[todayPhaseEstimate]}
                      biologicalCycleModel={biologicalCycleModel}
                    />
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
          </Block>
        </div>
      )}

      {/* Predictions section */}
      {!loading && predictions.length > 0 && (
        <div role="region" aria-labelledby="predictions-heading">
          <Block strong>
          <h3 className="om-block-subtitle" id="predictions-heading">Predictions</h3>
          {predictions.map((prediction) => (
            <PredictionCard key={prediction.name} prediction={prediction} />
          ))}
          </Block>
        </div>
      )}

      {/* Insights section */}
      {!loading && insights.length > 0 && (
        <div role="region" aria-labelledby="insights-heading">
          <Block strong>
          <h3 className="om-block-subtitle" id="insights-heading">Insights</h3>
          {insights.map((insight) => (
            <InsightCard key={insight.name} insight={insight} />
          ))}
          </Block>
        </div>
      )}

      {/* Insights empty state */}
      {!loading && insights.length === 0 && cycles.length > 0 && profileComplete && (
        <Block className="om-empty-state">
          <p className="om-empty-state-message">
            Not enough data yet to generate insights. Continue tracking to unlock pattern analysis.
          </p>
        </Block>
      )}

      {/* Cycle history section */}
      {!loading && completedCycles.length > 0 && (
        <div role="region" aria-labelledby="history-heading">
          <Block strong>
          <h3 className="om-block-subtitle" id="history-heading">Cycle History</h3>
          {completedCycles.map((cycle) => (
            <CycleCard key={cycle.name} cycle={cycle} />
          ))}
          </Block>
        </div>
      )}
    </Page>
  );
};

export default CyclesPage;
