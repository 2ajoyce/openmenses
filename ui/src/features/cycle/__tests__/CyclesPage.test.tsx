import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import CyclesPage from "../CyclesPage";
import { CycleSource, CycleSchema } from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";

const mockNavigate = vi.fn();
const mockRouter = { navigate: mockNavigate } as never;

const mockListCycles = vi.fn();
const mockGetCycleStatistics = vi.fn();
const mockGetUserProfile = vi.fn();
const mockListTimeline = vi.fn();
const mockListPredictions = vi.fn();

vi.mock("../../../lib/client", () => ({
  client: {
    listCycles: (...args: unknown[]) => mockListCycles(...args),
    getCycleStatistics: (...args: unknown[]) => mockGetCycleStatistics(...args),
    getUserProfile: (...args: unknown[]) => mockGetUserProfile(...args),
    listTimeline: (...args: unknown[]) => mockListTimeline(...args),
    listPredictions: (...args: unknown[]) => mockListPredictions(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react", () => ({
  Page: ({ children }: { children: React.ReactNode }) => <div data-testid="page">{children}</div>,
  Navbar: ({ title }: { title: string }) => <div data-testid="navbar">{title}</div>,
  Block: ({ children }: { children: React.ReactNode }) => <div data-testid="block">{children}</div>,
  BlockTitle: ({ children }: { children: React.ReactNode }) => <h3>{children}</h3>,
  Card: ({ children }: { children: React.ReactNode }) => <div data-testid="card">{children}</div>,
  CardHeader: ({ children }: { children: React.ReactNode }) => <div data-testid="card-header">{children}</div>,
  CardContent: ({ children }: { children: React.ReactNode }) => <div data-testid="card-content">{children}</div>,
  Button: ({ children, onClick }: { children: React.ReactNode; onClick?: () => void }) => (
    <button onClick={onClick}>{children}</button>
  ),
  Segmented: ({ children }: { children: React.ReactNode }) => <div data-testid="segmented">{children}</div>,
  SegmentedButton: ({ children, onClick, active }: { children: React.ReactNode; onClick?: () => void; active?: boolean }) => (
    <button onClick={onClick} data-active={active}>{children}</button>
  ),
}));

describe("CyclesPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListCycles.mockResolvedValue({ cycles: [] });
    mockGetCycleStatistics.mockResolvedValue({ statistics: null });
    mockGetUserProfile.mockResolvedValue({ profile: null });
    mockListTimeline.mockResolvedValue({ records: [] });
    mockListPredictions.mockResolvedValue({ predictions: [] });
  });

  it("shows loading state when initially loading", () => {
    mockListCycles.mockImplementation(
      () =>
        new Promise(() => {
          /* never resolves */
        }),
    );

    render(<CyclesPage f7router={mockRouter} />);

    expect(screen.getByText("Loading cycles...")).toBeInTheDocument();
  });

  it("shows empty state when no cycles exist", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("No cycles detected yet")).toBeInTheDocument();
    });
  });

  it("shows empty state action button", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(
        screen.getByText("Log your first observation"),
      ).toBeInTheDocument();
    });
  });

  it("navigates to log page when empty state action is clicked", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(
        screen.getByText("Log your first observation"),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("Log your first observation"));
    expect(mockNavigate).toHaveBeenCalledWith("/log/");
  });

  it("renders cycles when they exist", async () => {
    const cycles = [
      create(CycleSchema, {
        name: "cycles/1",
        startDate: create(LocalDateSchema, { value: "2026-03-01" }),
        endDate: create(LocalDateSchema, { value: "2026-03-28" }),
        source: CycleSource.DERIVED_FROM_BLEEDING,
        userId: "users/default",
      }),
    ];

    mockListCycles.mockResolvedValue({ cycles });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText(/Derived from bleeding/)).toBeInTheDocument();
    });
  });

  it("fetches user profile on load", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockGetUserProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "users/default",
        }),
      );
    });
  });

  it("fetches cycle statistics on load", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockGetCycleStatistics).toHaveBeenCalledWith(
        expect.objectContaining({
          parent: "users/default",
          windowSize: 0,
        }),
      );
    });
  });

  it("fetches cycles with pagination", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListCycles).toHaveBeenCalledWith(
        expect.objectContaining({
          parent: "users/default",
          pagination: expect.objectContaining({
            pageSize: 100,
            pageToken: "",
          }),
        }),
      );
    });
  });

  it("fetches today's phase estimate via listTimeline", async () => {
    mockListCycles.mockResolvedValue({ cycles: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListTimeline).toHaveBeenCalledWith(
        expect.objectContaining({
          parent: "users/default",
        }),
      );
    });
  });

  it("fetches predictions on load", async () => {
    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListPredictions).toHaveBeenCalledWith(
        expect.objectContaining({
          parent: "users/default",
        }),
      );
    });
  });

  it("renders predictions section when predictions are available", async () => {
    mockListPredictions.mockResolvedValue({
      predictions: [
        {
          name: "users/default/predictions/01",
          userId: "users/default",
          kind: 1, // NEXT_BLEED
          predictedStartDate: { value: "2026-04-01" },
          predictedEndDate: { value: "2026-04-06" },
          confidence: 3, // HIGH
          rationale: [],
        },
      ],
    });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(screen.getByText("Next Period")).toBeInTheDocument();
    });
  });

  it("does not render predictions section when predictions are empty", async () => {
    mockListPredictions.mockResolvedValue({ predictions: [] });

    render(<CyclesPage f7router={mockRouter} />);

    await waitFor(() => {
      expect(mockListPredictions).toHaveBeenCalled();
    });

    expect(screen.queryByText("Predictions")).not.toBeInTheDocument();
  });
});
