import React from "react";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ReferenceLine, ResponsiveContainer } from "recharts";
import type { Cycle } from "@gen/openmenses/v1/model_pb";
import { fromLocalDate } from "../../lib/dates";
import { ChartContainer } from "../../components/ChartContainer";

interface CycleLengthChartProps {
  cycles: Cycle[];
}

export const CycleLengthChart: React.FC<CycleLengthChartProps> = ({ cycles }) => {
  // Filter to completed cycles only and sort by start date (oldest first)
  const completedCycles = cycles
    .filter((c) => c.startDate && c.endDate)
    .sort((a, b) => {
      const aStart = a.startDate ? fromLocalDate(a.startDate).getTime() : 0;
      const bStart = b.startDate ? fromLocalDate(b.startDate).getTime() : 0;
      return aStart - bStart;
    });

  // Need at least 2 cycles to show chart
  if (completedCycles.length < 2) {
    return null;
  }

  // Compute cycle length for each cycle
  const data = completedCycles.map((cycle, index) => {
    const start = cycle.startDate ? fromLocalDate(cycle.startDate) : new Date();
    const end = cycle.endDate ? fromLocalDate(cycle.endDate) : new Date();
    const length = Math.floor((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1;
    return {
      cycleIndex: index + 1,
      length,
      startDate: cycle.startDate,
      endDate: cycle.endDate,
    };
  });

  // Compute average cycle length
  const averageLength = data.length > 0 ? data.reduce((sum, d) => sum + d.length, 0) / data.length : 0;

  // Text summary for screen readers
  const summaryText = `Cycle length trend over ${data.length} cycles. Average cycle length is ${averageLength.toFixed(1)} days. `;
  const dataSummary = data.map((d) => `Cycle ${d.cycleIndex}: ${d.length} days`).join(", ");

  return (
    <ChartContainer data={data} title="Cycle Length Trend">
      <div role="img" aria-label="Line chart showing cycle length over time">
        <ResponsiveContainer width="100%" height={300}>
          <LineChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--om-color-chart-grid)" />
            <XAxis dataKey="cycleIndex" label={{ value: "Cycle #", position: "insideBottomRight", offset: -5 }} />
            <YAxis label={{ value: "Length (days)", angle: -90, position: "insideLeft" }} />
            <Tooltip
              contentStyle={{
                backgroundColor: "var(--om-color-chart-tooltip-bg)",
                border: `1px solid var(--om-color-chart-tooltip-border)`,
              }}
              formatter={(value) => `${value} days`}
              labelFormatter={(label) => `Cycle ${label}`}
            />
            <ReferenceLine y={averageLength} stroke="var(--om-color-primary)" strokeDasharray="5 5" label="Average" />
            <Line
              type="monotone"
              dataKey="length"
              stroke="var(--om-color-primary)"
              dot={{ fill: "var(--om-color-primary)", r: 4 }}
              activeDot={{ r: 6 }}
            />
          </LineChart>
        </ResponsiveContainer>
        <div className="om-sr-only">
          {summaryText}
          {dataSummary}
        </div>
      </div>
    </ChartContainer>
  );
};
