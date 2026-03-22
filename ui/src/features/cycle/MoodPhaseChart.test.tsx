import { create } from "@bufbuild/protobuf";
import {
  CycleSchema,
  CycleSource,
  MoodObservationSchema,
  MoodType,
} from "@gen/openmenses/v1/model_pb";
import { DateTimeSchema, LocalDateSchema } from "@gen/openmenses/v1/types_pb";
import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MoodPhaseChart } from "./MoodPhaseChart";

// Mock ResizeObserver before importing recharts
globalThis.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
})) as unknown as typeof ResizeObserver;

const mockListMoodObservations = vi.fn();
const mockListCycles = vi.fn();
const mockListTimeline = vi.fn();

// Mock the client
vi.mock("../../lib/client", () => ({
  client: {
    listMoodObservations: (...args: unknown[]) =>
      mockListMoodObservations(...args),
    listCycles: (...args: unknown[]) => mockListCycles(...args),
    listTimeline: (...args: unknown[]) => mockListTimeline(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

describe("MoodPhaseChart", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default: no phase estimates available (charts fall back to arithmetic)
    mockListTimeline.mockResolvedValue({ records: [] });
  });

  const createMoodObservation = (observedAtDate: string, moodType: MoodType) =>
    create(MoodObservationSchema, {
      name: "observations/test",
      timestamp: create(DateTimeSchema, {
        value: `${observedAtDate}T12:00:00Z`,
      }),
      mood: moodType,
    });

  const createCycle = (startDate: string, endDate: string) =>
    create(CycleSchema, {
      name: "cycles/test",
      startDate: create(LocalDateSchema, { value: startDate }),
      endDate: create(LocalDateSchema, { value: endDate }),
      source: CycleSource.DERIVED_FROM_BLEEDING,
    });

  it("returns null when no mood observations exist", async () => {
    mockListMoodObservations.mockResolvedValue({ observations: [] });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseChart />);
    // Allow async operations to complete
    await new Promise((r) => setTimeout(r, 100));
    expect(container.firstChild).toBeNull();
  });

  it("returns null when no cycles exist", async () => {
    mockListMoodObservations.mockResolvedValue({
      observations: [createMoodObservation("2024-01-10", MoodType.HAPPY)],
    });
    mockListCycles.mockResolvedValue({ cycles: [] });

    const { container } = render(<MoodPhaseChart />);
    await new Promise((r) => setTimeout(r, 100));
    expect(container.firstChild).toBeNull();
  });

  it("buckets moods correctly into cycle phases", async () => {
    const moods = [
      createMoodObservation("2024-01-02", MoodType.HAPPY), // Menstruation (day 2)
      createMoodObservation("2024-01-08", MoodType.ANXIOUS), // Follicular (day 8)
      createMoodObservation("2024-01-14", MoodType.SAD), // Ovulation (day 14)
      createMoodObservation("2024-01-20", MoodType.IRRITABLE), // Luteal (day 20)
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [createCycle("2024-01-01", "2024-01-28")],
    });

    const { container } = render(<MoodPhaseChart />);
    await new Promise((r) => setTimeout(r, 100));

    // Chart should be rendered
    const img = container.querySelector('[role="img"]');
    expect(img).toBeTruthy();
  });

  it("handles multiple moods across multiple cycles", async () => {
    const moods = [
      createMoodObservation("2024-01-02", MoodType.HAPPY),
      createMoodObservation("2024-01-02", MoodType.CALM),
      createMoodObservation("2024-02-02", MoodType.ANXIOUS),
      createMoodObservation("2024-02-08", MoodType.SAD),
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [
        createCycle("2024-01-01", "2024-01-28"),
        createCycle("2024-01-28", "2024-02-25"),
      ],
    });

    const { container } = render(<MoodPhaseChart />);
    await new Promise((r) => setTimeout(r, 100));

    const img = container.querySelector('[role="img"]');
    expect(img).toBeTruthy();
  });

  it("ignores moods outside cycle date ranges", async () => {
    const moods = [
      createMoodObservation("2024-01-02", MoodType.HAPPY),
      createMoodObservation("2024-03-15", MoodType.SAD), // Outside any cycle
    ];

    mockListMoodObservations.mockResolvedValue({ observations: moods });
    mockListCycles.mockResolvedValue({
      cycles: [
        createCycle("2024-01-01", "2024-01-28"),
        createCycle("2024-01-28", "2024-02-25"),
      ],
    });

    const { container } = render(<MoodPhaseChart />);
    await new Promise((r) => setTimeout(r, 100));

    const img = container.querySelector('[role="img"]');
    expect(img).toBeTruthy();
  });
});
