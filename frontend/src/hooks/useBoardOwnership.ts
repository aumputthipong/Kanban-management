import { useMemo } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import type { Card } from "@/types/board";

/**
 * Per-member ownership of the loaded board, derived from the Zustand store — no
 * endpoint, so it updates for free via optimistic mutations and WS broadcasts. Held =
 * assigned and not done, counted only within non-DONE columns. See AGENTS.md,
 * "Ownership View Pattern".
 */
export interface OwnershipColumn {
  id: string;
  title: string;
  position: number;
  /** Column colour key (from the board's column settings); null = Default. */
  color: string | null;
}

export interface MemberOwnership {
  userId: string;
  name: string;
  totalHeld: number;
  /** columnId → count of held cards in that column. */
  countByColumn: Record<string, number>;
  /** The held cards themselves, ordered by due date (Phase 2 expand). */
  cards: Card[];
}

export interface BoardOwnership {
  columns: OwnershipColumn[];
  members: MemberOwnership[];
}

function dueRank(card: Card): number {
  if (!card.due_date) return Number.POSITIVE_INFINITY;
  const t = new Date(card.due_date).getTime();
  return Number.isNaN(t) ? Number.POSITIVE_INFINITY : t;
}

export function useBoardOwnership(): BoardOwnership {
  const columns = useBoardStore((s) => s.columns);
  const boardMembers = useBoardStore((s) => s.boardMembers);

  return useMemo(() => {
    const activeColumns: OwnershipColumn[] = columns
      .filter((c) => c.category !== "DONE")
      .sort((a, b) => a.position - b.position)
      .map((c) => ({ id: c.id, title: c.title, position: c.position, color: c.color ?? null }));

    const byUser = new Map<string, MemberOwnership>();
    // Seed every board member so idle people still appear (totalHeld = 0).
    boardMembers.filter(Boolean).forEach((m) => {
      byUser.set(m.user_id, {
        userId: m.user_id,
        name: m.full_name,
        totalHeld: 0,
        countByColumn: {},
        cards: [],
      });
    });

    columns.forEach((col) => {
      if (col.category === "DONE") return; // done columns aren't "held"
      col.cards.forEach((card) => {
        if (card.is_done || !card.assignee_id) return; // active + assigned only
        let entry = byUser.get(card.assignee_id);
        if (!entry) {
          // Assignee no longer a board member (e.g. left) — keep them visible.
          entry = {
            userId: card.assignee_id,
            name: card.assignee_name ?? "ไม่ทราบชื่อ",
            totalHeld: 0,
            countByColumn: {},
            cards: [],
          };
          byUser.set(card.assignee_id, entry);
        }
        entry.totalHeld += 1;
        entry.countByColumn[col.id] = (entry.countByColumn[col.id] ?? 0) + 1;
        entry.cards.push(card);
      });
    });

    const members = Array.from(byUser.values()).sort(
      (a, b) => b.totalHeld - a.totalHeld || a.name.localeCompare(b.name),
    );
    members.forEach((m) => m.cards.sort((a, b) => dueRank(a) - dueRank(b)));

    return { columns: activeColumns, members };
  }, [columns, boardMembers]);
}
