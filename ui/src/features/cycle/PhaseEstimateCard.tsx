import React from "react";
import { Card, CardContent, CardHeader } from "framework7-react";
import type { PhaseEstimate, BiologicalCycleModel } from "@gen/openmenses/v1/model_pb";
import { BiologicalCycleModel as BiologicalCycleModelEnum } from "@gen/openmenses/v1/model_pb";
import { cyclePhaseLabel, suppressedCyclePhaseLabel, confidenceLevelLabel } from "../../lib/enums";
import { formatDate } from "../../lib/dates";

interface PhaseEstimateCardProps {
  estimates: PhaseEstimate[];
  biologicalCycleModel?: BiologicalCycleModel | undefined;
}

export const PhaseEstimateCard: React.FC<PhaseEstimateCardProps> = ({
  estimates,
  biologicalCycleModel,
}) => {
  if (estimates.length === 0) {
    return null;
  }

  // All estimates should be the same phase (pre-grouped)
  const phase = estimates[0]!.phase;
  const confidence = estimates[0]!.confidence;

  // Get date range
  const startDate = estimates[0]!.date;
  const endDate = estimates[estimates.length - 1]!.date;

  // Determine phase label based on biological cycle model
  const isHormonallySuppressed =
    biologicalCycleModel === BiologicalCycleModelEnum.HORMONALLY_SUPPRESSED;
  const phaseLabel = isHormonallySuppressed
    ? suppressedCyclePhaseLabel(phase)
    : cyclePhaseLabel(phase);

  // Determine phase color token based on phase
  const getPhaseColorToken = (phaseValue: number): string => {
    switch (phaseValue) {
      case 1: // MENSTRUATION
        return "var(--om-color-phase-menstruation)";
      case 2: // FOLLICULAR
        return "var(--om-color-phase-follicular)";
      case 3: // OVULATION_WINDOW
        return "var(--om-color-phase-ovulation)";
      case 4: // LUTEAL
        return "var(--om-color-phase-luteal)";
      default:
        return "var(--om-color-phase)";
    }
  };

  const phaseColor = getPhaseColorToken(phase);

  return (
    <div className="phase-estimate-card">
      <Card style={{ "--om-phase-color": phaseColor } as React.CSSProperties}>
        <CardHeader>
          <span className="om-card-title">{phaseLabel}</span>
        </CardHeader>
        <CardContent>
          {startDate && endDate && (
            <p className="om-card-timestamp">
              {formatDate(startDate)} – {formatDate(endDate)}
            </p>
          )}
          {startDate && !endDate && (
            <p className="om-card-timestamp">
              {formatDate(startDate)}
            </p>
          )}
          <div className="om-confidence-badge">
            <span
              className="om-confidence-indicator"
              style={{ backgroundColor: getConfidenceColor(confidence) }}
            />
            <span className="om-confidence-text">
              Confidence: {confidenceLevelLabel(confidence)}
            </span>
          </div>
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
