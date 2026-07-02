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
import {
  ColumnOptionsModal,
  getColumnColorHex,
  columnAccentColor,
  columnCapColor,
  columnBodyBg,
  columnBodyBorder,
} from "./ColumnOptionsModal";
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
    opts?: {
      assigneeId: string | null;
      priority: string | null;
      dueDate: string | null;
      description?: string | null;
      subtasks?: string[];
    },
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
  // Column identity colour — the custom colour, or a neutral/emerald category
  // fallback so every column still reads at a glance without a custom colour.
  const accentColor = columnAccentColor(colorHex, isDone);

  // D1 "Solid Cap": a filled identity-colour cap over a faint wash of the same
  // hue. Formulae live in ColumnOptionsModal so the options preview stays truthful.
  const capColor = columnCapColor(accentColor);
  const bodyStyle = {
    backgroundColor: columnBodyBg(accentColor, isOver),
    borderColor: columnBodyBorder(accentColor, isOver),
  };

  const showCollapsed = isDone && collapsed;

  // ── Collapsed DONE strip — still a live drop target (close a card by
  //    dragging it onto the strip). The droppable ref stays mounted here. ──
  if (showCollapsed) {
    return (
      <>
        <div
          ref={setNodeRef}
          className="group relative w-16 shrink-0 snap-start flex flex-col items-center rounded-2xl border transition-colors duration-200"
          style={{
            backgroundColor: columnBodyBg(accentColor, isOver),
            borderColor: isOver
              ? capColor
              : columnBodyBorder(accentColor, false),
            borderStyle: isOver ? "dashed" : "solid",
          }}
        >
          <span
            className="absolute left-3 right-3 top-0 h-[3px] rounded-b"
            style={{ backgroundColor: capColor }}
          />
          <button
            onClick={() => setCollapsedPersisted(false)}
            title="ขยายคอลัมน์ที่เสร็จแล้ว"
            className="flex h-full w-full cursor-pointer flex-col items-center pt-4 pb-3"
          >
            <span
              className="flex h-7 w-7 items-center justify-center rounded-lg text-white"
              style={{ backgroundColor: capColor }}
            >
              <CircleCheck size={16} />
            </span>
            <span
              className="mt-3 min-w-5 rounded-full px-2 py-0.5 text-center text-xs font-bold text-white"
              style={{ backgroundColor: capColor }}
            >
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
              style={{ writingMode: "vertical-rl", color: capColor }}
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
        className="w-72 shrink-0 flex flex-col snap-start rounded-2xl transition-all duration-200"
      >
        {/* Solid Cap — filled identity-colour header, white text (D1) */}
        <div
          className="sticky top-0 z-10 flex items-center gap-2 h-12 pl-4 pr-2 rounded-t-2xl"
          style={{ backgroundColor: capColor }}
        >
          {isDone && (
            <CircleCheck size={16} className="text-white shrink-0" />
          )}
          <h2 className="flex-1 min-w-0 truncate font-bold leading-tight text-white">
            {title}
          </h2>

          <div className="flex items-center gap-1 shrink-0">
            <span className="min-w-5 rounded-full bg-white/25 px-2 py-0.5 text-center text-xs font-bold text-white">
              {cards.length}
            </span>

            {/* DONE columns close cards by dragging — no inline add button */}
            {!isDone && (
              <button
                onClick={() => setTopAddOpen(true)}
                title="Add card"
                className="cursor-pointer text-white/85 hover:text-white p-1 rounded-md hover:bg-white/20 transition-colors"
              >
                <Plus size={16} />
              </button>
            )}

            {isDone && (
              <button
                onClick={() => setCollapsedPersisted(true)}
                title="ยุบคอลัมน์ที่เสร็จแล้ว"
                className="cursor-pointer text-white/85 hover:text-white p-1 rounded-md hover:bg-white/20 transition-colors"
              >
                <ChevronsRight size={16} />
              </button>
            )}

            {canManage && (
              <button
                onClick={() => setOptionsOpen(true)}
                title="Column options"
                className="cursor-pointer text-white/85 hover:text-white p-1 rounded-md hover:bg-white/20 transition-colors"
              >
                <MoreHorizontal size={16} />
              </button>
            )}
          </div>
        </div>

        {/* Body — faint wash of the identity colour, 1px hued border (D1) */}
        <SortableContext
          items={cards.map((c) => c.id)}
          strategy={verticalListSortingStrategy}
        >
          <div
            className="px-3 pt-3 pb-4 flex flex-col gap-2 flex-1 border border-t-0 rounded-b-2xl transition-colors"
            style={bodyStyle}
          >
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
