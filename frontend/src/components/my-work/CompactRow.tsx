"use client";

import { Check, Clock, ListChecks } from "lucide-react";
import { formatRelativeDueDate, formatThaiDate } from "@/utils/date_helper";
import type { MyWorkCard, MyWorkStatus } from "@/types/myWork";

interface CompactRowProps {
  card: MyWorkCard;
  /** Open the task detail modal — where done / snooze live. */
  onOpenCard: (card: MyWorkCard) => void;
  /** Project accent for the inline project dot (list variant). */
  projectColor?: string;
  /** Renders the tick-off checkbox (list variant). */
  onComplete?: (cardId: string) => void;
  /** Light tree row for the side rail: title + one right-aligned due/status text. */
  rail?: boolean;
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

function dueText(card: MyWorkCard): string {
  if (card.group === "today") return "วันนี้";
  if (!card.due_date) return "";
  if (card.group === "overdue") return `เลย ${overdueDays(card.due_date)} วัน`;
  return formatRelativeDueDate(card.due_date);
}

function railMeta(card: MyWorkCard): { text: string; className: string } {
  if (card.due_date) {
    const tone =
      card.group === "overdue" ? "text-red-700" : card.group === "today" ? "text-blue-700" : "text-slate-500";
    return { text: dueText(card), className: tone };
  }
  const tone =
    card.status === "in_progress" ? "text-blue-700" : card.status === "done" ? "text-emerald-700" : "text-slate-400";
  return { text: card.column_name, className: tone };
}

// Readable label + a small `priority.*` disc (design.md allows the disc inside a priority chip).
const PRIORITY: Record<NonNullable<MyWorkCard["priority"]>, { label: string; disc: string }> = {
  high: { label: "High", disc: "bg-red-600" },
  medium: { label: "Medium", disc: "bg-amber-500" },
  low: { label: "Low", disc: "bg-emerald-500" },
};

const FOCUS_RING = "focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-blue-600";

export function CompactRow({ card, onOpenCard, projectColor, onComplete, rail = false }: CompactRowProps) {
  const open = () => onOpenCard(card);
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      open();
    }
  };

  if (rail) {
    const meta = railMeta(card);
    return (
      <div
        role="button"
        tabIndex={0}
        onClick={open}
        onKeyDown={onKeyDown}
        title={card.due_date ? formatThaiDate(card.due_date) : undefined}
        className={`flex items-center gap-3 h-8 pl-3 pr-2 rounded-md cursor-pointer hover:bg-slate-50 transition-colors ${FOCUS_RING}`}
      >
        <span className="flex-1 min-w-0 truncate text-[13px] text-slate-700">{card.title}</span>
        {meta.text && (
          <span className={`shrink-0 text-xs font-medium whitespace-nowrap ${meta.className}`}>{meta.text}</span>
        )}
      </div>
    );
  }

  const total = card.total_subtasks ?? 0;
  const done = card.completed_subtasks ?? 0;
  // Every row in the Today note is due today, so the label would only repeat the heading.
  const showDue = card.due_date != null && card.group !== "today";
  const priority = card.priority ? PRIORITY[card.priority] : null;

  return (
    <div className="flex items-center gap-3 pl-3 pr-2 rounded-md hover:bg-slate-50 transition-colors">
      {onComplete && (
        <button
          type="button"
          onClick={() => onComplete(card.id)}
          aria-label={`ทำเสร็จ: ${card.title}`}
          title="ทำเสร็จ"
          className={`grid place-items-center w-5 h-5 shrink-0 rounded-full border-2 border-slate-300 text-transparent hover:border-emerald-700 hover:text-emerald-700 transition-colors ${FOCUS_RING}`}
        >
          <Check size={11} strokeWidth={3} />
        </button>
      )}
      <button
        type="button"
        onClick={open}
        className={`flex min-w-0 flex-1 items-center gap-4 py-2 text-left rounded-sm ${FOCUS_RING}`}
      >
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-semibold text-slate-900">{card.title}</div>
        <div className="mt-0.5 flex items-center gap-3 min-w-0 text-xs text-slate-500">
          {/* Omitted inside a project group, whose header already names the project. */}
          {projectColor && (
            <span className="inline-flex items-center gap-1.5 min-w-0">
              <span aria-hidden className="w-2 h-2 rounded-full shrink-0" style={{ background: projectColor }} />
              <span className="truncate">{card.board_name}</span>
            </span>
          )}
          {card.column_name && (
            <span className="inline-flex items-center gap-1.5 shrink-0 whitespace-nowrap">
              <span aria-hidden className={`w-1.5 h-1.5 rounded-full ${statusDot(card.status)}`} />
              {card.column_name}
            </span>
          )}
          {total > 0 && (
            <span
              className={`inline-flex items-center gap-1 shrink-0 tabular-nums ${done === total ? "text-emerald-700" : ""}`}
              title={`งานย่อย ${done}/${total}`}
            >
              <ListChecks size={12} />
              {done}/{total}
            </span>
          )}
        </div>
      </div>

      {showDue && card.due_date && (
        <span
          title={formatThaiDate(card.due_date)}
          className={`w-20 shrink-0 text-right text-xs font-medium whitespace-nowrap ${card.group === "overdue" ? "text-red-700" : "text-slate-500"}`}
        >
          {dueText(card)}
        </span>
      )}
      {/* Fixed-width columns keep priority and estimate aligned down the list for scanning. */}
      <span className="w-20 shrink-0">
        {priority && (
          <span className="inline-flex items-center gap-1.5 rounded-full border border-slate-200 px-2 py-0.5 text-xs font-medium text-slate-700">
            <span aria-hidden className={`w-2 h-2 rounded-full ${priority.disc}`} />
            {priority.label}
          </span>
        )}
      </span>
      <span className="hidden lg:inline-flex w-12 shrink-0 items-center justify-end gap-1 text-xs text-slate-400 tabular-nums">
        {card.estimated_hours ? (
          <>
            <Clock size={12} />
            {card.estimated_hours}h
          </>
        ) : null}
      </span>
      </button>
    </div>
  );
}
