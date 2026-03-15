import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import React from "react";
import { ConfirmDialog } from "../ConfirmDialog";

vi.mock("framework7-react");

describe("ConfirmDialog", () => {
  const onConfirm = vi.fn();
  const onCancel = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls f7.dialog.confirm when open is true", async () => {
    const { f7 } = await import("framework7-react");
    const confirmSpy = vi.spyOn(f7.dialog, "confirm");

    render(
      <ConfirmDialog
        open={true}
        title="Delete"
        message="Are you sure?"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    expect(confirmSpy).toHaveBeenCalledWith(
      "Are you sure?",
      "Delete",
      expect.any(Function),
      expect.any(Function),
    );
  });

  it("does not call f7.dialog.confirm when open is false", async () => {
    const { f7 } = await import("framework7-react");
    const confirmSpy = vi.spyOn(f7.dialog, "confirm");

    render(
      <ConfirmDialog
        open={false}
        title="Delete"
        message="Are you sure?"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    expect(confirmSpy).not.toHaveBeenCalled();
  });

  it("calls onConfirm when confirm callback is invoked", async () => {
    const { f7 } = await import("framework7-react");
    const confirmSpy = vi.spyOn(f7.dialog, "confirm");

    render(
      <ConfirmDialog
        open={true}
        title="Delete"
        message="Are you sure?"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    // Get the confirm callback (3rd argument: message, title, onConfirm, onCancel)
    const args = confirmSpy.mock.calls[0] as unknown as unknown[];
    const confirmCallback = args[2] as () => void;
    confirmCallback();

    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when cancel callback is invoked", async () => {
    const { f7 } = await import("framework7-react");
    const confirmSpy = vi.spyOn(f7.dialog, "confirm");

    render(
      <ConfirmDialog
        open={true}
        title="Delete"
        message="Are you sure?"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />,
    );

    // Get the cancel callback (4th argument: message, title, onConfirm, onCancel)
    const args = confirmSpy.mock.calls[0] as unknown as unknown[];
    const cancelCallback = args[3] as () => void;
    cancelCallback();

    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
