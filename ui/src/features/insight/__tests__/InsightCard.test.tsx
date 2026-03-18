import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { InsightCard } from "../InsightCard";
import {
  InsightType,
  ConfidenceLevel,
  InsightSchema,
} from "@gen/openmenses/v1/model_pb";
import { RecordRefSchema } from "@gen/openmenses/v1/types_pb";

vi.mock("framework7-react", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div data-testid="card">{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="card-header">{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div data-testid="card-content">{children}</div>,
}));

describe("InsightCard", () => {
  it("renders CYCLE_LENGTH_PATTERN insight type label", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Your cycle length has been gradually increasing",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText("Cycle Length Trend")).toBeInTheDocument();
  });

  it("renders SYMPTOM_PATTERN insight type label", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/02",
      userId: "users/default",
      kind: InsightType.SYMPTOM_PATTERN,
      summary: "Headaches tend to occur around cycle day 12",
      confidence: ConfidenceLevel.MEDIUM,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText("Symptom Pattern")).toBeInTheDocument();
  });

  it("renders MEDICATION_ADHERENCE_PATTERN insight type label", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/03",
      userId: "users/default",
      kind: InsightType.MEDICATION_ADHERENCE_PATTERN,
      summary: "Your adherence to Ibuprofen has been high at 95%",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText("Medication Adherence")).toBeInTheDocument();
  });

  it("renders BLEEDING_PATTERN insight type label", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/04",
      userId: "users/default",
      kind: InsightType.BLEEDING_PATTERN,
      summary: "Your period duration has been stable at around 5 days",
      confidence: ConfidenceLevel.MEDIUM,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText("Bleeding Pattern")).toBeInTheDocument();
  });

  it("displays summary text", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Your cycle length has been gradually increasing over the last 5 cycles",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText("Your cycle length has been gradually increasing over the last 5 cycles")).toBeInTheDocument();
  });

  it("displays confidence badge with HIGH level", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText(/Confidence: High/)).toBeInTheDocument();
  });

  it("displays confidence badge with MEDIUM level", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/02",
      userId: "users/default",
      kind: InsightType.SYMPTOM_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.MEDIUM,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText(/Confidence: Medium/)).toBeInTheDocument();
  });

  it("displays confidence badge with LOW level", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/03",
      userId: "users/default",
      kind: InsightType.MEDICATION_ADHERENCE_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.LOW,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText(/Confidence: Low/)).toBeInTheDocument();
  });

  it("displays evidence record references", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [
        create(RecordRefSchema, { name: "users/default/cycles/01" }),
        create(RecordRefSchema, { name: "users/default/cycles/02" }),
        create(RecordRefSchema, { name: "users/default/cycles/03" }),
      ],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.getByText("Evidence:")).toBeInTheDocument();
    expect(screen.getByText("users/default/cycles/01")).toBeInTheDocument();
    expect(screen.getByText("users/default/cycles/02")).toBeInTheDocument();
    expect(screen.getByText("users/default/cycles/03")).toBeInTheDocument();
  });

  it("does not render evidence section when no evidence references are present", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [],
    });

    render(<InsightCard insight={insight} />);

    expect(screen.queryByText("Evidence:")).not.toBeInTheDocument();
  });
});
