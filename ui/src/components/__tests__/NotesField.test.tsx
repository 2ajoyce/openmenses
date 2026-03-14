import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import React from "react";
import { NotesField } from "../NotesField";

// Mock framework7-react
vi.mock("framework7-react", () => ({
  ListInput: ({
    label,
    value,
    onInput,
    maxlength,
    info,
  }: {
    label: string;
    value: string;
    onInput: (e: { target: { value: string } }) => void;
    maxlength?: number;
    info?: string;
  }) => (
    <div>
      <label>{label}</label>
      <textarea
        value={value}
        onChange={(e) => onInput({ target: { value: e.target.value } })}
        maxLength={maxlength}
        data-testid="notes-textarea"
      />
      {info && <span data-testid="char-count">{info}</span>}
    </div>
  ),
}));

describe("NotesField", () => {
  it("renders with value", () => {
    render(<NotesField value="Hello" onChange={() => {}} />);
    const textarea = screen.getByTestId(
      "notes-textarea",
    ) as HTMLTextAreaElement;
    expect(textarea.value).toBe("Hello");
  });

  it("calls onChange when text changes", () => {
    const onChange = vi.fn();
    render(<NotesField value="" onChange={onChange} />);
    const textarea = screen.getByTestId("notes-textarea");
    fireEvent.change(textarea, { target: { value: "New text" } });
    expect(onChange).toHaveBeenCalledWith("New text");
  });

  it("shows character count", () => {
    render(<NotesField value="Hi" onChange={() => {}} maxLength={100} />);
    expect(screen.getByTestId("char-count").textContent).toBe("2/100");
  });
});
