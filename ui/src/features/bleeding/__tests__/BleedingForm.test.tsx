import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import BleedingForm from "../BleedingForm";

const mockBack = vi.fn();
const mockRouter = { back: mockBack, view: { main: false } } as never;

const mockCreateBleedingObservation = vi.fn();
const mockGetBleedingObservation = vi.fn();
const mockUpdateBleedingObservation = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    createBleedingObservation: (...args: unknown[]) =>
      mockCreateBleedingObservation(...args),
    getBleedingObservation: (...args: unknown[]) =>
      mockGetBleedingObservation(...args),
    updateBleedingObservation: (...args: unknown[]) =>
      mockUpdateBleedingObservation(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

describe("BleedingForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders form fields for create mode", () => {
    render(<BleedingForm f7router={mockRouter} />);
    expect(screen.getByText("Log Bleeding")).toBeInTheDocument();
    expect(screen.getByText("Flow")).toBeInTheDocument();
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("submits create request with form data", async () => {
    mockCreateBleedingObservation.mockResolvedValue({});
    render(<BleedingForm f7router={mockRouter} />);

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(mockCreateBleedingObservation).toHaveBeenCalledTimes(1);
    });

    const call = mockCreateBleedingObservation.mock.calls[0]![0];
    expect(call.parent).toBe("users/default");
    expect(call.observation).toBeDefined();
    expect(mockBack).toHaveBeenCalled();
  });

  it("renders edit mode when name is provided", async () => {
    mockGetBleedingObservation.mockResolvedValue({
      observation: {
        name: "obs-1",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        flow: 2,
        note: "test note",
        userId: "users/default",
      },
    });

    render(<BleedingForm f7router={mockRouter} name="obs-1" />);
    expect(screen.getByText("Edit Bleeding")).toBeInTheDocument();

    await waitFor(() => {
      expect(mockGetBleedingObservation).toHaveBeenCalledWith({
        name: "obs-1",
      });
    });
  });

  it("shows error dialog on submit failure", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockCreateBleedingObservation.mockRejectedValue(new Error("Server error"));

    render(<BleedingForm f7router={mockRouter} />);
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Server error", "Error");
    });
  });

  it("submits edit request with userId set to DEFAULT_PARENT", async () => {
    mockGetBleedingObservation.mockResolvedValue({
      observation: {
        name: "obs-1",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        flow: 2,
        note: "edited note",
        userId: "users/default",
      },
    });
    mockUpdateBleedingObservation.mockResolvedValue({});

    render(<BleedingForm f7router={mockRouter} name="obs-1" />);

    await waitFor(() => {
      expect(mockGetBleedingObservation).toHaveBeenCalledWith({
        name: "obs-1",
      });
    });

    fireEvent.click(screen.getByText("Update"));

    await waitFor(() => {
      expect(mockUpdateBleedingObservation).toHaveBeenCalledTimes(1);
    });

    const call = mockUpdateBleedingObservation.mock.calls[0]![0];
    expect(call.observation.userId).toBe("users/default");
    expect(call.observation.name).toBe("obs-1");
    expect(mockBack).toHaveBeenCalled();
  });
});
