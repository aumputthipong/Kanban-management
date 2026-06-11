"use client";

import { useState, useMemo, useRef, useEffect } from "react";
import {
  format,
  addMonths,
  subMonths,
  startOfMonth,
  endOfMonth,
  startOfWeek,
  endOfWeek,
  isSameMonth,
  isSameDay,
  addDays,
} from "date-fns";
import { createPortal } from "react-dom";
import dynamic from "next/dynamic";
import { X } from "lucide-react";
import { useBoardStore, UNASSIGNED_FILTER } from "@/store/useBoardStore";
import { useBoardActions } from "@/hooks/useBoardActions";
import { useCanEdit } from "@/hooks/useCanEdit";
import type { Card } from "@/types/board";
import type { FormState } from "@/components/board/card-modal/CardDetailModal";
import { CalendarHeader, type CalendarView } from "./CalendarHeader";
import { CalendarFilters } from "./CalendarFilters";
import { TaskPill } from "./TaskPill";
import { CalendarTaskModal } from "./CalendarTaskModal";
import { classifyPillState, type PillState } from "./pillState";

const CardDetailModal = dynamic(
  () =>
    import("@/components/board/card-modal/CardDetailModal").then((m) => ({
      default: m.CardDetailModal,
    })),
  { ssr: false },
);

interface Props {
  boardId: string;
}

// Hard cap on pills per cell — anything past this collapses into "+N more"
// so all cells render the same height regardless of load. design.md rule:
// "Don't grow calendar cells to fit content."
const MAX_PILLS_PER_CELL = 3;

export function ProjectCalendar({ boardId }: Props) {
  const [currentDate, setCurrentDate] = useState(new Date());
  const [view, setView] = useState<CalendarView>("month");
  const [statusFilter, setStatusFilter] = useState<PillState[]>([]);
  const [myTasksOnly, setMyTasksOnly] = useState(false);
  const [moreCellDate, setMoreCellDate] = useState<Date | null>(null);
  // Clicking a task opens a read-only detail modal; "เปิด task" promotes it to
  // the full editable card modal.
  const [selectedCard, setSelectedCard] = useState<Card | null>(null);
  const [editCard, setEditCard] = useState<Card | null>(null);
  const openCard = (c: Card) => {
    setMoreCellDate(null);
    setSelectedCard(c);
  };
  const today = useMemo(() => new Date(), []);

  const columns = useBoardStore((s) => s.columns);
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const filterAssigneeId = useBoardStore((s) => s.filterAssigneeId);
  const filterPriorities = useBoardStore((s) => s.filterPriorities);
  const filterTagIds = useBoardStore((s) => s.filterTagIds);

  // Flatten cards across columns, then apply all active filters.
  // Filters compose with AND semantics — every active chip narrows the set.
  const allDueCards = useMemo(() => {
    return columns
      .flatMap((c) => c.cards)
      .filter((card) => card.due_date)
      .filter((card) => {
        if (myTasksOnly && card.assignee_id !== currentUserId) return false;
        if (filterAssigneeId === UNASSIGNED_FILTER) {
          if (card.assignee_id != null) return false;
        } else if (filterAssigneeId && card.assignee_id !== filterAssigneeId) {
          return false;
        }
        if (filterPriorities.length > 0) {
          if (!card.priority || !filterPriorities.includes(card.priority)) return false;
        }
        if (filterTagIds.length > 0) {
          const cardTagIds = card.tags?.map((t) => t.id) ?? [];
          if (!filterTagIds.some((id) => cardTagIds.includes(id))) return false;
        }
        if (statusFilter.length > 0) {
          if (!statusFilter.includes(classifyPillState(card))) return false;
        }
        return true;
      });
  }, [
    columns,
    myTasksOnly,
    currentUserId,
    filterAssigneeId,
    filterPriorities,
    filterTagIds,
    statusFilter,
  ]);

  // Bucket filtered cards by ISO date for O(1) lookup during grid render.
  const cardsByDate = useMemo(() => {
    const map = new Map<string, Card[]>();
    for (const card of allDueCards) {
      if (!card.due_date) continue;
      const key = format(new Date(card.due_date), "yyyy-MM-dd");
      const bucket = map.get(key) ?? [];
      bucket.push(card);
      map.set(key, bucket);
    }
    return map;
  }, [allDueCards]);

  const monthStart = startOfMonth(currentDate);
  const monthEnd = endOfMonth(monthStart);
  const gridStart = startOfWeek(monthStart);
  const gridEnd = endOfWeek(monthEnd);

  const weeks: Date[][] = [];
  let cursor = gridStart;
  while (cursor <= gridEnd) {
    const week: Date[] = [];
    for (let i = 0; i < 7; i++) {
      week.push(cursor);
      cursor = addDays(cursor, 1);
    }
    weeks.push(week);
  }

  const toggleStatus = (s: PillState) =>
    setStatusFilter((prev) =>
      prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s],
    );

  return (
    <div className="mx-auto max-w-7xl">
      <CalendarHeader
        currentDate={currentDate}
        today={today}
        view={view}
        onPrev={() => setCurrentDate(subMonths(currentDate, 1))}
        onNext={() => setCurrentDate(addMonths(currentDate, 1))}
        onToday={() => setCurrentDate(new Date())}
        onSelectDate={setCurrentDate}
        onViewChange={setView}
      />

      <CalendarFilters
        statusFilter={statusFilter}
        onToggleStatus={toggleStatus}
        onClearStatus={() => setStatusFilter([])}
        myTasksOnly={myTasksOnly}
        onToggleMyTasks={() => setMyTasksOnly((v) => !v)}
      />

      <div className="overflow-hidden rounded-lg border border-slate-200">
        {/* Day-of-week header */}
        <div className="grid grid-cols-7 border-b border-slate-200 bg-slate-50">
          {["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"].map((d) => (
            <div
              key={d}
              className="px-2 py-2 text-center text-[10px] font-semibold tracking-wider text-slate-500"
            >
              {d}
            </div>
          ))}
        </div>

        {/* Grid */}
        <div>
          {weeks.map((week, wi) => (
            <div
              key={wi}
              className="grid grid-cols-7 border-b border-slate-200 last:border-b-0"
            >
              {week.map((day) => {
                const dayKey = format(day, "yyyy-MM-dd");
                const dayCards = cardsByDate.get(dayKey) ?? [];
                const isToday = isSameDay(day, today);
                const inMonth = isSameMonth(day, monthStart);
                const overflow = Math.max(0, dayCards.length - MAX_PILLS_PER_CELL);
                const visiblePills = dayCards.slice(0, MAX_PILLS_PER_CELL);

                return (
                  <div
                    key={day.toISOString()}
                    // Fixed height so every cell is the same regardless of pill count.
                    className={`relative flex h-32 flex-col gap-1 border-r border-slate-200 p-1.5 last:border-r-0 ${
                      isToday
                        ? "bg-indigo-50/60"
                        : inMonth
                          ? "bg-white"
                          : "bg-slate-50/50"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span
                        className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold ${
                          isToday
                            ? "bg-blue-800 text-white"
                            : inMonth
                              ? "text-slate-700"
                              : "text-slate-400"
                        }`}
                      >
                        {format(day, "d")}
                      </span>
                    </div>

                    <div className="flex flex-col gap-1 overflow-hidden">
                      {visiblePills.map((card) => (
                        <TaskPill key={card.id} card={card} onSelect={openCard} />
                      ))}
                      {overflow > 0 && (
                        <button
                          type="button"
                          onClick={() => setMoreCellDate(day)}
                          className="rounded px-1 text-left text-[10px] font-medium text-slate-500 hover:bg-slate-100 hover:text-slate-700"
                        >
                          + {overflow} more
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ))}
        </div>
      </div>

      {moreCellDate && (
        <DayDetailModal
          date={moreCellDate}
          cards={cardsByDate.get(format(moreCellDate, "yyyy-MM-dd")) ?? []}
          onSelect={openCard}
          onClose={() => setMoreCellDate(null)}
        />
      )}

      {selectedCard && (
        <CalendarTaskModal
          card={selectedCard}
          columnName={columns.find((c) => c.id === selectedCard.column_id)?.title}
          onClose={() => setSelectedCard(null)}
          onOpenTask={() => {
            setEditCard(selectedCard);
            setSelectedCard(null);
          }}
        />
      )}

      {editCard && (
        <EditCardModal
          card={editCard}
          boardId={boardId}
          onClose={() => setEditCard(null)}
        />
      )}
    </div>
  );
}

// Wrapper that owns the board-action hooks for the editable card modal — kept
// out of ProjectCalendar's body so its hooks only run while an edit is open.
function EditCardModal({
  card,
  boardId,
  onClose,
}: {
  card: Card;
  boardId: string;
  onClose: () => void;
}) {
  const { handleAddSubtask, handleDeleteCard, handleUpdateCard } =
    useBoardActions(boardId);
  const canEdit = useCanEdit(card);

  return (
    <CardDetailModal
      key={card.id}
      card={card}
      boardId={boardId}
      isOpen
      onClose={onClose}
      onUpdated={(cardId: string, form: FormState) =>
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
        })
      }
      onDelete={(cardId: string) => {
        handleDeleteCard(cardId);
        onClose();
      }}
      onAddSubtask={handleAddSubtask}
      canEdit={canEdit}
    />
  );
}

function DayDetailModal({
  date,
  cards,
  onSelect,
  onClose,
}: {
  date: Date;
  cards: Card[];
  onSelect: (card: Card) => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return createPortal(
    <div
      className="fixed inset-0 z-40 flex items-start justify-center bg-slate-900/40 p-8"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        ref={ref}
        className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
      >
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-base font-semibold text-slate-900">
            {format(date, "EEEE, d MMMM yyyy")}
          </h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="text-slate-400 hover:text-slate-600"
          >
            <X size={16} />
          </button>
        </div>
        <p className="mb-4 text-xs text-slate-500">
          {cards.length} task{cards.length === 1 ? "" : "s"}
        </p>
        <div className="flex flex-col gap-1.5">
          {cards.map((card) => (
            <TaskPill key={card.id} card={card} onSelect={onSelect} inPopover />
          ))}
        </div>
      </div>
    </div>,
    document.body,
  );
}
