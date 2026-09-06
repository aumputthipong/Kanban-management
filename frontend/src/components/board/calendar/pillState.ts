import type { Card } from "@/types/board";

export type PillState = "todo" | "inProgress" | "done" | "overdue";

/**
 * Maps a card to one of four calendar-pill states, checked in order: done, overdue,
 * inProgress, todo. A done card past its due date stays "done" — the work shipped,
 * just late — and a card due today is "todo", not overdue.
 */
export function classifyPillState(card: Card): PillState {
  if (card.is_done) return "done";

  if (card.due_date) {
    const dueStart = new Date(card.due_date).setHours(0, 0, 0, 0);
    const todayStart = new Date().setHours(0, 0, 0, 0);
    if (dueStart < todayStart) return "overdue";
  }

  if (card.total_subtasks > 0 && card.completed_subtasks > 0) {
    return "inProgress";
  }

  return "todo";
}
