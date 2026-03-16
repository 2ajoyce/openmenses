import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import SettingsPage from "../SettingsPage";
import {
  BiologicalCycleModel,
  CycleRegularity,
  TrackingFocus,
} from "@gen/openmenses/v1/model_pb";

const mockGetUserProfile = vi.fn();
const mockCreateUserProfile = vi.fn();
const mockUpdateUserProfile = vi.fn();

vi.mock("../../lib/client", () => ({
  client: {
    getUserProfile: (...args: unknown[]) => mockGetUserProfile(...args),
    createUserProfile: (...args: unknown[]) => mockCreateUserProfile(...args),
    updateUserProfile: (...args: unknown[]) => mockUpdateUserProfile(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react", () => ({
  Page: ({ children }: { children: React.ReactNode }) => <div data-testid="page">{children}</div>,
  Navbar: ({ title }: { title: string }) => <div data-testid="navbar">{title}</div>,
  Block: ({ children }: { children: React.ReactNode }) => <div data-testid="block">{children}</div>,
  BlockTitle: ({ children }: { children: React.ReactNode }) => <h3>{children}</h3>,
  List: ({ children }: { children: React.ReactNode }) => <ul>{children}</ul>,
  ListItem: ({ title }: { title: string }) => <li>{title}</li>,
  Button: ({ children, onClick, disabled }: { children: React.ReactNode; onClick?: () => void; disabled?: boolean }) => (
    <button onClick={onClick} disabled={disabled} data-testid="button">{children}</button>
  ),
  Icon: ({ ios, md }: { ios: string; md: string }) => <span data-icon={ios || md} />,
}));

describe("SettingsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetUserProfile.mockResolvedValue({ profile: null });
  });

  it("shows loading state initially", () => {
    mockGetUserProfile.mockImplementation(
      () =>
        new Promise(() => {
          /* never resolves */
        }),
    );

    render(<SettingsPage />);

    expect(screen.getByText("Loading profile...")).toBeInTheDocument();
  });

  it("renders profile form for first-time user (no existing profile)", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    expect(screen.getByLabelText("Cycle Regularity")).toBeInTheDocument();
    // Tracking Focus is rendered in the page
    expect(screen.getByRole("checkbox", { name: /Bleeding/ })).toBeInTheDocument();
  });

  it("populates form with existing profile data", async () => {
    mockGetUserProfile.mockResolvedValue({
      profile: {
        name: "users/default",
        biologicalCycle: BiologicalCycleModel.OVULATORY,
        cycleRegularity: CycleRegularity.REGULAR,
        trackingFocus: [TrackingFocus.BLEEDING, TrackingFocus.SYMPTOMS],
      },
    });

    render(<SettingsPage />);

    await waitFor(() => {
      const biologicalCycleSelect = screen.getByLabelText(
        "Biological Cycle Model",
      ) as HTMLSelectElement;
      expect(biologicalCycleSelect.value).toBe(
        String(BiologicalCycleModel.OVULATORY),
      );
    });

    const cycleRegularitySelect = screen.getByLabelText(
      "Cycle Regularity",
    ) as HTMLSelectElement;
    expect(cycleRegularitySelect.value).toBe(String(CycleRegularity.REGULAR));

    // Check that tracking focus checkboxes are checked
    const bleedingCheckbox = screen.getByRole("checkbox", {
      name: /Bleeding/,
    }) as HTMLInputElement;
    expect(bleedingCheckbox.checked).toBe(true);

    const symptomsCheckbox = screen.getByRole("checkbox", {
      name: /Symptoms/,
    }) as HTMLInputElement;
    expect(symptomsCheckbox.checked).toBe(true);
  });

  it("allows changing biological cycle model", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    const select = screen.getByLabelText("Biological Cycle Model");
    fireEvent.change(select, {
      target: { value: String(BiologicalCycleModel.HORMONALLY_SUPPRESSED) },
    });

    expect((select as HTMLSelectElement).value).toBe(
      String(BiologicalCycleModel.HORMONALLY_SUPPRESSED),
    );
  });

  it("allows changing cycle regularity", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Cycle Regularity")).toBeInTheDocument();
    });

    const select = screen.getByLabelText("Cycle Regularity");
    fireEvent.change(select, {
      target: { value: String(CycleRegularity.VERY_IRREGULAR) },
    });

    expect((select as HTMLSelectElement).value).toBe(
      String(CycleRegularity.VERY_IRREGULAR),
    );
  });

  it("allows selecting multiple tracking focus options", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByRole("checkbox", {
          name: /Bleeding/,
        }),
      ).toBeInTheDocument();
    });

    const bleedingCheckbox = screen.getByRole("checkbox", {
      name: /Bleeding/,
    });
    const symptomsCheckbox = screen.getByRole("checkbox", {
      name: /Symptoms/,
    });

    fireEvent.click(bleedingCheckbox);
    fireEvent.click(symptomsCheckbox);

    expect((bleedingCheckbox as HTMLInputElement).checked).toBe(true);
    expect((symptomsCheckbox as HTMLInputElement).checked).toBe(true);
  });

  it("creates a new profile when saving for first-time user", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });
    mockCreateUserProfile.mockResolvedValue({
      profile: {
        name: "users/default",
        biologicalCycle: BiologicalCycleModel.OVULATORY,
        cycleRegularity: CycleRegularity.REGULAR,
        trackingFocus: [TrackingFocus.BLEEDING],
      },
    });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    // Fill out form
    const biologicalCycleSelect = screen.getByLabelText(
      "Biological Cycle Model",
    );
    fireEvent.change(biologicalCycleSelect, {
      target: { value: String(BiologicalCycleModel.OVULATORY) },
    });

    const cycleRegularitySelect = screen.getByLabelText("Cycle Regularity");
    fireEvent.change(cycleRegularitySelect, {
      target: { value: String(CycleRegularity.REGULAR) },
    });

    const bleedingCheckbox = screen.getByRole("checkbox", {
      name: /Bleeding/,
    });
    fireEvent.click(bleedingCheckbox);

    // Save
    const saveButton = screen.getByText("Save Profile");
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockCreateUserProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          profile: expect.objectContaining({
            biologicalCycle: BiologicalCycleModel.OVULATORY,
            cycleRegularity: CycleRegularity.REGULAR,
            trackingFocus: [TrackingFocus.BLEEDING],
          }),
        }),
      );
    });
  });

  it("updates existing profile when saving", async () => {
    mockGetUserProfile.mockResolvedValue({
      profile: {
        name: "users/default",
        biologicalCycle: BiologicalCycleModel.OVULATORY,
        cycleRegularity: CycleRegularity.REGULAR,
        trackingFocus: [TrackingFocus.BLEEDING],
      },
    });
    mockUpdateUserProfile.mockResolvedValue({
      profile: {
        name: "users/default",
        biologicalCycle: BiologicalCycleModel.IRREGULAR,
        cycleRegularity: CycleRegularity.VERY_IRREGULAR,
        trackingFocus: [TrackingFocus.BLEEDING, TrackingFocus.SYMPTOMS],
      },
    });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    // Change biological cycle model
    const biologicalCycleSelect = screen.getByLabelText(
      "Biological Cycle Model",
    );
    fireEvent.change(biologicalCycleSelect, {
      target: { value: String(BiologicalCycleModel.IRREGULAR) },
    });

    // Add another tracking focus
    const symptomsCheckbox = screen.getByRole("checkbox", {
      name: /Symptoms/,
    });
    fireEvent.click(symptomsCheckbox);

    // Save
    const saveButton = screen.getByText("Save Profile");
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateUserProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          profile: expect.objectContaining({
            biologicalCycle: BiologicalCycleModel.IRREGULAR,
          }),
        }),
      );
    });
  });

  it("shows validation error when required fields are missing", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(screen.getByText("Save Profile")).toBeInTheDocument();
    });

    // Try to save without filling form
    const saveButton = screen.getByText("Save Profile");
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(
        screen.getByText(
          /Please fill in all fields: biological cycle model, cycle regularity, and at least one tracking focus/,
        ),
      ).toBeInTheDocument();
    });
  });

  it("shows error when tracking focus is empty", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    // Fill biological cycle and regularity but not tracking focus
    const biologicalCycleSelect = screen.getByLabelText(
      "Biological Cycle Model",
    );
    fireEvent.change(biologicalCycleSelect, {
      target: { value: String(BiologicalCycleModel.OVULATORY) },
    });

    const cycleRegularitySelect = screen.getByLabelText("Cycle Regularity");
    fireEvent.change(cycleRegularitySelect, {
      target: { value: String(CycleRegularity.REGULAR) },
    });

    // Try to save
    const saveButton = screen.getByText("Save Profile");
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(
        screen.getByText(
          /Please fill in all fields: biological cycle model, cycle regularity, and at least one tracking focus/,
        ),
      ).toBeInTheDocument();
    });
  });

  it("shows success message after saving", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });
    mockCreateUserProfile.mockResolvedValue({
      profile: {
        name: "users/default",
        biologicalCycle: BiologicalCycleModel.OVULATORY,
        cycleRegularity: CycleRegularity.REGULAR,
        trackingFocus: [TrackingFocus.BLEEDING],
      },
    });

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    // Fill and save
    const biologicalCycleSelect = screen.getByLabelText(
      "Biological Cycle Model",
    );
    fireEvent.change(biologicalCycleSelect, {
      target: { value: String(BiologicalCycleModel.OVULATORY) },
    });

    const cycleRegularitySelect = screen.getByLabelText("Cycle Regularity");
    fireEvent.change(cycleRegularitySelect, {
      target: { value: String(CycleRegularity.REGULAR) },
    });

    const bleedingCheckbox = screen.getByRole("checkbox", {
      name: /Bleeding/,
    });
    fireEvent.click(bleedingCheckbox);

    const saveButton = screen.getByText("Save Profile");
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(
        screen.getByText("Profile saved successfully"),
      ).toBeInTheDocument();
    });
  });

  it("disables inputs while saving", async () => {
    mockGetUserProfile.mockResolvedValue({ profile: null });
    let resolveCreate!: (value: unknown) => void;
    mockCreateUserProfile.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        }),
    );

    render(<SettingsPage />);

    await waitFor(() => {
      expect(
        screen.getByLabelText("Biological Cycle Model"),
      ).toBeInTheDocument();
    });

    // Fill and save
    const biologicalCycleSelect = screen.getByLabelText(
      "Biological Cycle Model",
    );
    fireEvent.change(biologicalCycleSelect, {
      target: { value: String(BiologicalCycleModel.OVULATORY) },
    });

    const cycleRegularitySelect = screen.getByLabelText("Cycle Regularity");
    fireEvent.change(cycleRegularitySelect, {
      target: { value: String(CycleRegularity.REGULAR) },
    });

    const bleedingCheckbox = screen.getByRole("checkbox", {
      name: /Bleeding/,
    });
    fireEvent.click(bleedingCheckbox);

    const saveButton = screen.getByText("Save Profile") as HTMLButtonElement;
    fireEvent.click(saveButton);

    // While saving, button should be disabled
    expect(saveButton.disabled).toBe(true);

    // Resolve the save
    resolveCreate({
      profile: {
        name: "users/default",
        biologicalCycle: BiologicalCycleModel.OVULATORY,
        cycleRegularity: CycleRegularity.REGULAR,
        trackingFocus: [TrackingFocus.BLEEDING],
      },
    });

    await waitFor(() => {
      expect(saveButton.disabled).toBe(false);
    });
  });
});
