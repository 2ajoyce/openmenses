import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import { EmptyState } from "../EmptyState";

// Mock framework7-react
vi.mock("framework7-react", () => ({
  Block: ({ children, ...props }: { children: React.ReactNode }) => (
    <div {...props}>{children}</div>
  ),
  Button: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => <button onClick={onClick}>{children}</button>,
}));

describe("EmptyState", () => {
  it("renders message", () => {
    render(<EmptyState message="No data found" />);
    expect(screen.getByText("No data found")).toBeInTheDocument();
  });

  it("renders action button when provided", () => {
    const onAction = vi.fn();
    render(
      <EmptyState
        message="No data"
        actionLabel="Add Item"
        onAction={onAction}
      />,
    );
    const btn = screen.getByText("Add Item");
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(onAction).toHaveBeenCalled();
  });

  it("does not render action button when no label", () => {
    render(<EmptyState message="No data" />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
