import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import ClinicianSummaryPage from "./ClinicianSummaryPage";
import {
  BiologicalCycleModel,
  CycleRegularity,
  MedicationCategory,
  PredictionType,
  InsightType,
  ConfidenceLevel,
  CycleSource,
  UserProfileSchema,
  CycleStatisticsSchema,
  CycleSchema,
  MedicationSchema,
  PredictionSchema,
  InsightSchema,
} from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";

const mockGetUserProfile = vi.fn();
const mockGetCycleStatistics = vi.fn();
const mockListCycles = vi.fn();
const mockListMedications = vi.fn();
const mockListPredictions = vi.fn();
const mockListInsights = vi.fn();

vi.mock("../../lib/client", () => ({
  client: {
    getUserProfile: (...args: unknown[]) => mockGetUserProfile(...args),
    getCycleStatistics: (...args: unknown[]) => mockGetCycleStatistics(...args),
    listCycles: (...args: unknown[]) => mockListCycles(...args),
    listMedications: (...args: unknown[]) => mockListMedications(...args),
    listPredictions: (...args: unknown[]) => mockListPredictions(...args),
    listInsights: (...args: unknown[]) => mockListInsights(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react", () => ({
  Page: ({ children, className }: { children: React.ReactNode; className?: string }) => (
    <div data-testid="page" className={className}>
      {children}
    </div>
  ),
  Navbar: ({ title }: { title: string }) => (
    <div data-testid="navbar">{title}</div>
  ),
  Block: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="block">{children}</div>
  ),
  Button: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button onClick={onClick} data-testid="button">
      {children}
    </button>
  ),
}));

describe("ClinicianSummaryPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const mockProfile = create(UserProfileSchema, {
    name: "users/default",
    biologicalCycle: BiologicalCycleModel.OVULATORY,
    cycleRegularity: CycleRegularity.REGULAR,
    trackingFocus: [],
  });

  const mockStatistics = create(CycleStatisticsSchema, {
    count: 12,
    average: 28.5,
    median: 28,
    min: 26,
    max: 32,
    stdDev: 1.8,
  });

  const mockCycles = [
    create(CycleSchema, {
      name: "users/default/cycles/1",
      startDate: create(LocalDateSchema, { value: "2024-03-01" }),
      endDate: create(LocalDateSchema, { value: "2024-03-29" }),
      source: CycleSource.USER_CONFIRMED,
    }),
  ];

  const mockMedications = [
    create(MedicationSchema, {
      name: "users/default/medications/1",
      displayName: "Birth Control Pill",
      category: MedicationCategory.BIRTH_CONTROL,
      active: true,
    }),
  ];

  const mockPredictions = [
    create(PredictionSchema, {
      name: "users/default/predictions/1",
      kind: PredictionType.NEXT_BLEED,
      predictedStartDate: create(LocalDateSchema, { value: "2024-04-15" }),
      predictedEndDate: create(LocalDateSchema, { value: "2024-04-18" }),
      confidence: ConfidenceLevel.HIGH,
    }),
  ];

  const mockInsights = [
    create(InsightSchema, {
      name: "users/default/insights/1",
      kind: InsightType.CYCLE_LENGTH_PATTERN,
      summary: "Your cycles are consistently 28-29 days long.",
      confidence: ConfidenceLevel.HIGH,
    }),
  ];

  it("should render clinician summary page with all sections", async () => {
    mockGetUserProfile.mockResolvedValueOnce({ profile: mockProfile });
    mockGetCycleStatistics.mockResolvedValueOnce({ statistics: mockStatistics });
    mockListCycles.mockResolvedValueOnce({ cycles: mockCycles });
    mockListMedications.mockResolvedValueOnce({ medications: mockMedications });
    mockListPredictions.mockResolvedValueOnce({ predictions: mockPredictions });
    mockListInsights.mockResolvedValueOnce({ insights: mockInsights });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<ClinicianSummaryPage f7router={{} as any} />);

    await waitFor(() => {
      expect(screen.getByText("Cycle Health Summary")).toBeInTheDocument();
      expect(screen.getByText("Profile")).toBeInTheDocument();
      expect(screen.getByText("Cycle Statistics")).toBeInTheDocument();
      expect(screen.getByText("Recent Cycles")).toBeInTheDocument();
      expect(screen.getByText("Active Medications")).toBeInTheDocument();
      expect(screen.getByText("Current Predictions")).toBeInTheDocument();
      expect(screen.getByText("Insights")).toBeInTheDocument();
    });
  });

  it("should render print button", async () => {
    mockGetUserProfile.mockResolvedValueOnce({ profile: mockProfile });
    mockGetCycleStatistics.mockResolvedValueOnce({ statistics: mockStatistics });
    mockListCycles.mockResolvedValueOnce({ cycles: mockCycles });
    mockListMedications.mockResolvedValueOnce({ medications: mockMedications });
    mockListPredictions.mockResolvedValueOnce({ predictions: mockPredictions });
    mockListInsights.mockResolvedValueOnce({ insights: mockInsights });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<ClinicianSummaryPage f7router={{} as any} />);

    await waitFor(() => {
      expect(screen.getByText("Print Summary")).toBeInTheDocument();
    });
  });

  it("should display profile data", async () => {
    mockGetUserProfile.mockResolvedValueOnce({ profile: mockProfile });
    mockGetCycleStatistics.mockResolvedValueOnce({ statistics: mockStatistics });
    mockListCycles.mockResolvedValueOnce({ cycles: mockCycles });
    mockListMedications.mockResolvedValueOnce({ medications: mockMedications });
    mockListPredictions.mockResolvedValueOnce({ predictions: mockPredictions });
    mockListInsights.mockResolvedValueOnce({ insights: mockInsights });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<ClinicianSummaryPage f7router={{} as any} />);

    await waitFor(() => {
      expect(screen.getByText("Ovulatory")).toBeInTheDocument();
      expect(screen.getByText("Regular")).toBeInTheDocument();
      // medication display name (not resource name)
      expect(screen.getByText("Birth Control Pill")).toBeInTheDocument();
      // cycle length column
      expect(screen.getByText("Length")).toBeInTheDocument();
      expect(screen.getByText("28 days")).toBeInTheDocument();
    });
  });

  it("should show empty state when no data", async () => {
    mockGetUserProfile.mockResolvedValueOnce({ profile: null });
    mockGetCycleStatistics.mockResolvedValueOnce({ statistics: null });
    mockListCycles.mockResolvedValueOnce({ cycles: [] });
    mockListMedications.mockResolvedValueOnce({ medications: [] });
    mockListPredictions.mockResolvedValueOnce({ predictions: [] });
    mockListInsights.mockResolvedValueOnce({ insights: [] });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<ClinicianSummaryPage f7router={{} as any} />);

    await waitFor(() => {
      expect(
        screen.getAllByText("No profile data available")[0]
      ).toBeInTheDocument();
      expect(screen.getByText("No cycle data available")).toBeInTheDocument();
      expect(
        screen.getByText("No active medications")
      ).toBeInTheDocument();
      expect(
        screen.getByText("No predictions available")
      ).toBeInTheDocument();
      expect(screen.getByText("No insights available")).toBeInTheDocument();
    });
  });

  it("should call print when button is clicked", async () => {
    const printSpy = vi.spyOn(window, "print").mockImplementation(() => {});

    mockGetUserProfile.mockResolvedValueOnce({ profile: mockProfile });
    mockGetCycleStatistics.mockResolvedValueOnce({ statistics: mockStatistics });
    mockListCycles.mockResolvedValueOnce({ cycles: mockCycles });
    mockListMedications.mockResolvedValueOnce({ medications: mockMedications });
    mockListPredictions.mockResolvedValueOnce({ predictions: mockPredictions });
    mockListInsights.mockResolvedValueOnce({ insights: mockInsights });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<ClinicianSummaryPage f7router={{} as any} />);

    const printButton = await screen.findByText("Print Summary");
    await userEvent.click(printButton);

    expect(printSpy).toHaveBeenCalled();
  });
});
