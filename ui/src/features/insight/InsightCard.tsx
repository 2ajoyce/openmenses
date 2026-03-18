import React from "react";
import { Card, CardContent, CardHeader } from "framework7-react";
import type { Insight } from "@gen/openmenses/v1/model_pb";
import { insightTypeLabel, confidenceLevelLabel } from "../../lib/enums";

interface InsightCardProps {
  insight: Insight;
}

export const InsightCard: React.FC<InsightCardProps> = ({ insight }) => {
  const typeLabel = insightTypeLabel(insight.kind!);
  const confidence = insight.confidence!;

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
          {insight.evidenceRecordRefs && insight.evidenceRecordRefs.length > 0 && (
            <div className="om-insight-evidence">
              <strong>Evidence:</strong>
              <ul>
                {insight.evidenceRecordRefs.map((ref, index) => (
                  <li key={index}>{ref.name}</li>
                ))}
              </ul>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

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
