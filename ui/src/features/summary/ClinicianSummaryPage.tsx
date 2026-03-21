import React, { useEffect, useState } from "react";
import { Page, Navbar, Button, Block } from "framework7-react";
import type { Router } from "framework7/types";
import type {
  UserProfile,
  CycleStatistics,
  Cycle,
  Medication,
  Prediction,
  Insight,
} from "@gen/openmenses/v1/model_pb";
import { client, DEFAULT_PARENT } from "../../lib/client";
import {
  biologicalCycleModelLabel,
  cycleRegularityLabel,
  cycleSourceLabel,
  predictionTypeLabel,
  confidenceLevelLabel,
  insightTypeLabel,
  medicationCategoryLabel,
} from "../../lib/enums";
import { formatDate } from "../../lib/dates";

interface ClinicianSummaryPageProps {
  f7router: Router.Router;
}

const ClinicianSummaryPage: React.FC<ClinicianSummaryPageProps> = () => {
  const [loading, setLoading] = useState(true);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [statistics, setStatistics] = useState<CycleStatistics | null>(null);
  const [cycles, setCycles] = useState<Cycle[]>([]);
  const [medications, setMedications] = useState<Medication[]>([]);
  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [insights, setInsights] = useState<Insight[]>([]);

  useEffect(() => {
    const fetchAllData = async () => {
      try {
        setLoading(true);

        // Fetch all required data in parallel
        const [
          profileRes,
          statsRes,
          cyclesRes,
          medicationsRes,
          predictionsRes,
          insightsRes,
        ] = await Promise.all([
          client.getUserProfile({ name: DEFAULT_PARENT }),
          client.getCycleStatistics({ parent: DEFAULT_PARENT, windowSize: 0 }),
          client.listCycles({
            parent: DEFAULT_PARENT,
            pagination: { pageSize: 6, pageToken: "" },
          }),
          client.listMedications({
            parent: DEFAULT_PARENT,
            pagination: { pageSize: 100, pageToken: "" },
          }),
          client.listPredictions({
            parent: DEFAULT_PARENT,
            pagination: { pageSize: 100, pageToken: "" },
          }),
          client.listInsights({
            parent: DEFAULT_PARENT,
            pagination: { pageSize: 100, pageToken: "" },
          }),
        ]);

        setProfile(profileRes.profile || null);
        setStatistics(statsRes.statistics || null);

        // Sort cycles by start date, most recent first
        const sortedCycles = [...(cyclesRes.cycles || [])].sort((a, b) => {
          const aDate = a.startDate
            ? new Date(a.startDate.value).getTime()
            : 0;
          const bDate = b.startDate
            ? new Date(b.startDate.value).getTime()
            : 0;
          return bDate - aDate;
        });
        setCycles(sortedCycles.slice(0, 6));

        // Filter for active medications (only show active ones)
        const activeMeds = (medicationsRes.medications || []).filter(
          (med) => med.active
        );
        setMedications(activeMeds);

        setPredictions(predictionsRes.predictions || []);
        setInsights(insightsRes.insights || []);
      } catch (err) {
        console.error("Failed to fetch clinician summary data:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchAllData();
  }, []);

  const handlePrint = () => {
    window.print();
  };

  if (loading) {
    return (
      <Page pageContent={false}>
        <div className="page-content">
        <Navbar title="Clinician Summary" />
        <Block className="text-align-center">
          <p className="om-muted">Loading summary...</p>
        </Block>
        </div>
      </Page>
    );
  }

  const generationDate = new Date().toLocaleDateString("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  return (
    <Page pageContent={false} className="clinician-summary">
      <div className="page-content">
      <Navbar title="Clinician Summary" />

      <Block>
        <Button onClick={handlePrint} fill large className="print-button">
          Print Summary
        </Button>
      </Block>

      <div className="summary-content">
        {/* Header Section */}
        <section className="summary-section">
          <h1>Cycle Health Summary</h1>
          <p className="summary-generation-date">
            Generated: {generationDate}
          </p>
        </section>

        {/* Profile Section */}
        <section className="summary-section">
          <h2>Profile</h2>
          {profile ? (
            <dl className="summary-definition-list">
              <div className="definition-item">
                <dt>Biological Cycle Model:</dt>
                <dd>{biologicalCycleModelLabel(profile.biologicalCycle)}</dd>
              </div>
              <div className="definition-item">
                <dt>Cycle Regularity:</dt>
                <dd>{cycleRegularityLabel(profile.cycleRegularity)}</dd>
              </div>
            </dl>
          ) : (
            <p className="no-data">No profile data available</p>
          )}
        </section>

        {/* Statistics Section */}
        <section className="summary-section">
          <h2>Cycle Statistics</h2>
          {statistics ? (
            <table className="summary-table">
              <caption>Cycle statistics summary</caption>
              <thead>
                <tr>
                  <th scope="col">Metric</th>
                  <th scope="col">Value</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>Average Length</td>
                  <td>{statistics.average?.toFixed(1) || "—"} days</td>
                </tr>
                <tr>
                  <td>Median Length</td>
                  <td>{statistics.median?.toFixed(1) || "—"} days</td>
                </tr>
                <tr>
                  <td>Minimum Length</td>
                  <td>{statistics.min || "—"} days</td>
                </tr>
                <tr>
                  <td>Maximum Length</td>
                  <td>{statistics.max || "—"} days</td>
                </tr>
                <tr>
                  <td>Standard Deviation</td>
                  <td>{statistics.stdDev?.toFixed(1) || "—"} days</td>
                </tr>
                <tr>
                  <td>Cycle Count</td>
                  <td>{statistics.count || 0}</td>
                </tr>
              </tbody>
            </table>
          ) : (
            <p className="no-data">No statistics available</p>
          )}
        </section>

        {/* Recent Cycles Section */}
        <section className="summary-section">
          <h2>Recent Cycles</h2>
          {cycles.length > 0 ? (
            <table className="summary-table">
              <caption>Last {cycles.length} recorded cycles</caption>
              <thead>
                <tr>
                  <th scope="col">Start Date</th>
                  <th scope="col">End Date</th>
                  <th scope="col">Length</th>
                  <th scope="col">Source</th>
                </tr>
              </thead>
              <tbody>
                {cycles.map((cycle, idx) => {
                  const startDateStr = cycle.startDate
                    ? formatDate(cycle.startDate)
                    : "—";
                  const endDateStr = cycle.endDate
                    ? formatDate(cycle.endDate)
                    : "—";
                  let lengthStr = "—";
                  if (cycle.startDate && cycle.endDate) {
                    const start = new Date(cycle.startDate.value);
                    const end = new Date(cycle.endDate.value);
                    const days = Math.round(
                      (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)
                    );
                    lengthStr = `${days} days`;
                  }
                  return (
                    <tr key={idx}>
                      <td>{startDateStr}</td>
                      <td>{endDateStr}</td>
                      <td>{lengthStr}</td>
                      <td>
                        {cycle.source ? cycleSourceLabel(cycle.source) : "—"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <p className="no-data">No cycle data available</p>
          )}
        </section>

        {/* Active Medications Section */}
        <section className="summary-section">
          <h2>Active Medications</h2>
          {medications.length > 0 ? (
            <table className="summary-table">
              <caption>Currently active medications</caption>
              <thead>
                <tr>
                  <th scope="col">Medication Name</th>
                  <th scope="col">Category</th>
                </tr>
              </thead>
              <tbody>
                {medications.map((med, idx) => (
                  <tr key={idx}>
                    <td>{med.displayName || "—"}</td>
                    <td>
                      {med.category
                        ? medicationCategoryLabel(med.category)
                        : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="no-data">No active medications</p>
          )}
        </section>

        {/* Current Predictions Section */}
        <section className="summary-section">
          <h2>Current Predictions</h2>
          {predictions.length > 0 ? (
            <table className="summary-table">
              <caption>Active cycle predictions</caption>
              <thead>
                <tr>
                  <th scope="col">Prediction Type</th>
                  <th scope="col">Date Range</th>
                  <th scope="col">Confidence</th>
                </tr>
              </thead>
              <tbody>
                {predictions.map((pred, idx) => {
                  const dateRangeStr =
                    pred.predictedStartDate && pred.predictedEndDate
                      ? `${formatDate(pred.predictedStartDate)} to ${formatDate(pred.predictedEndDate)}`
                      : "—";
                  return (
                    <tr key={idx}>
                      <td>{predictionTypeLabel(pred.kind)}</td>
                      <td>{dateRangeStr}</td>
                      <td>
                        {pred.confidence
                          ? confidenceLevelLabel(pred.confidence)
                          : "—"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <p className="no-data">No predictions available</p>
          )}
        </section>

        {/* Insights Section */}
        <section className="summary-section">
          <h2>Insights</h2>
          {insights.length > 0 ? (
            <div className="insights-list">
              {insights.map((insight, idx) => (
                <div key={idx} className="insight-item">
                  <h3 className="insight-title">
                    {insightTypeLabel(insight.kind)}
                  </h3>
                  <p className="insight-summary">{insight.summary || "—"}</p>
                  {insight.confidence && (
                    <p className="insight-confidence">
                      Confidence: {confidenceLevelLabel(insight.confidence)}
                    </p>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="no-data">No insights available</p>
          )}
        </section>
      </div>
      </div>
    </Page>
  );
};

export default ClinicianSummaryPage;
