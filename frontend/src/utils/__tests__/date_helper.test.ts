import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { dateKey, formatRelativeDueDate, formatThaiDate } from "@/utils/date_helper";

const NOW = new Date(2026, 9, 7, 15, 30);

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
});

afterEach(() => {
  vi.useRealTimers();
});

const daysFromNow = (n: number) => dateKey(new Date(2026, 9, 7 + n));

describe("formatRelativeDueDate", () => {
  it.each([
    [0, "Today"],
    [1, "Tomorrow"],
    [2, "In 2 days"],
    [7, "In 7 days"],
    [-1, "1 day overdue"],
    [-2, "2 days overdue"],
  ])("offset %i days -> %s", (offset, expected) => {
    expect(formatRelativeDueDate(daysFromNow(offset))).toBe(expected);
  });

  it("switches to an absolute Thai date after 7 days", () => {
    const due = daysFromNow(8);
    expect(formatRelativeDueDate(due)).toBe(formatThaiDate(due));
  });

  it("ignores the time of day on a full ISO timestamp", () => {
    const lateTonight = new Date(2026, 9, 7, 23, 59).toISOString();
    expect(formatRelativeDueDate(lateTonight)).toBe("Today");
  });

  it("crosses a month boundary", () => {
    vi.setSystemTime(new Date(2026, 9, 31, 9, 0));
    expect(formatRelativeDueDate("2026-11-01")).toBe("Tomorrow");
  });

  it.each(["", "not-a-date", "2026-13-45"])("returns empty string for invalid input %j", (input) => {
    expect(formatRelativeDueDate(input)).toBe("");
  });
});

describe("formatThaiDate", () => {
  it("formats in the Buddhist calendar", () => {
    expect(formatThaiDate("2026-10-07")).toContain("2569");
  });

  it.each(["", "not-a-date"])("returns empty string for invalid input %j", (input) => {
    expect(formatThaiDate(input)).toBe("");
  });
});

describe("dateKey", () => {
  it("zero-pads month and day", () => {
    expect(dateKey(new Date(2026, 0, 5))).toBe("2026-01-05");
  });

  it("uses the local calendar day, not UTC", () => {
    expect(dateKey(new Date(2026, 9, 7, 0, 30))).toBe("2026-10-07");
  });

  it("handles a leap day", () => {
    expect(dateKey(new Date(2028, 1, 29))).toBe("2028-02-29");
  });
});
