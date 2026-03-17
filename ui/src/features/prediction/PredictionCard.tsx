import React from "react";
import { Card, CardContent, CardHeader } from "framework7-react";
import type { Prediction } from "@gen/openmenses/v1/model_pb";
import { predictionTypeLabel, confidenceLevelLabel } from "../../lib/enums";
import { formatDate } from "../../lib/dates";

interface PredictionCardProps {
  prediction: Prediction;
}

export const PredictionCard: React.FC<PredictionCardProps> = ({ prediction }) => {
  const kindLabel = predictionTypeLabel(prediction.kind!);
  const startDate = prediction.predictedStartDate;
  const endDate = prediction.predictedEndDate;
  const confidence = prediction.confidence!;

  // Determine prediction color token based on prediction type
  const getPredictionColorToken = (kind: number): string => {
    switch (kind) {
      case 1: // NEXT_BLEED
        return "var(--om-color-prediction-bleed)";
      case 2: // PMS_WINDOW
        return "var(--om-color-prediction-pms)";
      case 3: // OVULATION_WINDOW
        return "var(--om-color-prediction-ovulation)";
      case 4: // SYMPTOM_WINDOW
        return "var(--om-color-prediction-symptom)";
      default:
        return "var(--om-color-prediction)";
    }
  };

  const predictionColor = getPredictionColorToken(prediction.kind!);

  return (
    <div className="prediction-card">
      <Card
        style={{ "--om-prediction-color": predictionColor } as React.CSSProperties}
      >
        <CardHeader>
          <span className="om-card-title">{kindLabel}</span>
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
          {prediction.rationale && prediction.rationale.length > 0 && (
            <div className="om-prediction-rationale">
              <ul>
                {prediction.rationale.map((reason, index) => (
                  <li key={index}>{reason}</li>
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
