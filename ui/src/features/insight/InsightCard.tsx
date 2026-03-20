import React, { useState } from "react";
import { Card, CardContent, CardHeader } from "framework7-react";
import type { Insight, Cycle, BleedingObservation } from "@gen/openmenses/v1/model_pb";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import type { RecordRef } from "@gen/openmenses/v1/types_pb";
import { insightTypeLabel, confidenceLevelLabel, bleedingFlowLabel } from "../../lib/enums";
import { formatDate, fromLocalDate, fromDateTime } from "../../lib/dates";

interface InsightCardProps {
  insight: Insight;
  recordLookup?: Record<string, TimelineRecord>;
}

export const InsightCard: React.FC<InsightCardProps> = ({ insight, recordLookup }) => {
  const [evidenceExpanded, setEvidenceExpanded] = useState(false);
  const typeLabel = insightTypeLabel(insight.kind!);
  const confidence = insight.confidence!;
  const refs = insight.evidenceRecordRefs;
  const lookup = recordLookup ?? {};
  const resolvedRefs = refs.filter((ref) => ref.name in lookup);
  const hiddenCount = refs.length - resolvedRefs.length;

  return (
    <div className="insight-card">
      <Card>
        <CardHeader>
          <span className="om-card-title">{typeLabel}</span>
        </CardHeader>
        <CardContent>
          <p className="om-card-notes">{insight.summary}</p>
          <div className="om-confidence-badge">
            <span
              className="om-confidence-indicator"
              style={{ backgroundColor: getConfidenceColor(confidence) }}
            />
            <span className="om-confidence-text">
              Confidence: {confidenceLevelLabel(confidence)}
            </span>
          </div>
          {refs && refs.length > 0 && (
            <div className="om-insight-evidence">
              <button
                className="om-insight-evidence-toggle"
                onClick={() => setEvidenceExpanded((e) => !e)}
                aria-expanded={evidenceExpanded}
              >
                Based on {refs.length} {refs.length === 1 ? "record" : "records"}
                <span className="om-insight-evidence-chevron">
                  {evidenceExpanded ? "▴" : "▾"}
                </span>
              </button>
              {evidenceExpanded && (
                <>
                  {resolvedRefs.length > 0 && (
                    <ul className="om-insight-evidence-list">
                      {resolvedRefs.map((ref, index) => (
                        <li key={index}>{formatRef(ref, lookup)}</li>
                      ))}
                    </ul>
                  )}
                  {hiddenCount > 0 && (
                    <p className="om-insight-evidence-note">
                      {hiddenCount} {hiddenCount === 1 ? "record is" : "records are"} outside the current timeline range.
                    </p>
                  )}
                </>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

function formatRef(ref: RecordRef, lookup: Record<string, TimelineRecord>): string {
  const record = lookup[ref.name]!;

  switch (record.record.case) {
    case "cycle": {
      const cycle = record.record.value as Cycle;
      if (!cycle.startDate?.value) return "Cycle";
      const start = formatDate(cycle.startDate);
      if (!cycle.endDate?.value) return `${start} · ongoing`;
      const end = formatDate(cycle.endDate);
      const startMs = fromLocalDate(cycle.startDate).getTime();
      const endMs = fromLocalDate(cycle.endDate).getTime();
      const days = Math.round((endMs - startMs) / 86400000) + 1;
      return `${start} – ${end} · ${days} days`;
    }
    case "bleedingObservation": {
      const obs = record.record.value as BleedingObservation;
      if (!obs.timestamp?.value) return "Bleeding observation";
      const date = fromDateTime(obs.timestamp).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
      });
      return `${date} · ${bleedingFlowLabel(obs.flow)}`;
    }
    default:
      return record.record.case ?? "Record";
  }
}

function getConfidenceColor(confidence: number): string {
  switch (confidence) {
    case 1: // LOW
      return "var(--om-color-confidence-low)";
    case 2: // MEDIUM
      return "var(--om-color-confidence-medium)";
    case 3: // HIGH
      return "var(--om-color-confidence-high)";
    default:
      return "#999";
  }
}
