import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TimelineItem } from "../TimelineItem";
import type { TimelineRecord } from "@gen/openmenses/v1/service_pb";
import { InsightType, ConfidenceLevel } from "@gen/openmenses/v1/model_pb";

vi.mock("framework7-react", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div data-testid="card">{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="card-header">{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div data-testid="card-content">{children}</div>,
}));

const noop = () => {};

describe("TimelineItem prediction case", () => {
  it("renders PredictionCard for prediction records", () => {
    const record = {
      record: {
        case: "prediction" as const,
        value: {
          name: "users/default/predictions/01",
          userId: "users/default",
          kind: 1, // NEXT_BLEED
          predictedStartDate: { value: "2026-04-01" },
          predictedEndDate: { value: "2026-04-06" },
          confidence: 3, // HIGH
          rationale: ["Based on 3 completed cycles"],
        },
      },
    } as unknown as TimelineRecord;

    render(
      <TimelineItem
        record={record}
        onNavigateEdit={noop}
        onDeleted={noop}
      />,
    );

    expect(screen.getByText("Next Period")).toBeInTheDocument();
    expect(screen.getByText(/Confidence: High/)).toBeInTheDocument();
  });

  it("renders InsightCard for insight records", () => {
    const record = {
      record: {
        case: "insight" as const,
        value: {
          name: "users/default/insights/01",
          userId: "users/default",
          kind: InsightType.CYCLE_LENGTH_PATTERN,
          summary: "Your cycle length has been gradually increasing",
          confidence: ConfidenceLevel.HIGH,
          evidenceRecordRefs: [],
        },
      },
    } as unknown as TimelineRecord;

    render(
      <TimelineItem
        record={record}
        onNavigateEdit={noop}
        onDeleted={noop}
      />,
    );

    expect(screen.getByText("Cycle Length Trend")).toBeInTheDocument();
    expect(screen.getByText("Your cycle length has been gradually increasing")).toBeInTheDocument();
  });

  it("renders null for unknown record case", () => {
    const record = {
      record: { case: undefined, value: undefined },
    } as unknown as TimelineRecord;

    const { container } = render(
      <TimelineItem
        record={record}
        onNavigateEdit={noop}
        onDeleted={noop}
      />,
    );

    expect(container.firstChild).toBeNull();
  });
});
