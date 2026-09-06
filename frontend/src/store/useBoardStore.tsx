import { BoardMember, Card, Column, Subtask } from "@/types/board";
import { create } from "zustand";

/**
 * Single source of truth for one board's columns, cards and members. Setters are
 * driven both by optimistic user actions and by WS broadcasts, so they must stay
 * idempotent — that is what lets a writer skip filtering its own echo.
 */
/**
 * The viewed board's visual identity, hydrated on load. Separate from `columns`
 * because GET /boards/:id returns columns only — title/icon/color come from the list.
 */
export interface BoardMeta {
  title: string;
  color?: string;
  icon?: string;
}

// Sentinel for `filterAssigneeId` meaning "no assignee". null already means "All",
// and real assignees are UUIDs, so this string cannot collide with a user id.
export const UNASSIGNED_FILTER = "unassigned";

interface BoardState {
  columns: Column[];
  currentUserId: string;
  boardMembers: BoardMember[];
  boardMeta: BoardMeta | null;
  isLoading: boolean;
  filterAssigneeId: string | null;
  filterPriorities: string[];
  filterTagIds: string[];
  setColumns: (columns: Column[]) => void;
  setCurrentUser: (userId: string) => void;
  setBoardMembers: (members: BoardMember[]) => void;
  setBoardMeta: (meta: BoardMeta | null) => void;
  patchBoardMeta: (patch: Partial<BoardMeta>) => void;
  setLoading: (v: boolean) => void;
  setFilterAssigneeId: (id: string | null) => void;
  toggleFilterPriority: (p: string) => void;
  clearFilterPriorities: () => void;
  toggleFilterTag: (tagId: string) => void;
  clearFilterTags: () => void;
  moveCard: (
    cardId: string,
    toColumnId: string,
    position?: number,
    isDone?: boolean,
    completedAt?: string | null,
  ) => void;
  addCardToStore: (newCard: Card) => void;
  removeCardFromStore: (cardId: string) => void;
  updateCard: (updated: Card) => void;
  setSubtasksToCard: (cardId: string, subtasks: Subtask[]) => void;
  updateSubtaskInCard: (
    cardId: string,
    subtaskId: string,
    updatedData: Partial<Subtask>,
  ) => void;
  deleteSubtaskFromCard: (cardId: string, subtaskId: string) => void;
  addColumnToStore: (column: Column) => void;
  renameColumnInStore: (columnId: string, title: string) => void;
  removeColumnFromStore: (columnId: string) => void;
  updateColumnInStore: (columnId: string, patch: Partial<Pick<Column, "title" | "category" | "color">>) => void;
}

export const useBoardStore = create<BoardState>((set) => ({
  columns: [],
  currentUserId: "",
  boardMembers: [],
  boardMeta: null,
  isLoading: false,
  filterAssigneeId: null,
  filterPriorities: [],
  filterTagIds: [],
  setColumns: (columns) => set({ columns }),
  setCurrentUser: (userId) => set({ currentUserId: userId }),
  setBoardMembers: (members) => set({ boardMembers: members }),
  setBoardMeta: (meta) => set({ boardMeta: meta }),
  patchBoardMeta: (patch) =>
    set((state) =>
      state.boardMeta ? { boardMeta: { ...state.boardMeta, ...patch } } : state,
    ),
  setLoading: (v) => set({ isLoading: v }),
  setFilterAssigneeId: (id) => set({ filterAssigneeId: id }),
  toggleFilterPriority: (p) =>
    set((state) => ({
      filterPriorities: state.filterPriorities.includes(p)
        ? state.filterPriorities.filter((x) => x !== p)
        : [...state.filterPriorities, p],
    })),
  clearFilterPriorities: () => set({ filterPriorities: [] }),
  toggleFilterTag: (tagId) =>
    set((state) => ({
      filterTagIds: state.filterTagIds.includes(tagId)
        ? state.filterTagIds.filter((x) => x !== tagId)
        : [...state.filterTagIds, tagId],
    })),
  clearFilterTags: () => set({ filterTagIds: [] }),

  moveCard: (cardId, toColumnId, position, isDone, completedAt) =>
    set((state) => {
      const newColumns = state.columns.map((col) => ({
        ...col,
        cards: [...col.cards],
      }));

      let fromColIndex = -1;
      let cardIndex = -1;
      for (let i = 0; i < newColumns.length; i++) {
        const idx = newColumns[i].cards.findIndex((c) => c.id === cardId);
        if (idx !== -1) {
          fromColIndex = i;
          cardIndex = idx;
          break;
        }
      }

      if (fromColIndex === -1) return state;

      const toColIndex = newColumns.findIndex((c) => c.id === toColumnId);
      if (toColIndex === -1) return state;

      const [movedCard] = newColumns[fromColIndex].cards.splice(cardIndex, 1);

      movedCard.column_id = toColumnId;
      if (position !== undefined) movedCard.position = position;
      if (isDone !== undefined) movedCard.is_done = isDone;
      if (completedAt !== undefined)
        movedCard.completed_at = completedAt ?? null;

      newColumns[toColIndex].cards.push(movedCard);
      newColumns[toColIndex].cards.sort((a, b) => a.position - b.position);

      return { columns: newColumns };
    }),

  addCardToStore: (newCard) =>
    set((state) => {
      // Idempotent — ignore a card id we already hold (e.g. our own WS echo).
      const isAlreadyExists = state.columns.some((col) =>
        col.cards.some((card) => card.id === newCard.id),
      );
      if (isAlreadyExists) {
        return state;
      }

      return {
        columns: state.columns.map((col) => {
          if (col.id === newCard.column_id) {
            return {
              ...col,
              cards: [...col.cards, newCard],
            };
          }
          return col;
        }),
      };
    }),
  removeCardFromStore: (cardId) =>
    set((state) => ({
      columns: state.columns.map((col) => ({
        ...col,
        cards: col.cards.filter((card) => card.id !== cardId),
      })),
    })),

  updateCard: (updated) =>
    set((state) => ({
      columns: state.columns.map((col) => ({
        ...col,
        cards: col.cards.map((card) =>
          card.id === updated.id
            ? {
                ...card,
                ...updated,
                total_subtasks: updated.total_subtasks ?? card.total_subtasks,
                completed_subtasks:
                  updated.completed_subtasks ?? card.completed_subtasks,
                subtasks: updated.subtasks ?? card.subtasks,
              }
            : card,
        ),
      })),
    })),

  setSubtasksToCard: (cardId, subtasks) =>
    set((state) => ({
      columns: state.columns.map((col) => ({
        ...col,
        cards: col.cards.map((card) =>
          card.id === cardId
            ? {
                ...card,
                subtasks,
                total_subtasks: subtasks.length,
                completed_subtasks: subtasks.filter((st) => st.is_done).length,
              }
            : card,
        ),
      })),
    })),

  updateSubtaskInCard: (cardId, subtaskId, updatedData) =>
    set((state) => ({
      columns: state.columns.map((col) => ({
        ...col,
        cards: col.cards.map((card) => {
          if (card.id !== cardId) return card;
          const newSubtasks = card.subtasks
            ? card.subtasks.map((st) =>
                st.id === subtaskId ? { ...st, ...updatedData } : st,
              )
            : card.subtasks;
          let completedDelta = 0;
          if (updatedData.is_done !== undefined && card.subtasks) {
            const old = card.subtasks.find((st) => st.id === subtaskId);
            if (old && old.is_done !== updatedData.is_done) {
              completedDelta = updatedData.is_done ? 1 : -1;
            }
          }
          return {
            ...card,
            subtasks: newSubtasks,
            completed_subtasks: card.completed_subtasks + completedDelta,
          };
        }),
      })),
    })),

  deleteSubtaskFromCard: (cardId, subtaskId) =>
    set((state) => ({
      columns: state.columns.map((col) => ({
        ...col,
        cards: col.cards.map((card) => {
          if (card.id !== cardId || !card.subtasks) return card;
          const removed = card.subtasks.find((st) => st.id === subtaskId);
          return {
            ...card,
            subtasks: card.subtasks.filter((st) => st.id !== subtaskId),
            total_subtasks: card.total_subtasks - 1,
            completed_subtasks:
              card.completed_subtasks - (removed?.is_done ? 1 : 0),
          };
        }),
      })),
    })),

  addColumnToStore: (column) =>
    set((state) => {
      if (state.columns.some((c) => c.id === column.id)) return state;
      const sorted = [...state.columns, column].sort(
        (a, b) => a.position - b.position,
      );
      return { columns: sorted };
    }),

  renameColumnInStore: (columnId, title) =>
    set((state) => ({
      columns: state.columns.map((col) =>
        col.id === columnId ? { ...col, title } : col,
      ),
    })),

  removeColumnFromStore: (columnId) =>
    set((state) => ({
      columns: state.columns.filter((col) => col.id !== columnId),
    })),

  updateColumnInStore: (columnId, patch) =>
    set((state) => ({
      columns: state.columns.map((col) =>
        col.id === columnId ? { ...col, ...patch } : col
      ),
    })),
}));
