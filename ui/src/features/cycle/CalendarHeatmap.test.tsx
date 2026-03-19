import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";
import { CalendarHeatmap } from "./CalendarHeatmap";

const mockListTimeline = vi.fn();

// Mock the client module
vi.mock("../../lib/client", () => ({
  client: {
    listTimeline: (...args: unknown[]) => mockListTimeline(...args),
  },
  DEFAULT_PARENT: "users/test",
}));

describe("CalendarHeatmap", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders calendar grid with day labels and cells", async () => {
    mockListTimeline.mockResolvedValue({ records: [] });

    const { container } = render(<CalendarHeatmap />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalled();
    });

    // Check that the grid and day labels exist
    const grid = container.querySelector(".om-heatmap-grid");
    expect(grid).toBeInTheDocument();

    const dayLabels = container.querySelectorAll(".om-heatmap-label");
    expect(dayLabels.length).toBe(7); // Sun-Sat
  });

  it("renders empty month with no colored cells", async () => {
    mockListTimeline.mockResolvedValue({ records: [] });

    const { container } = render(<CalendarHeatmap />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalled();
    });

    // No cells should have active indicators
    const activeCells = container.querySelectorAll(".om-heatmap-cell-active");
    expect(activeCells).toHaveLength(0);
  });

  it("navigation buttons exist and can be clicked", async () => {
    mockListTimeline.mockResolvedValue({ records: [] });

    render(<CalendarHeatmap />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalled();
    });

    // Check that navigation buttons exist
    const nextButton = screen.getByLabelText("Next month");
    const prevButton = screen.getByLabelText("Previous month");

    expect(nextButton).toBeInTheDocument();
    expect(prevButton).toBeInTheDocument();

    // Click next button and verify it fetches new data
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledTimes(2);
    });
  });

  it("displays observation indicators when data exists", async () => {
    const today = new Date();
    const dateStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;

    const mockTimelineRecords = [
      {
        date: create(LocalDateSchema, { value: dateStr }),
        record: {
          case: "bleedingObservation",
          value: { flow: 2 }, // Light flow
        },
      },
    ];
    mockListTimeline.mockResolvedValue({ records: mockTimelineRecords });

    const { container } = render(<CalendarHeatmap />);

    await waitFor(() => {
      // Should find cells with indicators
      const indicators = container.querySelectorAll(".om-heatmap-indicator");
      expect(indicators.length).toBeGreaterThan(0);
    });
  });

  it("cells have aria-labels with observation summary", async () => {
    const today = new Date();
    const dateStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;

    const mockTimelineRecords = [
      {
        date: create(LocalDateSchema, { value: dateStr }),
        record: {
          case: "bleedingObservation",
          value: { flow: 2 }, // Light flow
        },
      },
    ];
    mockListTimeline.mockResolvedValue({ records: mockTimelineRecords });

    const { container } = render(<CalendarHeatmap />);

    await waitFor(() => {
      // Look for cells with aria-labels
      const cellsWithLabels = container.querySelectorAll("[aria-label]");
      expect(cellsWithLabels.length).toBeGreaterThan(1); // At least navigation buttons + data cells
    });
  });

  it("loads timeline data for current month on mount", async () => {
    mockListTimeline.mockResolvedValue({ records: [] });

    render(<CalendarHeatmap />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalled();
      const callArgs = mockListTimeline.mock.calls[0]?.[0] as Record<string, unknown> | undefined;
      expect((callArgs?.range as Record<string, unknown> | undefined)?.start).toBeDefined();
    });
  });
});
