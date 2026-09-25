import { useMemo } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import type { Card } from "@/types/board";

// Derived from the store, no endpoint — see AGENTS.md "Ownership View Pattern".
export interface OwnershipColumn {
  id: string;
  title: string;
  position: number;
  color: string | null;
}

export interface MemberOwnership {
  userId: string;
  name: string;
  totalHeld: number;
  countByColumn: Record<string, number>;
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
      if (col.category === "DONE") return;
      col.cards.forEach((card) => {
        if (card.is_done || !card.assignee_id) return;
        let entry = byUser.get(card.assignee_id);
        if (!entry) {
          // Assignee has left the board — keep them visible.
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
