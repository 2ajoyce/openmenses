import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { PhaseEstimateCard } from "../PhaseEstimateCard";
import {
  CyclePhase,
  ConfidenceLevel,
  BiologicalCycleModel,
  PhaseEstimateSchema,
} from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";

vi.mock("framework7-react", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div data-testid="card">{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="card-header">{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div data-testid="card-content">{children}</div>,
}));

describe("PhaseEstimateCard", () => {
  it("returns null when estimates array is empty", () => {
    const { container } = render(<PhaseEstimateCard estimates={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders menstruation phase with correct label", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
    ];

    render(<PhaseEstimateCard estimates={estimates} />);

    expect(screen.getByText("Menstruation")).toBeInTheDocument();
  });

  it("renders follicular phase with correct label when not suppressed", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-05" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
    ];

    render(
      <PhaseEstimateCard
        estimates={estimates}
        biologicalCycleModel={BiologicalCycleModel.OVULATORY}
      />
    );

    expect(screen.getByText("Follicular")).toBeInTheDocument();
  });

  it("renders follicular phase as 'Pill-free / Active pill days' when suppressed", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-05" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
    ];

    render(
      <PhaseEstimateCard
        estimates={estimates}
        biologicalCycleModel={BiologicalCycleModel.HORMONALLY_SUPPRESSED}
      />
    );

    expect(screen.getByText("Pill-free / Active pill days")).toBeInTheDocument();
  });

  it("renders ovulation window phase", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-10" }),
        phase: CyclePhase.OVULATION_WINDOW,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
    ];

    render(<PhaseEstimateCard estimates={estimates} />);

    expect(screen.getByText("Ovulation Window")).toBeInTheDocument();
  });

  it("renders luteal phase", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-15" }),
        phase: CyclePhase.LUTEAL,
        confidence: ConfidenceLevel.LOW,
        userId: "users/default",
      }),
    ];

    render(<PhaseEstimateCard estimates={estimates} />);

    expect(screen.getByText("Luteal")).toBeInTheDocument();
  });

  it("displays confidence level label", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
    ];

    render(<PhaseEstimateCard estimates={estimates} />);

    expect(screen.getByText(/Confidence: High/)).toBeInTheDocument();
  });

  it("displays date range for grouped estimates", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-2",
        date: create(LocalDateSchema, { value: "2026-03-02" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-3",
        date: create(LocalDateSchema, { value: "2026-03-03" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
    ];

    render(<PhaseEstimateCard estimates={estimates} />);

    expect(screen.getByText(/Mar 1, 2026 – Mar 3, 2026/)).toBeInTheDocument();
  });

  it("displays single date when only one estimate", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-05" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
    ];

    render(<PhaseEstimateCard estimates={estimates} />);

    expect(screen.getByText(/Mar 5, 2026/)).toBeInTheDocument();
  });
});
