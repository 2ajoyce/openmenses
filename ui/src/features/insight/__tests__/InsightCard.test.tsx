import { render, screen, fireEvent } from "@testing-library/react";
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

  it("displays evidence count badge and collapsed list", () => {
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

    const toggle = screen.getByRole("button", { name: /based on 3 records/i });
    expect(toggle).toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });

  it("expands to show records in lookup and note for hidden records", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [
        create(RecordRefSchema, { name: "users/default/cycles/01" }),
        create(RecordRefSchema, { name: "users/default/cycles/02" }),
      ],
    });

    // Only one ref is in the lookup — the other is outside the timeline range
    const recordLookup = {
      "users/default/cycles/01": {
        record: { case: "cycle", value: { startDate: { value: "2026-01-01" }, endDate: { value: "2026-01-28" } } },
      },
    } as never;

    render(<InsightCard insight={insight} recordLookup={recordLookup} />);

    const toggle = screen.getByRole("button", { name: /based on 2 records/i });
    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getAllByRole("listitem")).toHaveLength(1);
    expect(screen.getByText(/1 record is outside the current timeline range/i)).toBeInTheDocument();
  });

  it("expands to show only a note when no refs are in the lookup", () => {
    const insight = create(InsightSchema, {
      name: "users/default/insights/01",
      userId: "users/default",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Test summary",
      confidence: ConfidenceLevel.HIGH,
      evidenceRecordRefs: [
        create(RecordRefSchema, { name: "users/default/cycles/01" }),
        create(RecordRefSchema, { name: "users/default/cycles/02" }),
      ],
    });

    render(<InsightCard insight={insight} />);

    const toggle = screen.getByRole("button", { name: /based on 2 records/i });
    fireEvent.click(toggle);

    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    expect(screen.getByText(/2 records are outside the current timeline range/i)).toBeInTheDocument();
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

    expect(screen.queryByRole("button", { name: /based on/i })).not.toBeInTheDocument();
  });
});
