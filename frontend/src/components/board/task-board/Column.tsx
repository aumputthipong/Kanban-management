// components/kanban/Column.tsx
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
import { ColumnOptionsModal, getColumnColorHex } from "./ColumnOptionsModal";
import { useCanManageBoard } from "@/hooks/useBoardRole";
import { UNASSIGNED_FILTER } from "@/store/useBoardStore";

// Collapse state for DONE columns is remembered per board+column in localStorage.
// DONE columns default to collapsed (design: keep finished work out of the way,
// but the strip stays a live drop target). Read lazily — these components only
// mount client-side once the board store has loaded, so there is no SSR/hydration
// render to mismatch against.
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
    opts?: { assigneeId: string | null; priority: string | null; dueDate: string | null },
  ) => void;
  onDeleteCard: (cardId: string) => void;
  onSaveCard: (cardId: string, form: FormState) => void;
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
      /* localStorage unavailable (private mode) — collapse stays per-session */
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

  const colorHex = getColumnColorHex(color);
  const solidBgStyle = colorHex
    ? { backgroundColor: `color-mix(in srgb, ${colorHex} 12%, white)` }
    : undefined;
  // Accent for the top strip + title dot — the column's own colour, or a neutral
  // category fallback so every column still reads at a glance without a custom colour.
  const accentColor = colorHex ?? (isDone ? "#10b981" : "#94a3b8");

  // Surface treatment when no custom color: DONE reads as a calm green zone,
  // TODO keeps the neutral slate it always had.
  const surfaceBg = colorHex
    ? ""
    : isDone
      ? isOver
        ? "bg-emerald-100"
        : "bg-emerald-50"
      : isOver
        ? "bg-blue-50"
        : "bg-slate-100";
  const surfaceBorder = colorHex
    ? "border-transparent"
    : isOver
      ? isDone
        ? "border-emerald-300"
        : "border-blue-300"
      : "border-transparent";

  const showCollapsed = isDone && collapsed;

  // ── Collapsed DONE strip — still a live drop target (close a card by
  //    dragging it onto the strip). The droppable ref stays mounted here. ──
  if (showCollapsed) {
    return (
      <>
        <div
          ref={setNodeRef}
          className={`group relative w-16 shrink-0 snap-start flex flex-col items-center rounded-2xl border-2 transition-colors duration-200 ${
            isOver
              ? "border-dashed border-emerald-400 bg-emerald-50"
              : "border-transparent bg-emerald-50/70"
          }`}
        >
          <span className="absolute left-3 right-3 top-0 h-[3px] rounded-b bg-emerald-400" />
          <button
            onClick={() => setCollapsedPersisted(false)}
            title="ขยายคอลัมน์ที่เสร็จแล้ว"
            className="flex h-full w-full cursor-pointer flex-col items-center pt-4 pb-3"
          >
            <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700">
              <CircleCheck size={16} />
            </span>
            <span className="mt-3 min-w-5 rounded-full bg-emerald-100 px-2 py-0.5 text-center text-xs font-bold text-emerald-700">
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
              className="pointer-events-none absolute inset-0 flex rotate-180 items-center justify-center text-[11px] font-bold text-emerald-700"
              style={{ writingMode: "vertical-rl" }}
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
        className={`w-72 shrink-0 flex flex-col snap-start rounded-2xl border-2 transition-all duration-200 ${surfaceBorder} ${surfaceBg}`}
        style={solidBgStyle}
      >
        {/* Sticky header — colored accent strip + title row */}
        <div
          className={`sticky top-0 z-10 rounded-t-2xl transition-colors pb-2 ${surfaceBg}`}
          style={solidBgStyle}
        >
          {/* Top accent strip — the column's colour as a thin full-width bar. */}
          <div
            className="h-1.5 rounded-t-2xl"
            style={{ backgroundColor: accentColor }}
          />
          <div className="flex items-center justify-between gap-3 px-4 pt-3">
            <div className="flex items-center gap-2 min-w-0 flex-1">
              {isDone ? (
                <CircleCheck size={16} className="text-emerald-600 shrink-0" />
              ) : (
                <span
                  aria-hidden
                  className="w-2.5 h-2.5 rounded-full shrink-0"
                  style={{ backgroundColor: accentColor }}
                />
              )}
              <h2 className="font-bold text-slate-700 leading-tight truncate">
                {title}
              </h2>
            </div>

            <div className="flex items-center gap-1 shrink-0">
              <span
                className={`text-xs font-bold px-2 py-0.5 rounded-full min-w-5 text-center ${
                  isDone
                    ? "bg-emerald-100 text-emerald-700"
                    : "bg-slate-200 text-slate-500"
                }`}
              >
                {cards.length}
              </span>

              {/* DONE columns close cards by dragging — no inline add button */}
              {!isDone && (
                <button
                  onClick={() => setTopAddOpen(true)}
                  title="Add card"
                  className="cursor-pointer text-slate-400 hover:text-blue-600 p-1 rounded-md hover:bg-slate-200 transition-colors"
                >
                  <Plus size={16} />
                </button>
              )}

              {isDone && (
                <button
                  onClick={() => setCollapsedPersisted(true)}
                  title="ยุบคอลัมน์ที่เสร็จแล้ว"
                  className="cursor-pointer text-slate-400 hover:text-slate-600 p-1 rounded-md hover:bg-slate-200 transition-colors"
                >
                  <ChevronsRight size={16} />
                </button>
              )}

              {canManage && (
                <button
                  onClick={() => setOptionsOpen(true)}
                  title="Column options"
                  className="cursor-pointer text-slate-400 hover:text-slate-600 p-1 rounded-md hover:bg-slate-200 transition-colors"
                >
                  <MoreHorizontal size={16} />
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Card List — fills remaining column height */}
        <SortableContext
          items={cards.map((c) => c.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="px-4 pb-4 flex flex-col gap-2 flex-1">
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

      {/* Column "+" opens the full create modal preset to this column (drops the
          old title-only inline form for a consistent create UX). */}
      {topAddOpen && (
        <CreateTaskModal
          defaultColumnId={id}
          onCreate={onAddCard}
          onClose={() => setTopAddOpen(false)}
        />
      )}

      {/* `key` remounts the modal whenever a different column opens it, so
          internal state initialises fresh from the new initialTitle/Category/Color
          props instead of being sync'd inside an effect. */}
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
