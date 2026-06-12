"use client";

// SessionFilterChips — single-select chip row above the items list. Five
// buttons cover the meaningful slices of a planning session: all (what's
// still in play), one per type (REQ/DEC/Q), and a "paused" bucket for
// dropped items. Counts come from the parent so the chip can show "(3)"
// without re-deriving from the source list here.
//
// This is intentionally simpler than the calendar's CalendarFilters
// (which uses multi-select dropdowns) — a session is a small flat list
// where one active slice at a time is the more useful interaction.
import type { ReactNode } from "react";
import type { PlanningItemType } from "@/types/planning";

export type SessionFilter = "all" | "req" | "dec" | "q" | "dropped";

const TYPE_BY_FILTER: Record<Exclude<SessionFilter, "all" | "dropped">, PlanningItemType> = {
  req: "REQ",
  dec: "DEC",
  q: "Q",
};

const FILTER_LABELS: Record<SessionFilter, string> = {
  all: "ทั้งหมด",
  req: "สิ่งที่อยากได้",
  dec: "ที่ตกลงแล้ว",
  q: "คำถามค้าง",
  dropped: "พักไว้ก่อน",
};

// Active-state colour per filter. "All" uses the indigo accent (primary
// action); type filters reuse the soft chip palette so the active state
// reads as the same colour family as each row's chip; "dropped" stays
// muted because paused items are deliberately less prominent.
const FILTER_ACTIVE_CLASS: Record<SessionFilter, string> = {
  all: "border-indigo-300 bg-indigo-50 text-indigo-800",
  req: "border-red-300 bg-red-50 text-red-800",
  dec: "border-blue-300 bg-blue-50 text-blue-800",
  q: "border-amber-300 bg-amber-50 text-amber-800",
  dropped: "border-slate-300 bg-slate-100 text-slate-700",
};

const FILTER_ORDER: SessionFilter[] = ["all", "req", "dec", "q", "dropped"];

interface Props {
  active: SessionFilter;
  counts: Record<SessionFilter, number>;
  onChange: (next: SessionFilter) => void;
  /** Right-aligned slot on the same row as the chips (the select-mode controls
   *  live here so the bulk action sits next to the list it acts on). */
  trailing?: ReactNode;
}

export function SessionFilterChips({ active, counts, onChange, trailing }: Props) {
  return (
    <div className="mb-3 flex items-start justify-between gap-3">
      <div className="flex flex-wrap items-center gap-1.5" role="tablist">
        {FILTER_ORDER.map((f) => {
          const isActive = f === active;
          const count = counts[f];
          return (
            <button
              key={f}
              type="button"
              role="tab"
              aria-selected={isActive}
              onClick={() => onChange(f)}
              className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
                isActive
                  ? FILTER_ACTIVE_CLASS[f]
                  : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
              }`}
            >
              {FILTER_LABELS[f]}
              <span
                className={`rounded-full px-1.5 text-[10px] font-bold ${
                  isActive
                    ? "bg-white/70 text-slate-700"
                    : "bg-slate-100 text-slate-500"
                }`}
              >
                {count}
              </span>
            </button>
          );
        })}
      </div>
      {trailing && <div className="shrink-0">{trailing}</div>}
    </div>
  );
}

// applySessionFilter is the single source of truth for which items the
// chips surface. Kept beside the chip component so the labels and the
// predicates stay in sync — change the contract here and the chips
// inherit it automatically.
export function applySessionFilter<T extends { type: PlanningItemType; status: string }>(
  items: T[],
  filter: SessionFilter,
): T[] {
  if (filter === "dropped") return items.filter((it) => it.status === "dropped");
  // Non-dropped buckets always exclude dropped items — those are the
  // "paused" pile and have their own chip.
  const visible = items.filter((it) => it.status !== "dropped");
  if (filter === "all") return visible;
  return visible.filter((it) => it.type === TYPE_BY_FILTER[filter]);
}
