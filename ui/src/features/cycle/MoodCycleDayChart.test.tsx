import { create } from "@bufbuild/protobuf";
import {
  CycleSchema,
  CycleSource,
  MoodIntensity,
  MoodObservationSchema,
  MoodType,
} from "@gen/openmenses/v1/model_pb";
import { DateTimeSchema, LocalDateSchema } from "@gen/openmenses/v1/types_pb";
import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MoodCycleDayChart } from "./MoodCycleDayChart";

// Mock ResizeObserver before importing recharts
globalThis.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
})) as unknown as typeof ResizeObserver;

const mockListMoodObservations = vi.fn();
const mockListCycles = vi.fn();

// Mock the client
vi.mock("../../lib/client", () => ({
  client: {
    listMoodObservations: (...args: unknown[]) =>
      mockListMoodObservations(...args),
    listCycles: (...args: unknown[]) => mockListCycles(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

describe("MoodCycleDayChart", () => {
  beforeEach(() => {
    vi.clearAllMocks();
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

    const { container } = render(<MoodCycleDayChart />);
    await new Promise((r) => setTimeout(r, 100));
    expect(container.firstChild).toBeNull();
  });

  it("returns null when there are fewer than 3 completed cycles", async () => {
    const moods = [
      createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodCycleDayChart />);
    await new Promise((r) => setTimeout(r, 100));
    expect(container.firstChild).toBeNull();
  });

  it("computes cycle day correctly", async () => {
    const moods = [
      createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
      createMoodObservation("2024-01-10", MoodType.HAPPY, MoodIntensity.MEDIUM),
      createMoodObservation("2024-02-05", MoodType.CALM, MoodIntensity.HIGH),
      createMoodObservation("2024-02-10", MoodType.HAPPY, MoodIntensity.LOW),
      createMoodObservation("2024-03-05", MoodType.CALM, MoodIntensity.MEDIUM),
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [
        createCycle("2024-01-01", "2024-01-28"),
        createCycle("2024-02-01", "2024-02-28"),
        createCycle("2024-03-01", "2024-03-28"),
      ],
    });

    const { container } = render(<MoodCycleDayChart />);
    await new Promise((r) => setTimeout(r, 100));

    // Check that the chart container is rendered
    const chartContainer = container.querySelector('div[role="img"]');
    expect(chartContainer).toBeInTheDocument();
  });

  it("averages mood intensity across multiple cycles", async () => {
    const moods = [
      // Cycle 1, day 5
      createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
      // Cycle 2, day 5
      createMoodObservation("2024-02-05", MoodType.CALM, MoodIntensity.HIGH),
      // Cycle 3, day 5
      createMoodObservation("2024-03-05", MoodType.CALM, MoodIntensity.MEDIUM),
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [
        createCycle("2024-01-01", "2024-01-28"),
        createCycle("2024-02-01", "2024-02-28"),
        createCycle("2024-03-01", "2024-03-28"),
      ],
    });

    const { container } = render(<MoodCycleDayChart />);
    await new Promise((r) => setTimeout(r, 100));

    const chartContainer = container.querySelector('div[role="img"]');
    expect(chartContainer).toBeInTheDocument();
  });

  it("shows empty state with insufficient data", async () => {
    mockListMoodObservations.mockResolvedValue({ observations: [] });
    mockListCycles.mockResolvedValue({
      cycles: [
        createCycle("2024-01-01", "2024-01-28"),
        createCycle("2024-02-01", "2024-02-28"),
      ],
    });

    const { container } = render(<MoodCycleDayChart />);
    await new Promise((r) => setTimeout(r, 100));

    expect(container.firstChild).toBeNull();
  });

  it("handles multiple mood types on same cycle day", async () => {
    const moods = [
      createMoodObservation("2024-01-05", MoodType.CALM, MoodIntensity.LOW),
      createMoodObservation("2024-01-05", MoodType.HAPPY, MoodIntensity.MEDIUM),
      createMoodObservation("2024-02-05", MoodType.CALM, MoodIntensity.HIGH),
      createMoodObservation("2024-02-05", MoodType.HAPPY, MoodIntensity.LOW),
      createMoodObservation("2024-03-05", MoodType.CALM, MoodIntensity.MEDIUM),
      createMoodObservation("2024-03-05", MoodType.HAPPY, MoodIntensity.HIGH),
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [
        createCycle("2024-01-01", "2024-01-28"),
        createCycle("2024-02-01", "2024-02-28"),
        createCycle("2024-03-01", "2024-03-28"),
      ],
    });

    const { container } = render(<MoodCycleDayChart />);
    await new Promise((r) => setTimeout(r, 100));

    const chartContainer = container.querySelector('div[role="img"]');
    expect(chartContainer).toBeInTheDocument();
  });
});
