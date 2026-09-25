"use client";

import { memo, useState, useMemo } from "react";
import { useDroppable } from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
  ChevronsLeft,
  ChevronsRight,
  CircleCheck,
  MoreHorizontal,
  Plus,
} from "lucide-react";
import { TaskCard } from "./TaskCard";
import { CreateTaskModal } from "./CreateTaskModal";
import type { Card, Column } from "@/types/board";
import { FormState } from "../card-modal/CardDetailModal";
import {
  ColumnOptionsModal,
  getColumnColorHex,
  columnAccentColor,
  columnBarColor,
} from "./ColumnOptionsModal";
import { useCanManageBoard } from "@/hooks/useBoardRole";
import { UNASSIGNED_FILTER } from "@/store/useBoardStore";

// DONE columns default to collapsed; the choice is saved per board + column.
const collapseKey = (boardId: string, columnId: string) =>
  `turtask:col-collapsed:${boardId}:${columnId}`;

function readCollapsed(
  boardId: string,
  columnId: string,
  fallback: boolean,
): boolean {
  if (typeof window === "undefined") return fallback;
  try {
    const v = window.localStorage.getItem(collapseKey(boardId, columnId));
    return v === null ? fallback : v === "1";
  } catch {
    return fallback;
  }
}

interface ColumnProps {
  id: string;
  title: string;
  category: Column["category"];
  color?: string | null;
  boardId: string;
  cards: Card[];
  onAddCard: (
    columnId: string,
    title: string,
    opts?: {
      assigneeId: string | null;
      priority: string | null;
      dueDate: string | null;
      description?: string | null;
      subtasks?: string[];
    },
  ) => void;
  onDeleteCard: (cardId: string) => void;
  onSaveCard: (cardId: string, form: FormState, field: keyof FormState) => void;
  onDeleteColumn: (columnId: string) => void;
  onUpdateColumn: (
    columnId: string,
    title: string,
    category: "TODO" | "DONE",
    color: string | null,
  ) => void;
  filterAssigneeId?: string | null;
  filterPriorities?: string[];
  filterTagIds?: string[];
  dropIndicatorBeforeId?: string | null;
}

const DropIndicator = () => (
  <div className="h-0.5 bg-blue-400 rounded-full mx-1 my-0.5" />
);

export const KanbanColumn = memo(function KanbanColumn({
  id,
  boardId,
  title,
  category,
  color,
  cards,
  onAddCard,
  onDeleteCard,
  onSaveCard,
  onDeleteColumn,
  onUpdateColumn,
  filterAssigneeId,
  filterPriorities,
  filterTagIds,
  dropIndicatorBeforeId,
}: ColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id });
  const [optionsOpen, setOptionsOpen] = useState(false);
  const [topAddOpen, setTopAddOpen] = useState(false);
  const canManage = useCanManageBoard();

  const isDone = category === "DONE";
  const [collapsed, setCollapsed] = useState(() =>
    readCollapsed(boardId, id, isDone),
  );
  const setCollapsedPersisted = (next: boolean) => {
    setCollapsed(next);
    try {
      window.localStorage.setItem(collapseKey(boardId, id), next ? "1" : "0");
    } catch {
      /* localStorage unavailable (private mode) */
    }
  };

  const visibleCards = useMemo(
    () =>
      cards.filter(
        (card) =>
          (filterAssigneeId == null ||
            (filterAssigneeId === UNASSIGNED_FILTER
              ? card.assignee_id == null
              : card.assignee_id === filterAssigneeId)) &&
          (filterPriorities == null ||
            filterPriorities.length === 0 ||
            filterPriorities.includes(card.priority ?? "")) &&
          (filterTagIds == null ||
            filterTagIds.length === 0 ||
            (card.tags?.some((t) => filterTagIds.includes(t.id)) ?? false)),
      ),
    [cards, filterAssigneeId, filterPriorities, filterTagIds],
  );

  // Bar formula lives in ColumnOptionsModal so its preview matches.
  const barColor = columnBarColor(
    columnAccentColor(getColumnColorHex(color), isDone),
  );
  const dropRing = isOver
    ? { boxShadow: `inset 0 0 0 2px ${barColor}` }
    : undefined;

  const showCollapsed = isDone && collapsed;

  // Collapsed DONE strip — still a drop target.
  if (showCollapsed) {
    return (
      <>
        <div
          ref={setNodeRef}
          className="group relative w-16 shrink-0 snap-start flex flex-col items-center overflow-hidden rounded-xl bg-slate-100 transition-shadow duration-200"
          style={dropRing}
        >
          <span
            aria-hidden
            className="absolute inset-x-0 top-0 h-0.75"
            style={{ backgroundColor: barColor }}
          />
          <button
            onClick={() => setCollapsedPersisted(false)}
            title="ขยายคอลัมน์ที่เสร็จแล้ว"
            className="flex h-full w-full cursor-pointer flex-col items-center pt-4 pb-3"
          >
            <CircleCheck size={16} className="shrink-0" style={{ color: barColor }} />
            <span className="mt-3 min-w-5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-center text-xs font-semibold text-slate-600">
              {cards.length}
            </span>
            <span
              className="mt-3 rotate-180 text-[13px] font-bold text-slate-600"
              style={{ writingMode: "vertical-rl" }}
            >
              {title}
            </span>
            <span className="mt-auto pt-3 text-slate-400 transition-colors group-hover:text-slate-600">
              <ChevronsLeft size={16} />
            </span>
          </button>
          {isOver && (
            <span
              className="pointer-events-none absolute inset-0 flex rotate-180 items-center justify-center text-[11px] font-bold"
              style={{ writingMode: "vertical-rl", color: barColor }}
            >
              วางเพื่อปิดงาน
            </span>
          )}
        </div>

        <ColumnOptionsModal
          key={`${id}-${optionsOpen}`}
          open={optionsOpen}
          initialTitle={title}
          initialCategory={category}
          initialColor={color ?? null}
          cardCount={cards.length}
          onSave={(t, cat, col) => onUpdateColumn(id, t, cat, col)}
          onDelete={() => onDeleteColumn(id)}
          onClose={() => setOptionsOpen(false)}
        />
      </>
    );
  }

  return (
    <>
      <div
        ref={setNodeRef}
        className="w-72 shrink-0 flex flex-col snap-start rounded-xl bg-slate-100 transition-shadow duration-200"
        style={dropRing}
      >
        {/* Header — sticky, so the accent bar rides inside it */}
        <div className="sticky top-0 z-10 overflow-hidden flex items-center gap-2 h-12 pl-4 pr-2 rounded-t-xl bg-slate-100 border-b border-slate-200">
          <span
            aria-hidden
            className="absolute inset-x-0 top-0 h-0.75"
            style={{ backgroundColor: barColor }}
          />
          {isDone ? (
            <CircleCheck size={14} className="shrink-0" style={{ color: barColor }} />
          ) : (
            <span
              className="h-1.25 w-1.25 shrink-0 rounded-full"
              style={{ backgroundColor: barColor }}
            />
          )}
          <h2 className="min-w-0 truncate text-sm font-semibold leading-tight text-slate-900">
            {title}
          </h2>
          <span className="min-w-5 shrink-0 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-center text-xs font-semibold text-slate-600">
            {cards.length}
          </span>

          <div className="ml-auto flex items-center gap-0.5 shrink-0">
            {!isDone && (
              <button
                onClick={() => setTopAddOpen(true)}
                title="Add card"
                className="cursor-pointer text-slate-500 hover:text-slate-900 p-1 rounded-md hover:bg-slate-200 transition-colors"
              >
                <Plus size={16} />
              </button>
            )}

            {isDone && (
              <button
                onClick={() => setCollapsedPersisted(true)}
                title="ยุบคอลัมน์ที่เสร็จแล้ว"
                className="cursor-pointer text-slate-500 hover:text-slate-900 p-1 rounded-md hover:bg-slate-200 transition-colors"
              >
                <ChevronsRight size={16} />
              </button>
            )}

            {canManage && (
              <button
                onClick={() => setOptionsOpen(true)}
                title="Column options"
                className="cursor-pointer text-slate-500 hover:text-slate-900 p-1 rounded-md hover:bg-slate-200 transition-colors"
              >
                <MoreHorizontal size={16} />
              </button>
            )}
          </div>
        </div>

        {/* Body */}
        <SortableContext
          items={cards.map((c) => c.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="px-3 pt-3 pb-4 flex flex-col gap-3 flex-1">
            {visibleCards.map((card) => (
              <div key={card.id}>
                {dropIndicatorBeforeId === card.id && <DropIndicator />}
                <TaskCard
                  boardId={boardId}
                  card={card}
                  onDeleteCard={onDeleteCard}
                  onSaveCard={onSaveCard}
                />
              </div>
            ))}
            {dropIndicatorBeforeId === null && <DropIndicator />}
            {isDone && visibleCards.length === 0 && (
              <p className="px-1 py-6 text-center text-xs text-slate-400">
                ยังไม่มีงานเสร็จ — ลากการ์ดมาที่นี่เพื่อปิดงาน
              </p>
            )}
          </div>
        </SortableContext>
      </div>

      {topAddOpen && (
        <CreateTaskModal
          defaultColumnId={id}
          onCreate={onAddCard}
          onClose={() => setTopAddOpen(false)}
        />
      )}

      {/* `key` remounts per column so state starts fresh. */}
      <ColumnOptionsModal
        key={`${id}-${optionsOpen}`}
        open={optionsOpen}
        initialTitle={title}
        initialCategory={category}
        initialColor={color ?? null}
        cardCount={cards.length}
        onSave={(t, cat, col) => onUpdateColumn(id, t, cat, col)}
        onDelete={() => onDeleteColumn(id)}
        onClose={() => setOptionsOpen(false)}
      />
    </>
  );
});
