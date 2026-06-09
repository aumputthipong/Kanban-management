"use client";

import { BoardGlyph, boardColor } from "@/lib/boardAppearance";
import { CompactRow } from "./CompactRow";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard } from "@/types/myWork";

interface Props {
  cards: MyWorkCard[];
  boardMeta: Map<string, BoardMeta>;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string, label: string) => void;
  onOpenCard: (card: MyWorkCard) => void;
  /** Forwarded to CompactRow — drop due/estimate columns. */
  slim?: boolean;
  /** Forwarded to CompactRow — taller hero rows. */
  hero?: boolean;
}

// Groups a panel's cards into project containers (project glyph + name + count,
// then the rows) so each date panel still answers "which project". The caller
// owns the card order; within a project that order is preserved.
export function ProjectGroupedList({
  cards,
  boardMeta,
  onComplete,
  onSnooze,
  onOpenCard,
  slim,
  hero,
}: Props) {
  const groups: { boardId: string; name: string; cards: MyWorkCard[] }[] = [];
  const indexById = new Map<string, number>();
  for (const c of cards) {
    let idx = indexById.get(c.board_id);
    if (idx === undefined) {
      idx = groups.length;
      indexById.set(c.board_id, idx);
      groups.push({ boardId: c.board_id, name: c.board_name, cards: [] });
    }
    groups[idx].cards.push(c);
  }
  // Busiest project first, then alphabetical — stable across re-renders.
  groups.sort((a, b) => b.cards.length - a.cards.length || a.name.localeCompare(b.name));

  return (
    <div>
      {groups.map((g) => {
        const meta = boardMeta.get(g.boardId);
        return (
          <div key={g.boardId} className="border-b border-slate-100 last:border-b-0">
            <div className="flex items-center gap-2 px-[18px] pt-2.5 pb-1.5">
              <span
                aria-hidden
                className="w-5 h-5 rounded flex items-center justify-center text-white shrink-0"
                style={{ background: boardColor(meta?.color) }}
              >
                <BoardGlyph icon={meta?.icon} size={12} />
              </span>
              <span className="text-[12.5px] font-bold text-slate-800 truncate">
                {g.name}
              </span>
              <span className="ml-auto text-[11px] font-bold tabular-nums text-slate-400">
                {g.cards.length}
              </span>
            </div>
            <div>
              {g.cards.map((c) => (
                <CompactRow
                  key={c.id}
                  card={c}
                  slim={slim}
                  hero={hero}
                  onComplete={onComplete}
                  onSnooze={onSnooze}
                  onOpenCard={onOpenCard}
                />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
