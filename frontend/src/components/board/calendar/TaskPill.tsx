"use client";

import { useState, useRef, useEffect } from "react";
import dynamic from "next/dynamic";
import { Check, Circle, AlertCircle, Loader2 } from "lucide-react";
import type { Card } from "@/types/board";
import type { FormState } from "@/components/board/card-modal/CardDetailModal";
import { useBoardActions } from "@/hooks/useBoardActions";
import { useCanEdit } from "@/hooks/useCanEdit";
import { getAvatarColor } from "@/utils/avatar";
import { getColumnColorHex } from "@/components/board/task-board/ColumnOptionsModal";
import { classifyPillState, type PillState } from "./pillState";
import { CardPreviewPopover } from "./CardPreviewPopover";

const CardDetailModal = dynamic(
  () =>
    import("@/components/board/card-modal/CardDetailModal").then((m) => ({
      default: m.CardDetailModal,
    })),
  { ssr: false },
);

interface TaskPillProps {
  card: Card;
  boardId: string;
  /** when true the pill renders flat into a popover list (no priority bar wrap, no hover popover) */
  inPopover?: boolean;
}

// Priority bar colors — see frontend/design.md `priority.*` tokens.
// Stored here as Tailwind classes whose hex values match the design.md spec
// exactly (no raw bg-[#...] per the no-hardcoded-color rule).
const PRIORITY_BAR: Record<NonNullable<Card["priority"]> | "none", string> = {
  high: "bg-red-600",         // priority.high #DC2626
  medium: "bg-amber-500",     // priority.medium #F59E0B
  low: "bg-emerald-500",      // priority.low #10B981
  none: "bg-slate-300",       // priority.none
};

// Per-state background + text classes. Reference design.md state-*-bg/fg.
// Hex equivalents:
//   todo        → surface white
//   inProgress  → state-progress-bg #DBEAFE = blue-100, fg #1D4ED8 = blue-700
//   done        → state-done-bg #D1FAE5 = emerald-100, fg #047857 = emerald-700
//   overdue     → state-overdue-bg #FEE2E2 = red-100, fg #B91C1C = red-700
const STATE_STYLE: Record<PillState, { bg: string; text: string; ring: string }> = {
  todo:       { bg: "bg-white",         text: "text-slate-800", ring: "ring-slate-200" },
  inProgress: { bg: "bg-blue-100",      text: "text-blue-900",  ring: "ring-blue-200" },
  done:       { bg: "bg-emerald-100",   text: "text-emerald-900", ring: "ring-emerald-200" },
  overdue:    { bg: "bg-red-100",       text: "text-red-900",   ring: "ring-red-200" },
};

function StatusIcon({ state }: { state: PillState }) {
  // 14px = size.status-icon
  if (state === "done") return <Check size={14} strokeWidth={3} className="text-emerald-700 shrink-0" />;
  if (state === "overdue") return <AlertCircle size={14} className="text-red-700 shrink-0" />;
  if (state === "inProgress") return <Loader2 size={14} className="text-blue-700 shrink-0" />;
  return <Circle size={10} className="text-slate-400 shrink-0" />;
}

function formatDuration(card: Card, state: PillState, daysFromToday: number): string | null {
  // For overdue, show how many days late (ref2: "Xd")
  if (state === "overdue" && daysFromToday < 0) {
    return `${Math.abs(daysFromToday)}d`;
  }
  // For in-progress with subtasks, show AC progress
  if (state === "inProgress" && card.total_subtasks > 0) {
    const pct = Math.round((card.completed_subtasks / card.total_subtasks) * 100);
    return `${pct}%`;
  }
  // Otherwise estimated hours if set
  if (card.estimated_hours) return `${card.estimated_hours}h`;
  return null;
}

export function TaskPill({ card, boardId, inPopover = false }: TaskPillProps) {
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const [isHovering, setIsHovering] = useState(false);
  const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Delayed-close timer. Without this, leaving the pill to enter the popover
  // would close the popover before the pointer arrived (the popover is
  // portalled to <body>, so it's not part of the pill's hover region).
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Anchor element for the popover. Stored in state (not a ref) so we can
  // pass it as a prop without reading .current during render — React 19's
  // react-hooks/refs rule forbids that. A callback ref captures the DOM
  // node when it mounts and triggers a re-render so the popover sees it.
  const [pillEl, setPillEl] = useState<HTMLButtonElement | null>(null);

  const { handleAddSubtask, handleDeleteCard, handleUpdateCard } =
    useBoardActions(boardId);
  const canEdit = useCanEdit(card);

  const state = classifyPillState(card);
  const styles = STATE_STYLE[state];
  const priorityKey = card.priority ?? "none";

  // Days from today for "Xd" overdue display.
  const daysFromToday = card.due_date
    ? Math.round(
        (new Date(card.due_date).setHours(0, 0, 0, 0) -
          new Date().setHours(0, 0, 0, 0)) /
          86400000,
      )
    : 0;
  const duration = formatDuration(card, state, daysFromToday);

  const onPointerEnter = () => {
    if (inPopover) return;
    // Re-entering the pill cancels any pending close from a previous leave.
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    if (hoverTimerRef.current) clearTimeout(hoverTimerRef.current);
    hoverTimerRef.current = setTimeout(() => setIsHovering(true), 300);
  };
  const onPointerLeave = () => {
    if (hoverTimerRef.current) clearTimeout(hoverTimerRef.current);
    // 150ms grace so the user has time to move the pointer onto the popover
    // before it closes. The popover's onMouseEnter cancels this timer.
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    closeTimerRef.current = setTimeout(() => setIsHovering(false), 150);
  };

  // Popover hover handlers — keep the popover open as long as the pointer is
  // anywhere on it, and start the same 150ms close grace when the pointer
  // leaves. This lets the user click "Open card" or scroll the AC list.
  const onPopoverEnter = () => {
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
  };
  const onPopoverLeave = () => {
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    closeTimerRef.current = setTimeout(() => setIsHovering(false), 150);
  };

  useEffect(() => {
    return () => {
      if (hoverTimerRef.current) clearTimeout(hoverTimerRef.current);
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  const onSave = (cardId: string, form: FormState) => {
    handleUpdateCard(cardId, {
      title: form.title,
      description: form.description,
      due_date: form.due_date,
      assignee_id: form.assignee_id,
      priority: form.priority,
      estimated_hours: form.estimated_hours,
      tags: form.tags,
      acceptance_criteria: form.acceptance_criteria,
      implementation_note: form.implementation_note,
    });
  };

  const onDelete = (cardId: string) => {
    handleDeleteCard(cardId);
    setIsDetailOpen(false);
  };

  // Progress bar (in-progress state only)
  const showProgressBar =
    state === "inProgress" && card.total_subtasks > 0;
  const progressPct = showProgressBar
    ? Math.round((card.completed_subtasks / card.total_subtasks) * 100)
    : 0;

  return (
    <>
      <button
        ref={setPillEl}
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setIsDetailOpen(true);
        }}
        onPointerEnter={onPointerEnter}
        onPointerLeave={onPointerLeave}
        onFocus={onPointerEnter}
        onBlur={onPointerLeave}
        title={card.title}
        // 24px tall (size.pill-h) — overflow handled by parent (max 3 + "+N more")
        className={`group relative flex h-6 w-full items-center gap-1.5 overflow-hidden rounded ring-1 ${styles.ring} ${styles.bg} pl-0 pr-1.5 text-left transition-colors hover:bg-indigo-50 ${state === "overdue" ? "font-semibold" : "font-medium"}`}
      >
        {/* Priority bar — 3px wide, full height (size.priority-bar-w) */}
        <span
          aria-hidden
          className={`block h-full w-[3px] shrink-0 ${PRIORITY_BAR[priorityKey]}`}
        />

        <StatusIcon state={state} />

        <span
          className={`flex-1 truncate text-[11px] ${styles.text} ${state === "done" ? "" : ""}`}
        >
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

      {isHovering && !inPopover && (
        <CardPreviewPopover
          anchorEl={pillEl}
          card={card}
          state={state}
          boardId={boardId}
          onClose={() => setIsHovering(false)}
          onPointerEnter={onPopoverEnter}
          onPointerLeave={onPopoverLeave}
          onOpenCard={() => {
            setIsHovering(false);
            setIsDetailOpen(true);
          }}
        />
      )}

      {isDetailOpen && (
        <CardDetailModal
          key={card.id}
          card={card}
          boardId={boardId}
          isOpen={isDetailOpen}
          onClose={() => setIsDetailOpen(false)}
          onUpdated={onSave}
          onDelete={onDelete}
          onAddSubtask={handleAddSubtask}
          canEdit={canEdit}
        />
      )}
    </>
  );
}
