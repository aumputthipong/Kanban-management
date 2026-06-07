import { BoardMember, Card, Column, Subtask } from "@/types/board";
import { create } from "zustand";

/**
 * Single source of truth for one viewed board's columns + cards + members.
 *
 * Mutations come from two places:
 *   1. User actions through API responses — handlers call `set*` / `update*` /
 *      `addCardToStore` etc. directly (optimistic UI).
 *   2. WebSocket broadcasts — `useWebSocket` dispatches CARD_*, COLUMN_*,
 *      ACTIVITY_* messages straight into these setters. They are designed
 *      to be **idempotent** so receiving an event for state that already
 *      reflects it is a no-op (this is what lets the writer skip filtering
 *      out their own broadcasts).
 *
 * `currentUserId` and `boardMembers` are set once on board load and used by
 * permission-aware UI (hiding manager-only buttons, etc.).
 */
/**
 * The viewed board's visual identity (title + accent colour + glyph key),
 * hydrated once on board load so the header can show "which project am I in".
 * Kept separate from `columns` because GET /boards/:id returns columns only —
 * the board row's title/icon/color come from the list endpoint.
 */
export interface BoardMeta {
  title: string;
  color?: string;
  icon?: string;
}

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

// 3. สร้าง Store ด้วย Zustand (ตัวจัดการ State ที่ทำงานเร็วกว่า Redux และตั้งค่าง่ายกว่ามาก)
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

      // ค้นหา card จากทุก column
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

      // ดึงการ์ดออกจาก column เดิม
      const [movedCard] = newColumns[fromColIndex].cards.splice(cardIndex, 1);

      // อัปเดต fields
      movedCard.column_id = toColumnId;
      if (position !== undefined) movedCard.position = position;
      if (isDone !== undefined) movedCard.is_done = isDone;
      if (completedAt !== undefined)
        movedCard.completed_at = completedAt ?? null;

      // เพิ่มการ์ดและ sort ตาม position
      newColumns[toColIndex].cards.push(movedCard);
      newColumns[toColIndex].cards.sort((a, b) => a.position - b.position);

      return { columns: newColumns };
    }),

  addCardToStore: (newCard) =>
    set((state) => {
      // 1. ตรวจสอบก่อนว่า ID นี้มีอยู่ในการ์ดใบไหนในทุก Column หรือยัง?
      // ใช้ .some() เพื่อความรวดเร็วในการเช็ก
      const isAlreadyExists = state.columns.some((col) =>
        col.cards.some((card) => card.id === newCard.id),
      );

      // 2. ถ้ามี ID นี้อยู่แล้ว ให้คืนค่า state เดิมกลับไป (ไม่ต้องอัปเดต)
      if (isAlreadyExists) {
        return state;
      }

      // 3. ถ้ายังไม่มี (เป็นเคสที่คนอื่นสร้าง หรือเราเพิ่งได้รับครั้งแรก) ค่อยเพิ่มเข้าไป
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
        // กรอง (filter) เอาเฉพาะการ์ดที่ ID ไม่ตรงกับตัวที่ถูกลบ ไว้ในคอลัมน์เดิม
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

  // ✨ [เพิ่มใหม่]: Implementation ของ setSubtasksToCard
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
          // อัปเดต completed_subtasks count ถ้ามีการเปลี่ยน is_done
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
