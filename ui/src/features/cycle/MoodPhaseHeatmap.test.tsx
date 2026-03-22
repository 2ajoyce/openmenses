import { create } from "@bufbuild/protobuf";
import {
  CycleSchema,
  CycleSource,
  MoodIntensity,
  MoodObservationSchema,
  MoodType,
} from "@gen/openmenses/v1/model_pb";
import { DateTimeSchema, LocalDateSchema } from "@gen/openmenses/v1/types_pb";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MoodPhaseHeatmap } from "./MoodPhaseHeatmap";

// ChartContainer uses Recharts ResponsiveContainer which requires ResizeObserver
globalThis.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
})) as unknown as typeof ResizeObserver;

const mockListMoodObservations = vi.fn();
const mockListCycles = vi.fn();
const mockListTimeline = vi.fn();

vi.mock("../../lib/client", () => ({
  client: {
    listMoodObservations: (...args: unknown[]) =>
      mockListMoodObservations(...args),
    listCycles: (...args: unknown[]) => mockListCycles(...args),
    listTimeline: (...args: unknown[]) => mockListTimeline(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

describe("MoodPhaseHeatmap", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default: timeline returns no phase estimates (component falls back to arithmetic)
    mockListTimeline.mockResolvedValue({ records: [], pagination: {} });
  });

  const createMoodObservation = (
    observedAtDate: string,
    moodType: MoodType,
    intensity: MoodIntensity,
  ) =>
    create(MoodObservationSchema, {
      name: "observations/test",
      timestamp: create(DateTimeSchema, {
        value: `${observedAtDate}T12:00:00Z`,
      }),
      mood: moodType,
      intensity: intensity,
    });

  const createCycle = (startDate: string, endDate: string) =>
    create(CycleSchema, {
      name: "cycles/test",
      startDate: create(LocalDateSchema, { value: startDate }),
      endDate: create(LocalDateSchema, { value: endDate }),
      source: CycleSource.DERIVED_FROM_BLEEDING,
    });

  it("returns null when there are no mood observations", async () => {
    mockListMoodObservations.mockResolvedValue({ observations: [] });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));
    expect(container.firstChild).toBeNull();
  });

  it("returns null when there are no cycles", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
      ],
    });
    mockListCycles.mockResolvedValue({ cycles: [] });

    const { container } = render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));
    expect(container.firstChild).toBeNull();
  });

  it("renders the heatmap grid when data is available", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
        createMoodObservation(
          "2024-01-10",
          MoodType.HAPPY,
          MoodIntensity.MEDIUM,
        ),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    expect(container.querySelector('div[role="img"]')).toBeInTheDocument();
    expect(container.querySelector(".om-mood-heatmap")).toBeInTheDocument();
  });

  it("shows phase column headers", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    expect(screen.getByText("Mens.")).toBeInTheDocument();
    expect(screen.getByText("Foll.")).toBeInTheDocument();
    expect(screen.getByText("Ovul.")).toBeInTheDocument();
    expect(screen.getByText("Luteal")).toBeInTheDocument();
  });

  it("shows mood type row labels for observed moods only", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
        createMoodObservation(
          "2024-01-10",
          MoodType.IRRITABLE,
          MoodIntensity.HIGH,
        ),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    expect(screen.getByText("Calm")).toBeInTheDocument();
    expect(screen.getByText("Irritable")).toBeInTheDocument();
    expect(screen.queryByText("Happy")).not.toBeInTheDocument();
  });

  it("shows Low label for low intensity observations", async () => {
    // Day 2 falls in menstruation phase (day 0-based < 5)
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.LOW),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    const lowElements = screen.getAllByText("Low");
    expect(lowElements.length).toBeGreaterThanOrEqual(1);
  });

  it("shows High label for high intensity observations", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.HIGH),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    const highElements = screen.getAllByText("High");
    expect(highElements.length).toBeGreaterThanOrEqual(1);
  });

  it("shows empty cells for mood/phase combos with no observations", async () => {
    // One observation on day 2 (menstruation). The other 3 phases have no data → empty cells.
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.LOW),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    const emptyCells = container.querySelectorAll(
      ".om-mood-heatmap-cell--empty",
    );
    expect(emptyCells.length).toBe(3);
  });

  it("averages intensity correctly across multiple observations in same phase", async () => {
    // Two LOW (1) observations in menstruation phase → avg 1.0 → "Low"
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.LOW),
        createMoodObservation("2024-01-03", MoodType.CALM, MoodIntensity.LOW),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    const lowElements = screen.getAllByText("Low");
    // Legend always shows "Low"; cell should also show "Low" → at least 2
    expect(lowElements.length).toBeGreaterThanOrEqual(2);
  });

  it("shows Med when averaged intensity is between 1.5 and 2.5", async () => {
    // LOW (1) + HIGH (3) = avg 2.0 → "Med"
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.LOW),
        createMoodObservation("2024-01-03", MoodType.CALM, MoodIntensity.HIGH),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    const medElements = screen.getAllByText("Med");
    // Legend always shows "Med"; cell should also show "Med" → at least 2
    expect(medElements.length).toBeGreaterThanOrEqual(2);
  });

  it("renders the intensity legend", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.LOW),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    expect(
      container.querySelector(".om-mood-heatmap-intensity-legend"),
    ).toBeInTheDocument();
    expect(screen.getByText("Intensity:")).toBeInTheDocument();
  });

  it("handles multiple mood types on the same cycle day", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        createMoodObservation("2024-01-02", MoodType.CALM, MoodIntensity.LOW),
        createMoodObservation(
          "2024-01-02",
          MoodType.HAPPY,
          MoodIntensity.MEDIUM,
        ),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    expect(screen.getByText("Calm")).toBeInTheDocument();
    expect(screen.getByText("Happy")).toBeInTheDocument();
  });

  it("ignores mood observations with UNSPECIFIED intensity", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [
        create(MoodObservationSchema, {
          name: "observations/test",
          timestamp: create(DateTimeSchema, { value: "2024-01-02T12:00:00Z" }),
          mood: MoodType.CALM,
          intensity: MoodIntensity.UNSPECIFIED,
        }),
      ],
    });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseHeatmap />);
    await new Promise((r) => setTimeout(r, 100));

    expect(container.firstChild).toBeNull();
  });
});
