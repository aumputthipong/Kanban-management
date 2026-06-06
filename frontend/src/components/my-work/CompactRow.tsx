"use client";

import Link from "next/link";
import { Calendar, Check, Clock } from "lucide-react";
import { formatRelativeDueDate, formatThaiDate } from "@/utils/date_helper";
import type { MyWorkCard, MyWorkGroup, MyWorkStatus } from "@/types/myWork";
import { SnoozeMenu } from "./SnoozeMenu";

interface CompactRowProps {
  card: MyWorkCard;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string) => void;
  /** Drop the due/estimate columns (used by the "no date" panel). */
  slim?: boolean;
  /** Taller, slightly larger title — used inside the Today hero panel. */
  hero?: boolean;
}

// Deterministic project dot color — boards have no stored color, so we hash the
// id onto a fixed palette that mirrors the sidebar project dots.
const BOARD_PALETTE = [
  "#1E40AF",
  "#0EA5A4",
  "#7C3AED",
  "#F59E0B",
  "#EC4899",
  "#10B981",
  "#3B82F6",
  "#F97316",
];

function boardColor(id: string): string {
  let hash = 0;
  for (let i = 0; i < id.length; i++) hash = (hash * 31 + id.charCodeAt(i)) >>> 0;
  return BOARD_PALETTE[hash % BOARD_PALETTE.length];
}

function statusDot(status: MyWorkStatus): string {
  switch (status) {
    case "in_progress":
      return "bg-blue-700";
    case "done":
      return "bg-emerald-700";
    default:
      return "bg-slate-400";
  }
}

function overdueDays(due: string): number {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const d = new Date(due);
  d.setHours(0, 0, 0, 0);
  return Math.max(1, Math.round((today.getTime() - d.getTime()) / 86_400_000));
}

// Thai-first due label. Today + overdue are special-cased so the dense rows
// read in Thai ("วันนี้" / "เลย N วัน"); future dates fall back to the shared
// helper used elsewhere in the app.
function dueText(card: MyWorkCard): string {
  if (card.group === "today") return "วันนี้";
  if (!card.due_date) return "";
  if (card.group === "overdue") return `เลย ${overdueDays(card.due_date)} วัน`;
  return formatRelativeDueDate(card.due_date);
}

function dueClass(group: MyWorkGroup): string {
  switch (group) {
    case "overdue":
      return "bg-red-50 text-red-700 border border-red-200 font-semibold";
    case "today":
      return "bg-blue-50 text-blue-700 border border-blue-200 font-semibold";
    case "this_week":
      return "text-slate-600";
    default:
      return "text-slate-500";
  }
}

const PRI_BAR: Record<NonNullable<MyWorkCard["priority"]> | "none", string> = {
  high: "bg-red-600",
  medium: "bg-amber-500",
  low: "bg-emerald-500",
  none: "bg-slate-400",
};

export function CompactRow({ card, onComplete, onSnooze, slim = false, hero = false }: CompactRowProps) {
  const handleComplete = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onComplete(card.id);
  };

  return (
    <Link
      href={`/board/${card.board_id}/tasks`}
      className={`group relative flex items-center gap-3 pl-[18px] pr-3 border-b border-slate-100 last:border-b-0 hover:bg-slate-50 transition-colors ${hero ? "h-[52px]" : "h-12"}`}
    >
      <span
        aria-hidden
        className={`absolute left-0 top-[9px] bottom-[9px] w-[3px] rounded-r-sm ${PRI_BAR[card.priority ?? "none"]}`}
      />

      <button
        type="button"
        onClick={handleComplete}
        className="w-[18px] h-[18px] shrink-0 rounded-sm border-[1.75px] border-slate-300 flex items-center justify-center text-transparent hover:border-blue-700 hover:bg-blue-50 hover:text-blue-700 transition-colors"
        aria-label="ทำเครื่องหมายว่าเสร็จแล้ว"
      >
        <Check size={11} strokeWidth={3.2} className="opacity-0 group-hover:opacity-100 transition-opacity" />
      </button>

      <div className="flex items-center gap-2.5 min-w-0 flex-1">
        <span className={`font-semibold text-slate-900 truncate min-w-0 ${hero ? "text-sm" : "text-[13.5px]"}`}>
          {card.title}
        </span>
        <span className="flex items-center gap-2 shrink-0">
          <span
            aria-hidden
            title={card.board_name}
            className="w-2 h-2 rounded-full shrink-0"
            style={{ background: boardColor(card.board_id) }}
          />
          {card.column_name && (
            <span className="hidden md:inline-flex items-center gap-1.5 text-[11.5px] font-medium text-slate-500 whitespace-nowrap">
              <span aria-hidden className={`w-1.5 h-1.5 rounded-full shrink-0 ${statusDot(card.status)}`} />
              {card.column_name}
            </span>
          )}
        </span>
      </div>

      {!slim && (
        <>
          <div className="w-28 flex justify-end shrink-0">
            {card.due_date && (
              <span
                className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-sm text-xs whitespace-nowrap ${dueClass(card.group)}`}
                title={formatThaiDate(card.due_date)}
              >
                <Calendar size={12} />
                {dueText(card)}
              </span>
            )}
          </div>
          <div className="hidden lg:flex w-9 items-center justify-end gap-1 text-xs font-semibold text-slate-400 shrink-0">
            {card.estimated_hours != null && (
              <>
                <Clock size={12} />
                {card.estimated_hours}h
              </>
            )}
          </div>
        </>
      )}

      <div className="w-6 flex justify-end shrink-0 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
        <SnoozeMenu onSnooze={(dueDate) => onSnooze(card.id, dueDate)} />
      </div>
    </Link>
  );
}
