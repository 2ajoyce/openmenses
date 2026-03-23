import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import MedicationEventForm from "../MedicationEventForm";

const mockBack = vi.fn();
const mockRouter = { back: mockBack, view: { main: false } } as never;

const mockCreateMedicationEvent = vi.fn();
const mockUpdateMedicationEvent = vi.fn();
const mockListMedications = vi.fn();
const mockGetMedicationEvent = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    createMedicationEvent: (...args: unknown[]) =>
      mockCreateMedicationEvent(...args),
    updateMedicationEvent: (...args: unknown[]) =>
      mockUpdateMedicationEvent(...args),
    listMedications: (...args: unknown[]) => mockListMedications(...args),
    getMedicationEvent: (...args: unknown[]) => mockGetMedicationEvent(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

const activeMedications = [
  {
    name: "med-1",
    displayName: "Birth Control",
    category: 1,
    active: true,
    note: "",
  },
  {
    name: "med-2",
    displayName: "Ibuprofen",
    category: 2,
    active: true,
    note: "",
  },
];

describe("MedicationEventForm", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListMedications.mockResolvedValue({
      medications: activeMedications,
    });
    mockGetMedicationEvent.mockResolvedValue({ event: null });
  });

  it("renders form fields", async () => {
    render(<MedicationEventForm f7router={mockRouter} />);
    expect(screen.getByText("Log Medication")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();

    await waitFor(() => {
      expect(mockListMedications).toHaveBeenCalled();
    });
  });

  it("fetches active medications on mount", async () => {
    render(<MedicationEventForm f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListMedications).toHaveBeenCalledWith(
        expect.objectContaining({ parent: "users/default" }),
      );
    });
  });

  it("submits with DEFAULT_PARENT as parent, not medicationId", async () => {
    mockCreateMedicationEvent.mockResolvedValue({});
    render(<MedicationEventForm f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListMedications).toHaveBeenCalled();
    });

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(mockCreateMedicationEvent).toHaveBeenCalledTimes(1);
    });

    const call = mockCreateMedicationEvent.mock.calls[0]![0];
    expect(call.parent).toBe("users/default");
    expect(mockBack).toHaveBeenCalled();
  });

  it("shows error dialog when listMedications rejects", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockListMedications.mockRejectedValue(new Error("Network error"));

    render(<MedicationEventForm f7router={mockRouter} />);

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Network error", "Error");
    });
  });

  it("shows error dialog when getMedicationEvent rejects in edit mode", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockGetMedicationEvent.mockRejectedValue(new Error("Not found"));

    render(<MedicationEventForm f7router={mockRouter} name="evt-1" />);

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Not found", "Error");
    });
  });

  it("shows error dialog when createMedicationEvent rejects", async () => {
    const { f7 } = await import("framework7-react");
    const alertSpy = vi.spyOn(f7.dialog, "alert");
    mockCreateMedicationEvent.mockRejectedValue(new Error("Server error"));

    render(<MedicationEventForm f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListMedications).toHaveBeenCalled();
    });

    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      expect(alertSpy).toHaveBeenCalledWith("Server error", "Error");
    });
  });

  it("submits edit request with userId set to DEFAULT_PARENT", async () => {
    mockGetMedicationEvent.mockResolvedValue({
      event: {
        name: "evt-1",
        medicationId: "med-1",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        status: 1,
        dose: "200mg",
        note: "test",
        userId: "users/default",
      },
    });
    mockUpdateMedicationEvent.mockResolvedValue({});

    render(<MedicationEventForm f7router={mockRouter} name="evt-1" />);

    await waitFor(() => {
      expect(mockGetMedicationEvent).toHaveBeenCalledWith({ name: "evt-1" });
    });

    fireEvent.click(screen.getByText("Update"));

    await waitFor(() => {
      expect(mockUpdateMedicationEvent).toHaveBeenCalledTimes(1);
    });

    const call = mockUpdateMedicationEvent.mock.calls[0]![0];
    expect(call.event.userId).toBe("users/default");
    expect(call.event.name).toBe("evt-1");
  });

  it("preserves event medication in edit mode (race condition guard)", async () => {
    mockGetMedicationEvent.mockResolvedValue({
      event: {
        name: "evt-1",
        medicationId: "med-2",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        status: 1,
        dose: "",
        note: "",
        userId: "users/default",
      },
    });

    render(<MedicationEventForm f7router={mockRouter} name="evt-1" />);

    await waitFor(() => {
      expect(mockGetMedicationEvent).toHaveBeenCalled();
      expect(mockListMedications).toHaveBeenCalled();
    });

    // The medication selector should show med-2 (the event's medication),
    // not med-1 (the first in the list)
    const select = screen.getByRole("combobox", { name: "Medication" });
    await waitFor(() => {
      expect(select).toHaveValue("med-2");
    });
  });

  it("disables medication selector in edit mode", async () => {
    mockGetMedicationEvent.mockResolvedValue({
      event: {
        name: "evt-1",
        medicationId: "med-1",
        timestamp: { value: "2026-03-01T12:00:00Z" },
        status: 1,
        dose: "",
        note: "",
        userId: "users/default",
      },
    });

    render(<MedicationEventForm f7router={mockRouter} name="evt-1" />);

    await waitFor(() => {
      expect(mockGetMedicationEvent).toHaveBeenCalled();
    });

    const select = screen.getByRole("combobox", { name: "Medication" });
    expect(select).toBeDisabled();
  });
});
