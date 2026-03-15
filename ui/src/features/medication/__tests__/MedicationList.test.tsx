import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import React from "react";
import MedicationList from "../MedicationList";

const mockNavigate = vi.fn();
const mockRouter = { navigate: mockNavigate } as never;

const mockListMedications = vi.fn();
const mockDeleteMedication = vi.fn();
const mockUpdateMedication = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    listMedications: (...args: unknown[]) => mockListMedications(...args),
    deleteMedication: (...args: unknown[]) => mockDeleteMedication(...args),
    updateMedication: (...args: unknown[]) => mockUpdateMedication(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

const medications = [
  {
    name: "med-1",
    displayName: "Yasmin",
    category: 1,
    active: true,
    note: "",
  },
  {
    name: "med-2",
    displayName: "Advil",
    category: 2,
    active: false,
    note: "",
  },
];

describe("MedicationList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders medication list from listMedications response", async () => {
    mockListMedications.mockResolvedValue({ medications });

    render(<MedicationList f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Yasmin")).toBeInTheDocument();
      expect(screen.getByText("Advil")).toBeInTheDocument();
    });
  });

  it("shows EmptyState when no medications exist", async () => {
    mockListMedications.mockResolvedValue({ medications: [] });

    render(<MedicationList f7router={mockRouter} />);

    await waitFor(() => {
      expect(
        screen.getByText("No medications added yet"),
      ).toBeInTheDocument();
    });
  });

  it("calls deleteMedication then re-fetches list on success", async () => {
    mockListMedications.mockResolvedValue({ medications });
    mockDeleteMedication.mockResolvedValue({});

    render(<MedicationList f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Yasmin")).toBeInTheDocument();
    });

    // Click the Delete button (rendered by SwipeoutButton mock)
    const deleteButtons = screen.getAllByText("Delete");
    fireEvent.click(deleteButtons[0]!);

    await waitFor(() => {
      expect(mockDeleteMedication).toHaveBeenCalledWith({ name: "med-1" });
    });

    // Should re-fetch after successful delete
    await waitFor(() => {
      expect(mockListMedications).toHaveBeenCalledTimes(2);
    });
  });

  it("re-fetches list to restore state when deleteMedication fails", async () => {
    mockListMedications.mockResolvedValue({ medications });
    mockDeleteMedication.mockRejectedValue(new Error("Delete failed"));

    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");

    render(<MedicationList f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Yasmin")).toBeInTheDocument();
    });

    const deleteButtons = screen.getAllByText("Delete");
    fireEvent.click(deleteButtons[0]!);

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Delete failed", "Error");
    });

    // Should re-fetch to restore list after failed delete
    await waitFor(() => {
      expect(mockListMedications).toHaveBeenCalledTimes(2);
    });
  });

  it("calls updateMedication with active: false when Deactivate is clicked", async () => {
    mockListMedications.mockResolvedValue({ medications });
    mockUpdateMedication.mockResolvedValue({});

    render(<MedicationList f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Yasmin")).toBeInTheDocument();
    });

    // Click the Deactivate button for the active medication
    const deactivateButton = screen.getByText("Deactivate");
    fireEvent.click(deactivateButton);

    await waitFor(() => {
      expect(mockUpdateMedication).toHaveBeenCalledWith(
        expect.objectContaining({
          medication: expect.objectContaining({ active: false }),
          updateMask: { paths: ["active"] },
        }),
      );
    });
  });
});
