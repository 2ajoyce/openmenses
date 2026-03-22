import { render, screen } from "@testing-library/react";
import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ChartContainer } from "./ChartContainer";

// Mock ResizeObserver before importing recharts
globalThis.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
})) as unknown as typeof ResizeObserver;

// Mock Framework7 context for Icon and Block components
vi.mock("framework7-react", async () => {
  const actual = await vi.importActual("framework7-react");
  return {
    ...actual,
    Icon: ({ f7 }: { f7: string }) => <span data-testid="icon">{f7}</span>,
    Block: ({
      children,
      className,
    }: {
      children: React.ReactNode;
      className?: string;
    }) => (
      <div className={className} data-testid="block">
        {children}
      </div>
    ),
  };
});

describe("ChartContainer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders empty state when data is not provided", () => {
    render(
      <ChartContainer>
        <div>Chart content</div>
      </ChartContainer>,
    );

    expect(
      screen.getByText("No data available for this chart"),
    ).toBeInTheDocument();
  });

  it("renders empty state when data array is empty", () => {
    render(
      <ChartContainer data={[]}>
        <div>Chart content</div>
      </ChartContainer>,
    );

    expect(
      screen.getByText("No data available for this chart"),
    ).toBeInTheDocument();
  });

  it("renders children when data is provided", () => {
    const testData = [{ value: 1 }, { value: 2 }];

    render(
      <ChartContainer data={testData}>
        <div>Chart content</div>
      </ChartContainer>,
    );

    expect(screen.getByText("Chart content")).toBeInTheDocument();
    expect(
      screen.queryByText("No data available for this chart"),
    ).not.toBeInTheDocument();
  });

  it("renders title when provided", () => {
    const testData = [{ value: 1 }];

    render(
      <ChartContainer data={testData} title="Test Chart">
        <div>Chart content</div>
      </ChartContainer>,
    );

    expect(screen.getByText("Test Chart")).toBeInTheDocument();
  });

  it("does not render title when not provided", () => {
    const testData = [{ value: 1 }];

    render(
      <ChartContainer data={testData}>
        <div>Chart content</div>
      </ChartContainer>,
    );

    // Just verify that no h3 was created
    const headings = screen.queryAllByRole("heading", { level: 3 });
    expect(headings).toHaveLength(0);
  });

  it("renders description when provided", () => {
    const testData = [{ value: 1 }];

    render(
      <ChartContainer
        data={testData}
        description="A helpful explanation of this chart."
      >
        <div>Chart content</div>
      </ChartContainer>,
    );

    expect(
      screen.getByText("A helpful explanation of this chart."),
    ).toBeInTheDocument();
  });

  it("does not render description when not provided", () => {
    const testData = [{ value: 1 }];

    render(
      <ChartContainer data={testData}>
        <div>Chart content</div>
      </ChartContainer>,
    );

    expect(screen.queryByRole("paragraph")).not.toBeInTheDocument();
  });
});
