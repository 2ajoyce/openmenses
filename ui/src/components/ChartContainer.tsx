import React from "react";
import { ResponsiveContainer } from "recharts";
import { EmptyState } from "./EmptyState";

interface ChartContainerProps {
  data?: unknown[];
  children: React.ReactElement;
  title?: string;
}

export const ChartContainer: React.FC<ChartContainerProps> = ({
  data,
  children,
  title,
}) => {
  const hasData = data && data.length > 0;

  return (
    <div className="om-chart-container">
      {hasData ? (
        <>
          {title && <h3>{title}</h3>}
          <ResponsiveContainer width="100%" height={300}>
            {children}
          </ResponsiveContainer>
        </>
      ) : (
        <EmptyState message="No data available for this chart" icon="chart_bar" />
      )}
    </div>
  );
};
