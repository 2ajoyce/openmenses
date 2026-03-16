import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import TimelinePage from "../TimelinePage";

const mockNavigate = vi.fn();
const mockRouter = { navigate: mockNavigate } as never;

const mockListTimeline = vi.fn();
const mockListMedications = vi.fn();
const mockGetUserProfile = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    listTimeline: (...args: unknown[]) => mockListTimeline(...args),
    listMedications: (...args: unknown[]) => mockListMedications(...args),
    getUserProfile: (...args: unknown[]) => mockGetUserProfile(...args),
    deleteBleedingObservation: vi.fn().mockResolvedValue({}),
    deleteSymptomObservation: vi.fn().mockResolvedValue({}),
    deleteMoodObservation: vi.fn().mockResolvedValue({}),
    deleteMedicationEvent: vi.fn().mockResolvedValue({}),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react");

const bleedingRecord = {
  record: {
    case: "bleedingObservation" as const,
    value: {
      name: "obs-1",
      flow: 2,
      note: "light bleeding",
      timestamp: { value: "2026-03-10T12:00:00Z" },
      userId: "users/default",
    },
  },
};

const symptomRecord = {
  record: {
    case: "symptomObservation" as const,
    value: {
      name: "obs-2",
      symptom: 1,
      severity: 2,
      note: "",
      timestamp: { value: "2026-03-09T10:00:00Z" },
      userId: "users/default",
    },
  },
};

const moodRecord = {
  record: {
    case: "moodObservation" as const,
    value: {
      name: "obs-3",
      mood: 1,
      intensity: 2,
      note: "",
      timestamp: { value: "2026-03-08T09:00:00Z" },
      userId: "users/default",
    },
  },
};

const medicationRecord = {
  record: {
    case: "medication" as const,
    value: {
      name: "med-1",
      displayName: "Yasmin",
      category: 1,
      active: true,
      note: "",
      userId: "users/default",
    },
  },
};

const medicationEventRecord = {
  record: {
    case: "medicationEvent" as const,
    value: {
      name: "evt-1",
      medicationId: "med-1",
      status: 1,
      dose: "200mg",
      note: "",
      timestamp: { value: "2026-03-07T08:00:00Z" },
      userId: "users/default",
    },
  },
};

describe("TimelinePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListMedications.mockResolvedValue({ medications: [] });
    mockGetUserProfile.mockResolvedValue({ profile: {} });
  });

  it("renders timeline with mixed observation types", async () => {
    mockListTimeline.mockResolvedValue({
      records: [bleedingRecord, symptomRecord, moodRecord],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Timeline")).toBeInTheDocument();
    });

    // Bleeding card shows flow label
    expect(screen.getByText(/Light/)).toBeInTheDocument();
    // Symptom card shows type label
    expect(screen.getByText(/Cramps/)).toBeInTheDocument();
    // Mood card shows mood label
    expect(screen.getByText(/Calm/)).toBeInTheDocument();
  });

  it("shows empty state when no records", async () => {
    mockListTimeline.mockResolvedValue({
      records: [],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(
        screen.getByText("No observations logged yet"),
      ).toBeInTheDocument();
    });
  });

  it("renders filter chips", async () => {
    mockListTimeline.mockResolvedValue({
      records: [bleedingRecord, symptomRecord],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Bleeding")).toBeInTheDocument();
      expect(screen.getByText("Symptoms")).toBeInTheDocument();
      expect(screen.getByText("Mood")).toBeInTheDocument();
      expect(screen.getByText("Medication")).toBeInTheDocument();
    });
  });

  it("filters records when chip is clicked", async () => {
    mockListTimeline.mockResolvedValue({
      records: [bleedingRecord, symptomRecord, moodRecord],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText(/Light/)).toBeInTheDocument();
    });

    // Click "Bleeding" chip to filter
    fireEvent.click(screen.getByText("Bleeding"));

    // Bleeding should still be visible
    expect(screen.getByText(/Light/)).toBeInTheDocument();
    // Other types should be filtered out
    expect(screen.queryByText(/Cramps/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Calm/)).not.toBeInTheDocument();
  });

  it("navigates to log page from empty state action", async () => {
    mockListTimeline.mockResolvedValue({
      records: [],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(
        screen.getByText("Log your first observation"),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Log your first observation"));
    expect(mockNavigate).toHaveBeenCalledWith("/log/");
  });

  it("renders medication record via MedicationCard", async () => {
    mockListTimeline.mockResolvedValue({
      records: [medicationRecord],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText(/Yasmin/)).toBeInTheDocument();
    });
  });

  it("resolves medication name for medication event records", async () => {
    mockListMedications.mockResolvedValue({
      medications: [
        {
          name: "med-1",
          displayName: "Birth Control Pill",
          category: 1,
          active: true,
          note: "",
        },
      ],
    });
    mockListTimeline.mockResolvedValue({
      records: [medicationEventRecord],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText(/Birth Control Pill/)).toBeInTheDocument();
    });
  });

  it("shows medication records when Medication chip is active", async () => {
    mockListTimeline.mockResolvedValue({
      records: [medicationRecord, bleedingRecord],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText(/Yasmin/)).toBeInTheDocument();
      expect(screen.getByText(/Light/)).toBeInTheDocument();
    });

    // Click "Medication" chip to filter
    fireEvent.click(screen.getByText("Medication"));

    // Medication record should still be visible
    expect(screen.getByText(/Yasmin/)).toBeInTheDocument();
    // Bleeding should be filtered out
    expect(screen.queryByText(/Light/)).not.toBeInTheDocument();
  });

  it("calls listTimeline only once for loadMore when in-flight guard is active", async () => {
    let resolveFirst!: (value: unknown) => void;
    const firstPagePromise = new Promise((resolve) => {
      resolveFirst = resolve;
    });

    mockListTimeline
      .mockReturnValueOnce(
        firstPagePromise.then(() => ({
          records: [bleedingRecord],
          pagination: { nextPageToken: "token-1" },
        })),
      )
      .mockResolvedValue({
        records: [],
        pagination: { nextPageToken: "" },
      });

    render(<TimelinePage f7router={mockRouter} />);

    // Wait for initial fetch to complete (first call returns with nextPageToken)
    resolveFirst(undefined);
    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledTimes(1);
    });

    // initial fetch resolves with nextPageToken, so loadMore can now be called
    await waitFor(() => {
      expect(screen.getByTestId("trigger-infinite")).toBeInTheDocument();
    });

    // Block the second listTimeline call so the guard is active during both clicks
    let resolveSecond!: (value: unknown) => void;
    const secondPagePromise = new Promise((resolve) => {
      resolveSecond = resolve;
    });
    mockListTimeline.mockReturnValueOnce(
      secondPagePromise.then(() => ({
        records: [],
        pagination: { nextPageToken: "" },
      })),
    );

    const triggerBtn = screen.getByTestId("trigger-infinite");
    // Fire onInfinite twice before the second fetch resolves
    fireEvent.click(triggerBtn);
    fireEvent.click(triggerBtn);

    // Resolve the second fetch
    resolveSecond(undefined);

    await waitFor(() => {
      // Initial fetch + exactly one paginated fetch (guard blocked the duplicate)
      expect(mockListTimeline).toHaveBeenCalledTimes(2);
    });
  });

  it("renders From and To date pickers", async () => {
    mockListTimeline.mockResolvedValue({
      records: [],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByLabelText("From")).toBeInTheDocument();
      expect(screen.getByLabelText("To")).toBeInTheDocument();
    });
  });

  it("re-fetches timeline with updated range.start when start date changes", async () => {
    mockListTimeline.mockResolvedValue({
      records: [],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledTimes(1);
    });

    const fromInput = screen.getByLabelText("From");
    fireEvent.change(fromInput, { target: { value: "2026-01-01T00:00" } });

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledTimes(2);
    });
    expect(mockListTimeline).toHaveBeenLastCalledWith(
      expect.objectContaining({
        range: expect.objectContaining({
          start: expect.objectContaining({ value: "2026-01-01" }),
        }),
      }),
    );
  });

  it("re-fetches timeline with updated range.end when end date changes", async () => {
    mockListTimeline.mockResolvedValue({
      records: [],
      pagination: { nextPageToken: "" },
    });

    render(<TimelinePage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledTimes(1);
    });

    const toInput = screen.getByLabelText("To");
    fireEvent.change(toInput, { target: { value: "2026-04-01T00:00" } });

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledTimes(2);
    });
    expect(mockListTimeline).toHaveBeenLastCalledWith(
      expect.objectContaining({
        range: expect.objectContaining({
          end: expect.objectContaining({ value: "2026-04-01" }),
        }),
      }),
    );
  });
});
