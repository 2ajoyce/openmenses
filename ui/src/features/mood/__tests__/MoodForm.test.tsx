import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import React from "react";
import MoodForm from "../MoodForm";

const mockBack = vi.fn();
const mockRouter = { back: mockBack } as never;

const mockCreateMoodObservation = vi.fn();
const mockGetMoodObservation = vi.fn();
const mockUpdateMoodObservation = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    createMoodObservation: (...args: unknown[]) =>
      mockCreateMoodObservation(...args),
    getMoodObservation: (...args: unknown[]) =>
      mockGetMoodObservation(...args),
    updateMoodObservation: (...args: unknown[]) =>
      mockUpdateMoodObservation(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

describe("MoodForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders form fields", () => {
    render(<MoodForm f7router={mockRouter} />);
    expect(screen.getByText("Log Mood")).toBeInTheDocument();
    expect(screen.getByText("Mood")).toBeInTheDocument();
    expect(screen.getByText("Intensity")).toBeInTheDocument();
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("submits create request", async () => {
    mockCreateMoodObservation.mockResolvedValue({});
    render(<MoodForm f7router={mockRouter} />);

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(mockCreateMoodObservation).toHaveBeenCalledTimes(1);
    });

    const call = mockCreateMoodObservation.mock.calls[0]![0];
    expect(call.parent).toBe("users/default");
    expect(mockBack).toHaveBeenCalled();
  });

  it("shows error dialog on submit failure", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockCreateMoodObservation.mockRejectedValue(new Error("Server error"));

    render(<MoodForm f7router={mockRouter} />);
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Server error", "Error");
    });
  });

  it("submits edit request with userId set to DEFAULT_PARENT", async () => {
    mockGetMoodObservation.mockResolvedValue({
      observation: {
        name: "obs-1",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        mood: 1,
        intensity: 2,
        note: "test",
        userId: "users/default",
      },
    });
    mockUpdateMoodObservation.mockResolvedValue({});

    render(<MoodForm f7router={mockRouter} name="obs-1" />);

    await waitFor(() => {
      expect(mockGetMoodObservation).toHaveBeenCalledWith({
        name: "obs-1",
      });
    });

    fireEvent.click(screen.getByText("Update"));

    await waitFor(() => {
      expect(mockUpdateMoodObservation).toHaveBeenCalledTimes(1);
    });

    const call = mockUpdateMoodObservation.mock.calls[0]![0];
    expect(call.observation.userId).toBe("users/default");
    expect(call.observation.name).toBe("obs-1");
    expect(mockBack).toHaveBeenCalled();
  });
});
