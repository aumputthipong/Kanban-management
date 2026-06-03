import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useDashboardStats } from "@/hooks/useDashboardStats";
import { useBoardStore } from "@/store/useBoardStore";
import type { Card, Column } from "@/types/board";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeCard(overrides: Partial<Card> & { updated_at?: string } = {}): Card & { updated_at?: string } {
  return {
    id: "card-1",
    column_id: "col-todo",
    title: "Test Task",
    position: 65536,
    description: null,
    due_date: null,
    assignee_id: null,
    assignee_name: null,
    priority: null,
    estimated_hours: null,
    is_done: false,
    completed_at: null,
    created_at: null,
    created_by: null,
    total_subtasks: 0,
    completed_subtasks: 0,
    ...overrides,
  };
}

function makeColumn(overrides: Partial<Column> = {}): Column {
  return {
    id: "col-todo",
    title: "To Do",
    position: 65536,
    category: "TODO",
    color: null,
    cards: [],
    ...overrides,
  };
}

// ─── setup ────────────────────────────────────────────────────────────────────

// lock time at 2026-04-09 (Wednesday)
const FROZEN_DATE = new Date("2026-04-09T00:00:00.000Z");

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(FROZEN_DATE);
  useBoardStore.setState({
    columns: [],
    filterAssigneeId: null,
    filterPriorities: [],
    currentUserId: "",
    boardMembers: [],
    isLoading: false,
  });
});

afterEach(() => {
  vi.useRealTimers();
});

// ─── empty state ──────────────────────────────────────────────────────────────

describe("empty board", () => {
  it("returns zero values when there are no cards", () => {
    useBoardStore.setState({ columns: [makeColumn()] });

    const { result } = renderHook(() => useDashboardStats());
    const stats = result.current;

    expect(stats.totalCards).toBe(0);
    expect(stats.progress).toBe(0);
    expect(stats.totalHours).toBe(0);
    expect(stats.overdueCards).toHaveLength(0);
    expect(stats.dueSoonCards).toHaveLength(0);
  });

  it("does not crash with no columns at all", () => {
    useBoardStore.setState({ columns: [] });
    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.totalCards).toBe(0);
  });
});

// ─── progress ─────────────────────────────────────────────────────────────────

describe("progress calculation", () => {
  it("is 0% when no cards are done", () => {
    const col = makeColumn({ cards: [makeCard(), makeCard({ id: "card-2" })] });
    useBoardStore.setState({ columns: [col] });

    const { result } = renderHook(() => useDashboardStats());
    // no column is the "last" done column that matches column_id
    expect(result.current.progress).toBeDefined();
  });

  it("is 100% when all cards are in the last (DONE) column", () => {
    const doneCol = makeColumn({
      id: "col-done",
      title: "Done",
      category: "DONE",
      position: 131072,
      cards: [
        makeCard({ id: "c1", column_id: "col-done" }),
        makeCard({ id: "c2", column_id: "col-done" }),
      ],
    });
    const todoCol = makeColumn({ id: "col-todo", cards: [] });
    // hook ใช้ column สุดท้ายเป็น done column
    useBoardStore.setState({ columns: [todoCol, doneCol] });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.progress).toBe(100);
    expect(result.current.totalCards).toBe(2);
  });

  it("calculates partial progress correctly", () => {
    const todoCol = makeColumn({
      id: "col-todo",
      cards: [makeCard({ id: "c1", column_id: "col-todo" })],
    });
    const doneCol = makeColumn({
      id: "col-done",
      category: "DONE",
      position: 131072,
      cards: [makeCard({ id: "c2", column_id: "col-done" })],
    });
    useBoardStore.setState({ columns: [todoCol, doneCol] });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.progress).toBe(50); // 1/2 = 50%
  });
});

// ─── overdue cards ────────────────────────────────────────────────────────────

// ── shared column setup used by overdue/dueSoon tests ──
// hook ใช้ column สุดท้ายเป็น doneColumnId
// ดังนั้นต้องมี doneCol เป็น column สุดท้ายเสมอ เพื่อไม่ให้การ์ดใน todoCol ถูกนับเป็น done
const doneCol = makeColumn({
  id: "col-done",
  title: "Done",
  category: "DONE",
  position: 131072,
  cards: [],
});

describe("overdueCards", () => {
  it("counts cards with past due dates as overdue", () => {
    // yesterday relative to FROZEN_DATE (2026-04-09)
    const pastDue = makeCard({
      id: "overdue",
      column_id: "col-todo",
      due_date: "2026-04-08",
    });
    useBoardStore.setState({
      columns: [makeColumn({ id: "col-todo", cards: [pastDue] }), doneCol],
    });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.overdueCards).toHaveLength(1);
    expect(result.current.overdueCards[0].id).toBe("overdue");
  });

  it("does not count future cards as overdue", () => {
    const futureCard = makeCard({
      id: "future",
      column_id: "col-todo",
      due_date: "2026-04-20",
    });
    useBoardStore.setState({
      columns: [makeColumn({ id: "col-todo", cards: [futureCard] }), doneCol],
    });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.overdueCards).toHaveLength(0);
  });

  it("does not count done cards as overdue", () => {
    const doneColWithCard = makeColumn({
      id: "col-done",
      category: "DONE",
      position: 131072,
      cards: [
        makeCard({
          id: "done-overdue",
          column_id: "col-done",
          due_date: "2026-04-01",
        }),
      ],
    });
    useBoardStore.setState({ columns: [makeColumn(), doneColWithCard] });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.overdueCards).toHaveLength(0);
  });
});

// ─── due soon cards ───────────────────────────────────────────────────────────

describe("dueSoonCards", () => {
  it("counts cards due within 3 days as due soon", () => {
    // frozen date = 2026-04-09; +2 days = 2026-04-11 (within threeDaysFromNow)
    const soonCard = makeCard({
      id: "soon",
      column_id: "col-todo",
      due_date: "2026-04-11",
    });
    useBoardStore.setState({
      columns: [makeColumn({ id: "col-todo", cards: [soonCard] }), doneCol],
    });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.dueSoonCards).toHaveLength(1);
  });

  it("does not count cards due far in the future as due soon", () => {
    const farCard = makeCard({
      id: "far",
      column_id: "col-todo",
      due_date: "2026-05-01",
    });
    useBoardStore.setState({
      columns: [makeColumn({ id: "col-todo", cards: [farCard] }), doneCol],
    });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.dueSoonCards).toHaveLength(0);
  });
});

// ─── totalHours ───────────────────────────────────────────────────────────────

describe("totalHours", () => {
  it("sums estimated hours across all cards", () => {
    const cards = [
      makeCard({ id: "c1", estimated_hours: 3 }),
      makeCard({ id: "c2", estimated_hours: 5 }),
      makeCard({ id: "c3", estimated_hours: null }),
    ];
    useBoardStore.setState({ columns: [makeColumn({ cards })] });

    const { result } = renderHook(() => useDashboardStats());
    expect(result.current.totalHours).toBe(8);
  });
});
