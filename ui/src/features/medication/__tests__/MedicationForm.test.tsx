import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import React from "react";
import MedicationForm from "../MedicationForm";

const mockBack = vi.fn();
const mockRouter = { back: mockBack } as never;

const mockCreateMedication = vi.fn();
const mockGetMedication = vi.fn();
const mockUpdateMedication = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    createMedication: (...args: unknown[]) => mockCreateMedication(...args),
    getMedication: (...args: unknown[]) => mockGetMedication(...args),
    updateMedication: (...args: unknown[]) => mockUpdateMedication(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

describe("MedicationForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders form fields for create mode", () => {
    render(<MedicationForm f7router={mockRouter} />);
    expect(screen.getByText("Add Medication")).toBeInTheDocument();
    expect(screen.getByText("Category")).toBeInTheDocument();
  });

  it("renders edit mode title when medicationName provided", async () => {
    mockGetMedication.mockResolvedValue({
      medication: {
        name: "med-1",
        displayName: "Aspirin",
        category: 2,
        note: "",
        active: true,
      },
    });

    render(
      <MedicationForm f7router={mockRouter} medicationName="med-1" />,
    );
    expect(screen.getByText("Edit Medication")).toBeInTheDocument();

    await waitFor(() => {
      expect(mockGetMedication).toHaveBeenCalledWith({ name: "med-1" });
    });
  });

  it("does not submit with empty name", () => {
    render(<MedicationForm f7router={mockRouter} />);
    const button = screen.getByText("Add");
    expect(button).toBeDisabled();
  });

  it("submits create request with entered name", async () => {
    mockCreateMedication.mockResolvedValue({});
    render(<MedicationForm f7router={mockRouter} />);

    const input = screen.getByPlaceholderText("Medication name");
    fireEvent.change(input, { target: { value: "Ibuprofen" } });

    fireEvent.click(screen.getByText("Add"));

    await waitFor(() => {
      expect(mockCreateMedication).toHaveBeenCalledTimes(1);
    });

    const call = mockCreateMedication.mock.calls[0]![0];
    expect(call.parent).toBe("users/default");
    expect(mockBack).toHaveBeenCalled();
  });

  it("shows error dialog on submit failure", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockCreateMedication.mockRejectedValue(new Error("Server error"));

    render(<MedicationForm f7router={mockRouter} />);

    const input = screen.getByPlaceholderText("Medication name");
    fireEvent.change(input, { target: { value: "Ibuprofen" } });

    fireEvent.click(screen.getByText("Add"));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Server error", "Error");
    });
  });

  it("submits edit request with userId set to DEFAULT_PARENT", async () => {
    mockGetMedication.mockResolvedValue({
      medication: {
        name: "med-1",
        displayName: "Aspirin",
        category: 2,
        note: "",
        active: true,
      },
    });
    mockUpdateMedication.mockResolvedValue({});

    render(
      <MedicationForm f7router={mockRouter} medicationName="med-1" />,
    );

    await waitFor(() => {
      expect(mockGetMedication).toHaveBeenCalledWith({ name: "med-1" });
    });

    fireEvent.click(screen.getByText("Update"));

    await waitFor(() => {
      expect(mockUpdateMedication).toHaveBeenCalledTimes(1);
    });

    const call = mockUpdateMedication.mock.calls[0]![0];
    expect(call.medication.userId).toBe("users/default");
    expect(call.medication.name).toBe("med-1");
    expect(mockBack).toHaveBeenCalled();
  });
});
