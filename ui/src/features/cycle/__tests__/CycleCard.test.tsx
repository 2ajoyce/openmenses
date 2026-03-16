import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { CycleCard } from "../CycleCard";
import { CycleSource, CycleSchema } from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";

vi.mock("framework7-react", () => ({
  Card: ({ children }: { children: React.ReactNode }) => <div data-testid="card">{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="card-header">{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div data-testid="card-content">{children}</div>,
}));

describe("CycleCard", () => {
  it("renders complete cycle with start and end dates", () => {
    const cycle = create(CycleSchema, {
      name: "cycles/1",
      startDate: create(LocalDateSchema, { value: "2026-03-01" }),
      endDate: create(LocalDateSchema, { value: "2026-03-28" }),
      source: CycleSource.DERIVED_FROM_BLEEDING,
      userId: "users/default",
    });

    render(<CycleCard cycle={cycle} />);

    expect(screen.getByText(/Cycle — Derived from bleeding/)).toBeInTheDocument();
    expect(screen.getByText(/Start: Mar 1, 2026/)).toBeInTheDocument();
    expect(screen.getByText(/End: Mar 28, 2026/)).toBeInTheDocument();
    expect(screen.getByText(/Length:/)).toBeInTheDocument();
  });

  it("renders open-ended cycle without end date", () => {
    const cycle = create(CycleSchema, {
      name: "cycles/2",
      startDate: create(LocalDateSchema, { value: "2026-03-01" }),
      source: CycleSource.DERIVED_FROM_BLEEDING,
      userId: "users/default",
    });

    render(<CycleCard cycle={cycle} />);

    expect(screen.getByText(/Current Cycle/)).toBeInTheDocument();
    expect(screen.getByText(/In progress/)).toBeInTheDocument();
    expect(screen.queryByText(/Length:/)).not.toBeInTheDocument();
  });

  it("renders cycle with user-confirmed source", () => {
    const cycle = create(CycleSchema, {
      name: "cycles/3",
      startDate: create(LocalDateSchema, { value: "2026-02-01" }),
      endDate: create(LocalDateSchema, { value: "2026-02-29" }),
      source: CycleSource.USER_CONFIRMED,
      userId: "users/default",
    });

    render(<CycleCard cycle={cycle} />);

    expect(screen.getByText(/User confirmed/)).toBeInTheDocument();
  });

  it("displays cycle length when both dates are present", () => {
    const cycle = create(CycleSchema, {
      name: "cycles/4",
      startDate: create(LocalDateSchema, { value: "2026-03-01" }),
      endDate: create(LocalDateSchema, { value: "2026-03-28" }),
      source: CycleSource.DERIVED_FROM_BLEEDING,
      userId: "users/default",
    });

    render(<CycleCard cycle={cycle} />);

    expect(screen.getByText(/Length: \d+ days/)).toBeInTheDocument();
  });

  it("handles cycle with no start date gracefully", () => {
    const cycle = create(CycleSchema, {
      name: "cycles/5",
      endDate: create(LocalDateSchema, { value: "2026-03-28" }),
      source: CycleSource.DERIVED_FROM_BLEEDING,
      userId: "users/default",
    });

    render(<CycleCard cycle={cycle} />);

    expect(screen.queryByText(/Start:/)).not.toBeInTheDocument();
    expect(screen.getByText(/End:/)).toBeInTheDocument();
  });
});
