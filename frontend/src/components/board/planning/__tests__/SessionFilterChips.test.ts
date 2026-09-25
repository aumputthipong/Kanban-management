import { describe, it, expect } from "vitest";
import { applySessionFilter } from "../SessionFilterChips";
import type { PlanningItemType } from "@/types/planning";

type T = { type: PlanningItemType; status: string };

const items: T[] = [
  { type: "REQ", status: "live" },
  { type: "REQ", status: "selected" },
  { type: "REQ", status: "dropped" },
  { type: "DEC", status: "live" },
  { type: "DEC", status: "promoted" },
  { type: "Q", status: "live" },
  { type: "Q", status: "dropped" },
];

describe("applySessionFilter", () => {
  it("all surfaces every non-dropped item (live, selected, promoted)", () => {
    // Promoted still counts as "in play" — only dropped items are hidden.
    const out = applySessionFilter(items, "all");
    expect(out).toHaveLength(5);
    expect(out.every((it) => it.status !== "dropped")).toBe(true);
  });

  it("type filters return only that type and exclude dropped", () => {
    expect(applySessionFilter(items, "req")).toEqual([
      { type: "REQ", status: "live" },
      { type: "REQ", status: "selected" },
    ]);
    expect(applySessionFilter(items, "dec")).toEqual([
      { type: "DEC", status: "live" },
      { type: "DEC", status: "promoted" },
    ]);
    expect(applySessionFilter(items, "q")).toEqual([
      { type: "Q", status: "live" },
    ]);
  });

  it("dropped surfaces only dropped items (any type)", () => {
    const out = applySessionFilter(items, "dropped");
    expect(out).toHaveLength(2);
    expect(out.every((it) => it.status === "dropped")).toBe(true);
  });

  it("empty input returns empty for every filter", () => {
    for (const f of ["all", "req", "dec", "q", "dropped"] as const) {
      expect(applySessionFilter([], f)).toEqual([]);
    }
  });
});
