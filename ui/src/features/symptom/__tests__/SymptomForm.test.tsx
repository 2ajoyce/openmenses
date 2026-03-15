import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import React from "react";
import SymptomForm from "../SymptomForm";

const mockBack = vi.fn();
const mockRouter = { back: mockBack } as never;

const mockCreateSymptomObservation = vi.fn();
const mockGetSymptomObservation = vi.fn();
const mockUpdateSymptomObservation = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    createSymptomObservation: (...args: unknown[]) =>
      mockCreateSymptomObservation(...args),
    getSymptomObservation: (...args: unknown[]) =>
      mockGetSymptomObservation(...args),
    updateSymptomObservation: (...args: unknown[]) =>
      mockUpdateSymptomObservation(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

describe("SymptomForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders form fields", () => {
    render(<SymptomForm f7router={mockRouter} />);
    expect(screen.getByText("Log Symptom")).toBeInTheDocument();
    expect(screen.getByText("Symptom")).toBeInTheDocument();
    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(screen.getByText("Save")).toBeInTheDocument();
  });

  it("submits create request", async () => {
    mockCreateSymptomObservation.mockResolvedValue({});
    render(<SymptomForm f7router={mockRouter} />);

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(mockCreateSymptomObservation).toHaveBeenCalledTimes(1);
    });

    const call = mockCreateSymptomObservation.mock.calls[0]![0];
    expect(call.parent).toBe("users/default");
    expect(mockBack).toHaveBeenCalled();
  });

  it("shows error dialog on submit failure", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockCreateSymptomObservation.mockRejectedValue(
      new Error("Server error"),
    );

    render(<SymptomForm f7router={mockRouter} />);
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Server error", "Error");
    });
  });

  it("submits edit request with userId set to DEFAULT_PARENT", async () => {
    mockGetSymptomObservation.mockResolvedValue({
      observation: {
        name: "obs-1",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        symptom: 1,
        severity: 2,
        note: "test",
        userId: "users/default",
      },
    });
    mockUpdateSymptomObservation.mockResolvedValue({});

    render(<SymptomForm f7router={mockRouter} name="obs-1" />);

    await waitFor(() => {
      expect(mockGetSymptomObservation).toHaveBeenCalledWith({
        name: "obs-1",
      });
    });

    fireEvent.click(screen.getByText("Update"));

    await waitFor(() => {
      expect(mockUpdateSymptomObservation).toHaveBeenCalledTimes(1);
    });

    const call = mockUpdateSymptomObservation.mock.calls[0]![0];
    expect(call.observation.userId).toBe("users/default");
    expect(call.observation.name).toBe("obs-1");
    expect(mockBack).toHaveBeenCalled();
  });
});
