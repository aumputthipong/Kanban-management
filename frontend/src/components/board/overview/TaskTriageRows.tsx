"use client";

import { AlarmClock, Calendar } from "lucide-react";
import type { Card } from "@/types/board";
import { getAvatarColor } from "@/utils/avatar";

// Priority signal lives ONLY on the 3px left rail (design.md rule). Colours
// mirror the existing UrgentTaskRow stripe so a card looks the same everywhere.
const RAIL: Record<"high" | "medium" | "low" | "none", string> = {
  high: "bg-rose-400",
  medium: "bg-amber-400",
  low: "bg-emerald-400",
  none: "bg-slate-300",
};

const PRIORITY_DOT: Record<"high" | "medium" | "low", string> = {
  high: "bg-rose-500",
  medium: "bg-amber-500",
  low: "bg-emerald-500",
};

/** Whole-day distance between a due date and today-at-midnight (local). */
export function daysOverdue(dueDateStr: string): number {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const due = new Date(dueDateStr);
  due.setHours(0, 0, 0, 0);
  return Math.max(0, Math.ceil((today.getTime() - due.getTime()) / 86400000));
}

// Severity reads from ORDER + ink darkness, never a colour wash — only the
// single critical hero keeps red.
function severityInk(days: number): string {
  if (days >= 20) return "text-slate-900";
  if (days >= 8) return "text-slate-600";
  return "text-slate-400";
}

function AssigneeAvatar({ card }: { card: Card }) {
  if (card.assignee_id && card.assignee_name) {
    return (
      <div
        title={card.assignee_name}
        className={`w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold text-white shrink-0 ${getAvatarColor(card.assignee_id)}`}
      >
        {card.assignee_name.trim().charAt(0).toUpperCase()}
      </div>
    );
  }
  return (
    <div className="w-5 h-5 rounded-full flex items-center justify-center text-[10px] font-bold bg-slate-200 text-slate-400 shrink-0">
      ?
    </div>
  );
}

function StatusMeta({ columnTitle, card }: { columnTitle: string; card: Card }) {
  return (
    <div className="flex items-center gap-2.5 mt-1 text-[11px] text-slate-500">
      {columnTitle && (
        <span className="inline-flex items-center gap-1.5">
          <span
            aria-hidden
            className={`w-[7px] h-[7px] rounded-sm ${card.is_done ? "bg-emerald-500" : "bg-slate-400"}`}
          />
          {columnTitle}
        </span>
      )}
      {card.estimated_hours != null && (
        <span className="text-slate-400">~{card.estimated_hours}h เหลือ</span>
      )}
    </div>
  );
}

/**
 * The single most-overdue card — the only place red survives in the Tasks
 * triage view. Renders as one big clickable banner.
 */
export function CriticalHero({
  card,
  columnTitle,
  onSelect,
}: {
  card: Card;
  columnTitle: string;
  onSelect: (card: Card) => void;
}) {
  const days = card.due_date ? daysOverdue(card.due_date) : 0;
  return (
    <button
      type="button"
      onClick={() => onSelect(card)}
      className="group relative w-full flex items-center gap-4 mb-6 rounded-xl border border-slate-200 bg-white p-4 text-left overflow-hidden shadow-sm hover:bg-slate-50 transition-colors"
    >
      <span aria-hidden className="absolute inset-y-0 left-0 w-1 bg-rose-500" />
      <span className="w-10 h-10 rounded-lg bg-rose-50 text-rose-600 flex items-center justify-center shrink-0">
        <AlarmClock size={20} />
      </span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="inline-flex items-center rounded bg-rose-50 text-rose-700 border border-rose-200 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider">
            เร่งด่วนที่สุด
          </span>
          <span className="text-[11px] font-medium text-slate-400">จัดการก่อน</span>
        </div>
        <p className="text-base font-bold text-slate-900 truncate mt-1">
          {card.title}
        </p>
        <div className="flex items-center gap-3.5 mt-1.5 text-xs text-slate-600 flex-wrap">
          {card.priority && (
            <span className="inline-flex items-center gap-1.5">
              <span
                aria-hidden
                className={`w-2 h-2 rounded-full ${PRIORITY_DOT[card.priority]}`}
              />
              {card.priority} priority
            </span>
          )}
          {columnTitle && (
            <span className="inline-flex items-center gap-1.5">
              <span aria-hidden className="w-[7px] h-[7px] rounded-sm bg-slate-400" />
              {columnTitle}
            </span>
          )}
          {card.assignee_id && card.assignee_name && (
            <span className="inline-flex items-center gap-1.5">
              <AssigneeAvatar card={card} />
              {card.assignee_name}
            </span>
          )}
          {card.estimated_hours != null && <span>~{card.estimated_hours}h เหลือ</span>}
        </div>
      </div>
      <div className="flex flex-col items-end gap-2 shrink-0">
        <span className="inline-flex items-center gap-1.5 rounded-md bg-rose-50 text-rose-700 border border-rose-200 px-2 py-1 text-xs font-semibold whitespace-nowrap tabular-nums">
          <Calendar size={12} />
          เลย {days} วัน
        </span>
        <span className="inline-flex items-center gap-1 px-3 h-8 rounded bg-blue-700 text-white text-xs font-semibold group-hover:bg-blue-800 transition-colors">
          เปิดงาน →
        </span>
      </div>
    </button>
  );
}

/** A ranked overdue row — neutral days gutter, priority on the rail only. */
export function RankedRow({
  card,
  columnTitle,
  onSelect,
}: {
  card: Card;
  columnTitle: string;
  onSelect: (card: Card) => void;
}) {
  const days = card.due_date ? daysOverdue(card.due_date) : 0;
  return (
    <button
      type="button"
      onClick={() => onSelect(card)}
      className="group grid grid-cols-[3px_46px_1fr_auto] items-center gap-x-3.5 w-full py-2.5 pr-3 border-b border-slate-100 last:border-b-0 hover:bg-slate-50 text-left transition-colors"
    >
      <span aria-hidden className={`self-stretch w-[3px] rounded-r ${RAIL[card.priority ?? "none"]}`} />
      <div className="flex flex-col items-center justify-center leading-none">
        <span className={`text-base font-bold tabular-nums ${severityInk(days)}`}>{days}</span>
        <span className="text-[9px] font-semibold uppercase tracking-wide text-slate-400 mt-0.5">วัน</span>
      </div>
      <div className="min-w-0">
        <p className="text-sm font-semibold text-slate-900 truncate">{card.title}</p>
        <StatusMeta columnTitle={columnTitle} card={card} />
      </div>
      <div className="flex items-center pl-2 shrink-0">
        <AssigneeAvatar card={card} />
      </div>
    </button>
  );
}

/** An upcoming (not-yet-overdue) row — the days gutter goes blue, not red. */
export function UpcomingRow({
  card,
  columnTitle,
  whenLabel,
  onSelect,
}: {
  card: Card;
  columnTitle: string;
  whenLabel: string;
  onSelect: (card: Card) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(card)}
      className="group grid grid-cols-[3px_46px_1fr_auto] items-center gap-x-3.5 w-full py-2.5 pr-3 border-b border-slate-100 last:border-b-0 hover:bg-slate-50 text-left transition-colors"
    >
      <span aria-hidden className={`self-stretch w-[3px] rounded-r ${RAIL[card.priority ?? "none"]}`} />
      <div className="flex flex-col items-center justify-center leading-none">
        <span className="text-xs font-bold text-primary text-center">{whenLabel}</span>
      </div>
      <div className="min-w-0">
        <p className="text-sm font-semibold text-slate-900 truncate">{card.title}</p>
        <StatusMeta columnTitle={columnTitle} card={card} />
      </div>
      <div className="flex items-center pl-2 shrink-0">
        <AssigneeAvatar card={card} />
      </div>
    </button>
  );
}

/** Collapsed one-line "nothing due" bucket. */
export function CollapsedEmpty({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-2.5 px-3.5 py-2.5 rounded border border-dashed border-slate-200 bg-slate-50 text-xs font-semibold text-slate-400">
      <span aria-hidden className="w-[7px] h-[7px] rounded-full bg-slate-300" />
      {label} · <span className="text-slate-500">ไม่มีงานครบกำหนด</span>
    </div>
  );
}
