import React from "react";
import { Card, CardContent, CardHeader } from "framework7-react";
import type { Cycle } from "@gen/openmenses/v1/model_pb";
import { cycleSourceLabel } from "../../lib/enums";
import { formatDate, fromLocalDate } from "../../lib/dates";

interface CycleCardProps {
  cycle: Cycle;
}

export const CycleCard: React.FC<CycleCardProps> = ({ cycle }) => {
  const startDate = cycle.startDate;
  const endDate = cycle.endDate;
  const isOpen = !endDate;

  // Calculate cycle length
  let cycleLength: number | null = null;
  if (startDate && endDate) {
    const start = fromLocalDate(startDate);
    const end = fromLocalDate(endDate);
    cycleLength = Math.floor((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1;
  }

  return (
    <div className="cycle-card">
      <Card>
        <CardHeader>
          <span className="om-card-title">
            {isOpen ? "Current Cycle" : "Cycle"} — {cycleSourceLabel(cycle.source)}
          </span>
        </CardHeader>
        <CardContent>
          {startDate && (
            <p className="om-card-timestamp">
              Start: {formatDate(startDate)}
            </p>
          )}
          {endDate ? (
            <p className="om-card-timestamp">
              End: {formatDate(endDate)}
            </p>
          ) : (
            <p className="om-card-timestamp om-muted">
              In progress
            </p>
          )}
          {cycleLength !== null && (
            <p className="om-card-notes">
              Length: {cycleLength} days
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
