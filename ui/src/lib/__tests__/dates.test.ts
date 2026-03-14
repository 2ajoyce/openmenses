import { describe, it, expect } from "vitest";
import {
  toLocalDate,
  toDateTime,
  fromLocalDate,
  fromDateTime,
  formatDate,
  formatTime,
  formatDateTime,
  nowDateTime,
  daysAgo,
} from "../dates";

describe("toLocalDate / fromLocalDate roundtrip", () => {
  it("converts a Date to LocalDate and back", () => {
    const original = new Date(2026, 2, 15); // March 15, 2026
    const ld = toLocalDate(original);
    expect(ld.value).toBe("2026-03-15");

    const result = fromLocalDate(ld);
    expect(result.getFullYear()).toBe(2026);
    expect(result.getMonth()).toBe(2);
    expect(result.getDate()).toBe(15);
  });

  it("pads single-digit months and days", () => {
    const date = new Date(2026, 0, 5); // Jan 5
    const ld = toLocalDate(date);
    expect(ld.value).toBe("2026-01-05");
  });
});

describe("toDateTime / fromDateTime roundtrip", () => {
  it("converts a Date to DateTime and back", () => {
    const original = new Date("2026-03-15T14:30:00Z");
    const dt = toDateTime(original);
    expect(dt.value).toBe("2026-03-15T14:30:00Z");

    const result = fromDateTime(dt);
    expect(result.getTime()).toBe(original.getTime());
  });
});

describe("formatDate", () => {
  it("formats a LocalDate as human-readable", () => {
    const ld = toLocalDate(new Date(2026, 2, 15));
    const formatted = formatDate(ld);
    expect(formatted).toContain("Mar");
    expect(formatted).toContain("15");
    expect(formatted).toContain("2026");
  });
});

describe("formatTime", () => {
  it("formats a DateTime time portion", () => {
    const dt = toDateTime(new Date("2026-03-15T14:30:00Z"));
    const formatted = formatTime(dt);
    // Output depends on locale/timezone, just verify it's a non-empty string
    expect(formatted.length).toBeGreaterThan(0);
  });
});

describe("formatDateTime", () => {
  it("formats a DateTime as date + time", () => {
    const dt = toDateTime(new Date("2026-03-15T14:30:00Z"));
    const formatted = formatDateTime(dt);
    expect(formatted).toContain("Mar");
    expect(formatted).toContain("15");
    expect(formatted).toContain("2026");
  });
});

describe("nowDateTime", () => {
  it("returns the current time as a DateTime", () => {
    const dt = nowDateTime();
    expect(dt.value).toMatch(
      /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/,
    );
  });
});

describe("daysAgo", () => {
  it("returns a date N days in the past", () => {
    const now = new Date();
    const result = daysAgo(7);
    const diff = now.getTime() - result.getTime();
    const daysDiff = Math.round(diff / (1000 * 60 * 60 * 24));
    expect(daysDiff).toBe(7);
  });
});
