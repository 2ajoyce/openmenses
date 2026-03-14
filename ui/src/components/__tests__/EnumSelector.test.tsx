import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import { EnumSelector } from "../EnumSelector";

// Mock framework7-react Button
vi.mock("framework7-react", () => ({
  Button: ({
    children,
    onClick,
    fill,
  }: {
    children: React.ReactNode;
    onClick: () => void;
    fill?: boolean;
  }) => (
    <button onClick={onClick} data-fill={fill ? "true" : "false"}>
      {children}
    </button>
  ),
}));

const options = [
  { value: 1, label: "Option A" },
  { value: 2, label: "Option B" },
  { value: 3, label: "Option C" },
];

describe("EnumSelector", () => {
  it("renders all options", () => {
    render(
      <EnumSelector options={options} selected={1} onChange={() => {}} />,
    );
    expect(screen.getByText("Option A")).toBeInTheDocument();
    expect(screen.getByText("Option B")).toBeInTheDocument();
    expect(screen.getByText("Option C")).toBeInTheDocument();
  });

  it("highlights the selected option", () => {
    render(
      <EnumSelector options={options} selected={2} onChange={() => {}} />,
    );
    const optB = screen.getByText("Option B").closest("button");
    expect(optB?.dataset["fill"]).toBe("true");
  });

  it("calls onChange when clicked", () => {
    const onChange = vi.fn();
    render(
      <EnumSelector options={options} selected={1} onChange={onChange} />,
    );
    fireEvent.click(screen.getByText("Option C"));
    expect(onChange).toHaveBeenCalledWith(3);
  });

  it("renders label when provided", () => {
    render(
      <EnumSelector
        options={options}
        selected={1}
        onChange={() => {}}
        label="Test Label"
      />,
    );
    expect(screen.getByText("Test Label")).toBeInTheDocument();
  });
});
