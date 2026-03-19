import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ExportPage from "./ExportPage";

const mockCreateDataExport = vi.fn();

vi.mock("../../lib/client", () => ({
  client: {
    createDataExport: (...args: unknown[]) => mockCreateDataExport(...args),
  },
  DEFAULT_PARENT: "users/default",
}));

vi.mock("framework7-react", () => ({
  Page: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="page">{children}</div>
  ),
  Navbar: ({ title }: { title: string }) => (
    <div data-testid="navbar">{title}</div>
  ),
  Block: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="block">{children}</div>
  ),
  BlockTitle: ({ children }: { children: React.ReactNode }) => (
    <h3>{children}</h3>
  ),
  Button: ({
    children,
    onClick,
    disabled,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button onClick={onClick} disabled={disabled} data-testid="button">
      {children}
    </button>
  ),
  Icon: ({ ios, md }: { ios: string; md: string }) => (
    <span data-icon={ios || md} />
  ),
}));

describe("ExportPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should render export buttons", () => {
    render(<ExportPage />);

    expect(screen.getByText("Export as JSON")).toBeInTheDocument();
    expect(screen.getByText("Export as CSV")).toBeInTheDocument();
  });

  it("should render data export description", () => {
    render(<ExportPage />);

    expect(
      screen.getByText(/Download your cycle tracking data/)
    ).toBeInTheDocument();
  });

  it("should call createDataExport RPC when Export as JSON is clicked", async () => {
    const mockData = JSON.stringify({
      version: "1",
      user_id: "users/default",
    });
    const mockResponse = {
      data: new TextEncoder().encode(mockData),
    };

    mockCreateDataExport.mockResolvedValueOnce(mockResponse);

    render(<ExportPage />);

    const jsonButton = screen.getByText("Export as JSON");
    await userEvent.click(jsonButton);

    await waitFor(() => {
      expect(mockCreateDataExport).toHaveBeenCalledWith({
        parent: "users/default",
      });
    });
  });

  it("should call createDataExport RPC when Export as CSV is clicked", async () => {
    const mockData = JSON.stringify({
      version: "1",
      user_id: "users/default",
    });
    const mockResponse = {
      data: new TextEncoder().encode(mockData),
    };

    mockCreateDataExport.mockResolvedValueOnce(mockResponse);

    render(<ExportPage />);

    const csvButton = screen.getByText("Export as CSV");
    await userEvent.click(csvButton);

    await waitFor(() => {
      expect(mockCreateDataExport).toHaveBeenCalledWith({
        parent: "users/default",
      });
    });
  });

  it("should display error on export failure", async () => {
    const errorMessage = "Network error";
    mockCreateDataExport.mockRejectedValueOnce(new Error(errorMessage));

    render(<ExportPage />);

    const jsonButton = screen.getByText("Export as JSON");
    await userEvent.click(jsonButton);

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });
  });
});
