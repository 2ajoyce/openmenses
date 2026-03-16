import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { groupPhaseEstimates } from "../groupPhaseEstimates";
import { CyclePhase, ConfidenceLevel, PhaseEstimateSchema } from "@gen/openmenses/v1/model_pb";
import { LocalDateSchema } from "@gen/openmenses/v1/types_pb";

describe("groupPhaseEstimates", () => {
  it("returns empty array for empty input", () => {
    const result = groupPhaseEstimates([]);
    expect(result).toEqual([]);
  });

  it("returns single group for single estimate", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
    ];

    const result = groupPhaseEstimates(estimates);

    expect(result).toHaveLength(1);
    expect(result[0]).toHaveLength(1);
    expect(result[0]?.[0]).toBe(estimates[0]);
  });

  it("groups consecutive same-phase estimates", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-2",
        date: create(LocalDateSchema, { value: "2026-03-02" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-3",
        date: create(LocalDateSchema, { value: "2026-03-03" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
    ];

    const result = groupPhaseEstimates(estimates);

    expect(result).toHaveLength(1);
    expect(result[0]).toHaveLength(3);
  });

  it("separates different phases into different groups", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-2",
        date: create(LocalDateSchema, { value: "2026-03-02" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-3",
        date: create(LocalDateSchema, { value: "2026-03-03" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-4",
        date: create(LocalDateSchema, { value: "2026-03-04" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
    ];

    const result = groupPhaseEstimates(estimates);

    expect(result).toHaveLength(2);
    expect(result[0]).toHaveLength(2);
    expect(result[1]).toHaveLength(2);
  });

  it("handles multiple phase transitions", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-2",
        date: create(LocalDateSchema, { value: "2026-03-02" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-3",
        date: create(LocalDateSchema, { value: "2026-03-03" }),
        phase: CyclePhase.FOLLICULAR,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-4",
        date: create(LocalDateSchema, { value: "2026-03-04" }),
        phase: CyclePhase.OVULATION_WINDOW,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-5",
        date: create(LocalDateSchema, { value: "2026-03-05" }),
        phase: CyclePhase.LUTEAL,
        confidence: ConfidenceLevel.MEDIUM,
        userId: "users/default",
      }),
    ];

    const result = groupPhaseEstimates(estimates);

    expect(result).toHaveLength(4);
    expect(result[0]).toHaveLength(1); // MENSTRUATION
    expect(result[1]).toHaveLength(2); // FOLLICULAR
    expect(result[2]).toHaveLength(1); // OVULATION_WINDOW
    expect(result[3]).toHaveLength(1); // LUTEAL
  });

  it("preserves order of estimates within groups", () => {
    const estimates = [
      create(PhaseEstimateSchema, {
        name: "estimate-1",
        date: create(LocalDateSchema, { value: "2026-03-01" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
      create(PhaseEstimateSchema, {
        name: "estimate-2",
        date: create(LocalDateSchema, { value: "2026-03-02" }),
        phase: CyclePhase.MENSTRUATION,
        confidence: ConfidenceLevel.HIGH,
        userId: "users/default",
      }),
    ];

    const result = groupPhaseEstimates(estimates);

    expect(result[0]?.[0]?.name).toBe("estimate-1");
    expect(result[0]?.[1]?.name).toBe("estimate-2");
  });
});
