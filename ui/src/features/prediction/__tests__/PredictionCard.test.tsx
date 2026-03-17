import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { PredictionCard } from "../PredictionCard";
import {
  PredictionType,
  ConfidenceLevel,
  PredictionSchema,
} from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";

vi.mock("framework7-react", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div data-testid="card">{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="card-header">{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div data-testid="card-content">{children}</div>,
}));

describe("PredictionCard", () => {
  it("renders NEXT_BLEED prediction type label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/01",
      userId: "users/default",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-01" }),
      predictedEndDate: create(LocalDateSchema, { value: "2026-04-06" }),
      confidence: ConfidenceLevel.HIGH,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText("Next Period")).toBeInTheDocument();
  });

  it("renders PMS_WINDOW prediction type label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/02",
      userId: "users/default",
      kind: PredictionType.PMS_WINDOW,
      predictedStartDate: create(LocalDateSchema, { value: "2026-03-22" }),
      predictedEndDate: create(LocalDateSchema, { value: "2026-03-31" }),
      confidence: ConfidenceLevel.MEDIUM,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText("PMS Window")).toBeInTheDocument();
  });

  it("renders OVULATION_WINDOW prediction type label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/03",
      userId: "users/default",
      kind: PredictionType.OVULATION_WINDOW,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-14" }),
      predictedEndDate: create(LocalDateSchema, { value: "2026-04-16" }),
      confidence: ConfidenceLevel.MEDIUM,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText("Ovulation Window")).toBeInTheDocument();
  });

  it("renders SYMPTOM_WINDOW prediction type label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/04",
      userId: "users/default",
      kind: PredictionType.SYMPTOM_WINDOW,
      predictedStartDate: create(LocalDateSchema, { value: "2026-03-25" }),
      predictedEndDate: create(LocalDateSchema, { value: "2026-03-29" }),
      confidence: ConfidenceLevel.LOW,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText("Symptom Window")).toBeInTheDocument();
  });

  it("displays date range when both start and end dates are present", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/01",
      userId: "users/default",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-01" }),
      predictedEndDate: create(LocalDateSchema, { value: "2026-04-06" }),
      confidence: ConfidenceLevel.HIGH,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText(/Apr 1, 2026 – Apr 6, 2026/)).toBeInTheDocument();
  });

  it("displays only start date when end date is absent", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/01",
      userId: "users/default",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-01" }),
      confidence: ConfidenceLevel.HIGH,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText(/Apr 1, 2026/)).toBeInTheDocument();
    expect(screen.queryByText(/–/)).not.toBeInTheDocument();
  });

  it("displays confidence badge with label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/01",
      userId: "users/default",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-01" }),
      predictedEndDate: create(LocalDateSchema, { value: "2026-04-06" }),
      confidence: ConfidenceLevel.HIGH,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText(/Confidence: High/)).toBeInTheDocument();
  });

  it("displays medium confidence label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/02",
      userId: "users/default",
      kind: PredictionType.PMS_WINDOW,
      predictedStartDate: create(LocalDateSchema, { value: "2026-03-22" }),
      confidence: ConfidenceLevel.MEDIUM,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText(/Confidence: Medium/)).toBeInTheDocument();
  });

  it("displays low confidence label", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/04",
      userId: "users/default",
      kind: PredictionType.SYMPTOM_WINDOW,
      predictedStartDate: create(LocalDateSchema, { value: "2026-03-25" }),
      confidence: ConfidenceLevel.LOW,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText(/Confidence: Low/)).toBeInTheDocument();
  });

  it("renders rationale list when rationale items are present", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/01",
      userId: "users/default",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-01" }),
      confidence: ConfidenceLevel.HIGH,
      rationale: ["Based on 3 completed cycles", "Average cycle length: 28 days"],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.getByText("Based on 3 completed cycles")).toBeInTheDocument();
    expect(screen.getByText("Average cycle length: 28 days")).toBeInTheDocument();
  });

  it("does not render rationale section when rationale is empty", () => {
    const prediction = create(PredictionSchema, {
      name: "users/default/predictions/01",
      userId: "users/default",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2026-04-01" }),
      confidence: ConfidenceLevel.HIGH,
      rationale: [],
    });

    render(<PredictionCard prediction={prediction} />);

    expect(screen.queryByRole("list")).not.toBeInTheDocument();
  });
});
