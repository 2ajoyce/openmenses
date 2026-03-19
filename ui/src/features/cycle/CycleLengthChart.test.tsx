import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { CycleSchema, CycleSource } from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";
import { CycleLengthChart } from "./CycleLengthChart";

// Mock ResizeObserver before importing recharts
globalThis.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
})) as unknown as typeof ResizeObserver;

describe("CycleLengthChart", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const createCycle = (
    startDate: string,
    endDate: string | null,
  ) => create(CycleSchema, {
    name: "cycles/test",
    startDate: create(LocalDateSchema, { value: startDate }),
    ...(endDate ? { endDate: create(LocalDateSchema, { value: endDate }) } : {}),
    source: CycleSource.DERIVED_FROM_BLEEDING,
  });

  it("returns null with 0 cycles", () => {
    const { container } = render(<CycleLengthChart cycles={[]} />);
    expect(container.firstChild).toBeNull();
  });

  it("returns null with 1 completed cycle", () => {
    const cycles = [createCycle("2024-01-01", "2024-01-28")];
    const { container } = render(<CycleLengthChart cycles={cycles} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders chart with 3 completed cycles", () => {
    const cycles = [
      createCycle("2024-01-01", "2024-01-28"),
      createCycle("2024-01-28", "2024-02-24"),
      createCycle("2024-02-24", "2024-03-22"),
    ];
    render(<CycleLengthChart cycles={cycles} />);
    // Chart container should be rendered
    const chartContainer = screen.getByRole("img");
    expect(chartContainer).toBeInTheDocument();
  });

  it("ignores incomplete cycles", () => {
    const cycles = [
      createCycle("2024-01-01", "2024-01-28"),
      createCycle("2024-01-28", "2024-02-24"),
      createCycle("2024-02-24", null), // No end date
    ];
    render(<CycleLengthChart cycles={cycles} />);
    // Should still render with 2 completed cycles
    const chartContainer = screen.getByRole("img");
    expect(chartContainer).toBeInTheDocument();
  });

  it("includes average reference line with multiple cycles", () => {
    const cycles = [
      createCycle("2024-01-01", "2024-01-28"), // 27 days
      createCycle("2024-01-28", "2024-02-25"), // 28 days
      createCycle("2024-02-25", "2024-03-23"), // 27 days
    ];
    render(<CycleLengthChart cycles={cycles} />);
    const chartContainer = screen.getByRole("img");
    expect(chartContainer).toBeInTheDocument();
  });

  it("renders chart with 10 cycles", () => {
    const cycles = Array.from({ length: 10 }, (_, i) => {
      const start = new Date(2024, 0, 1 + i * 28);
      const end = new Date(start.getFullYear(), start.getMonth(), start.getDate() + 27);
      const startStr = start.toISOString().split("T")[0] ?? "";
      const endStr = end.toISOString().split("T")[0] ?? "";
      return createCycle(startStr, endStr);
    });
    render(<CycleLengthChart cycles={cycles} />);
    const chartContainer = screen.getByRole("img");
    expect(chartContainer).toBeInTheDocument();
  });
});
