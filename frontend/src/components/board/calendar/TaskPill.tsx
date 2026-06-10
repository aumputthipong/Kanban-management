"use client";

import { Check, Circle, AlertCircle, Loader2 } from "lucide-react";
import type { Card } from "@/types/board";
import { getAvatarColor } from "@/utils/avatar";
import { getColumnColorHex } from "@/components/board/task-board/ColumnOptionsModal";
import { classifyPillState, type PillState } from "./pillState";

interface TaskPillProps {
  card: Card;
  /** Open the task's detail modal. */
  onSelect: (card: Card) => void;
  /** when true the pill renders flat into a popover/day list (no extra wrap). */
  inPopover?: boolean;
}

// Priority bar colors — see frontend/design.md `priority.*` tokens.
const PRIORITY_BAR: Record<NonNullable<Card["priority"]> | "none", string> = {
  high: "bg-red-600",
  medium: "bg-amber-500",
  low: "bg-emerald-500",
  none: "bg-slate-300",
};

// Per-state background + text classes. Reference design.md state-*-bg/fg.
const STATE_STYLE: Record<PillState, { bg: string; text: string; ring: string }> = {
  todo: { bg: "bg-white", text: "text-slate-800", ring: "ring-slate-200" },
  inProgress: { bg: "bg-blue-100", text: "text-blue-900", ring: "ring-blue-200" },
  done: { bg: "bg-emerald-100", text: "text-emerald-900", ring: "ring-emerald-200" },
  overdue: { bg: "bg-red-100", text: "text-red-900", ring: "ring-red-200" },
};

function StatusIcon({ state }: { state: PillState }) {
  if (state === "done") return <Check size={14} strokeWidth={3} className="text-emerald-700 shrink-0" />;
  if (state === "overdue") return <AlertCircle size={14} className="text-red-700 shrink-0" />;
  if (state === "inProgress") return <Loader2 size={14} className="text-blue-700 shrink-0" />;
  return <Circle size={10} className="text-slate-400 shrink-0" />;
}

function formatDuration(card: Card, state: PillState, daysFromToday: number): string | null {
  if (state === "overdue" && daysFromToday < 0) {
    return `${Math.abs(daysFromToday)}d`;
  }
  if (state === "inProgress" && card.total_subtasks > 0) {
    const pct = Math.round((card.completed_subtasks / card.total_subtasks) * 100);
    return `${pct}%`;
  }
  if (card.estimated_hours) return `${card.estimated_hours}h`;
  return null;
}

export function TaskPill({ card, onSelect, inPopover = false }: TaskPillProps) {
  const state = classifyPillState(card);
  const styles = STATE_STYLE[state];
  const priorityKey = card.priority ?? "none";

  const daysFromToday = card.due_date
    ? Math.round(
        (new Date(card.due_date).setHours(0, 0, 0, 0) -
          new Date().setHours(0, 0, 0, 0)) /
          86400000,
      )
    : 0;
  const duration = formatDuration(card, state, daysFromToday);

  const showProgressBar = state === "inProgress" && card.total_subtasks > 0;
  const progressPct = showProgressBar
    ? Math.round((card.completed_subtasks / card.total_subtasks) * 100)
    : 0;

  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        onSelect(card);
      }}
      title={card.title}
      // 24px tall (size.pill-h) — overflow handled by parent (max 3 + "+N more")
      className={`group relative flex h-6 w-full items-center gap-1.5 overflow-hidden rounded ring-1 ${styles.ring} ${styles.bg} pl-0 pr-1.5 text-left transition-colors hover:bg-indigo-50 ${state === "overdue" ? "font-semibold" : "font-medium"} ${inPopover ? "" : ""}`}
    >
      {/* Priority bar — 3px wide, full height (size.priority-bar-w) */}
      <span
        aria-hidden
        className={`block h-full w-[3px] shrink-0 ${PRIORITY_BAR[priorityKey]}`}
      />

      <StatusIcon state={state} />

      <span className={`flex-1 truncate text-[11px] ${styles.text}`}>
        {card.title}
      </span>

      {duration && (
        <span
          className={`shrink-0 text-[10px] ${state === "overdue" ? "font-bold text-red-700" : "text-slate-500"}`}
        >
          {duration}
        </span>
      )}

      {/* Tag dots — 5px (size.tag-dot) */}
      {card.tags && card.tags.length > 0 && (
        <span className="flex shrink-0 items-center gap-0.5">
          {card.tags.slice(0, 3).map((tag) => (
            <span
              key={tag.id}
              title={tag.name}
              className="h-[5px] w-[5px] rounded-full"
              style={{ backgroundColor: getColumnColorHex(tag.color) ?? "#94a3b8" }}
            />
          ))}
        </span>
      )}

      {/* Avatar — 18px (size.avatar-sm) */}
      {card.assignee_name && card.assignee_id ? (
        <span
          className={`flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full text-[9px] font-bold text-white ${getAvatarColor(card.assignee_id)}`}
        >
          {card.assignee_name.charAt(0).toUpperCase()}
        </span>
      ) : null}

      {/* In-progress progress bar at the bottom of the pill */}
      {showProgressBar && (
        <span
          aria-hidden
          className="absolute bottom-0 left-0 h-[2px] bg-blue-700"
          style={{ width: `${progressPct}%` }}
        />
      )}
    </button>
  );
}
