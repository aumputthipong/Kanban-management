"use client";

import { BoardGlyph, boardColor } from "@/lib/boardAppearance";
import { CompactRow } from "./CompactRow";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard } from "@/types/myWork";

interface Props {
  cards: MyWorkCard[];
  boardMeta: Map<string, BoardMeta>;
  onOpenCard: (card: MyWorkCard) => void;
  withIcon?: boolean;
  renderCard?: (card: MyWorkCard) => React.ReactNode;
}

export function ProjectGroupedList({ cards, boardMeta, onOpenCard, withIcon = false, renderCard }: Props) {
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
  groups.sort((a, b) => b.cards.length - a.cards.length || a.name.localeCompare(b.name));

  const row = renderCard ?? ((c: MyWorkCard) => <CompactRow key={c.id} card={c} rail onOpenCard={onOpenCard} />);

  return (
    <div className="flex flex-col gap-2">
      {groups.map((g) => {
        const meta = boardMeta.get(g.boardId);
        return (
          <div key={g.boardId} className="rounded-lg border border-slate-200 bg-white px-2 pt-2.5 pb-1.5">
            <div className="flex items-center gap-2 px-1.5 pb-1.5">
              {withIcon ? (
                <span
                  aria-hidden
                  className="w-5 h-5 rounded-md flex items-center justify-center text-white shrink-0"
                  style={{ background: boardColor(meta?.color) }}
                >
                  <BoardGlyph icon={meta?.icon} size={12} />
                </span>
              ) : (
                <span aria-hidden className="w-2 h-2 rounded-full shrink-0" style={{ background: boardColor(meta?.color) }} />
              )}
              <span className="flex-1 min-w-0 truncate text-[13px] font-semibold text-slate-900">{g.name}</span>
              <span className="shrink-0 text-xs text-slate-400 tabular-nums">{g.cards.length} รายการ</span>
            </div>
            <div className="ml-2.5 border-l border-slate-200">{g.cards.map(row)}</div>
          </div>
        );
      })}
    </div>
  );
}
