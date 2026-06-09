"use client";

import { useState } from "react";
import { AlertTriangle, ChevronDown } from "lucide-react";
import { ProjectGroupedList } from "./ProjectGroupedList";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard } from "@/types/myWork";

interface OverdueStripProps {
  cards: MyWorkCard[];
  boardMeta: Map<string, BoardMeta>;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string, label: string) => void;
  onOpenCard: (card: MyWorkCard) => void;
  className?: string;
}

/**
 * Overdue lives here as a quiet, collapsed-by-default strip — present and
 * countable, but it never dominates the page (per the Today-first hierarchy).
 * Red is reduced to a small badge + icon, not a full alarm section.
 */
export function OverdueStrip({ cards, boardMeta, onComplete, onSnooze, onOpenCard, className = "" }: OverdueStripProps) {
  const [open, setOpen] = useState(false);
  if (cards.length === 0) return null;

  const preview = cards.slice(0, 2).map((c) => c.title).join(" · ");
  const rest = cards.length - 2;
  const summary = rest > 0 ? `${preview} · & อีก ${rest} งาน` : preview;

  return (
    <div
      className={`shrink-0 border border-slate-200 rounded-xl bg-white shadow-sm overflow-hidden ${className}`}
    >
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex items-center gap-2.5 w-full px-4 py-3 text-left hover:bg-slate-50 transition-colors"
      >
        <span
          aria-hidden
          className="w-[22px] h-[22px] rounded-lg bg-red-50 text-red-700 flex items-center justify-center shrink-0"
        >
          <AlertTriangle size={13} />
        </span>
        <span className="text-[13.5px] font-bold text-slate-600 whitespace-nowrap">เลยกำหนด</span>
        <span className="inline-flex items-center justify-center min-w-5 h-[19px] px-1.5 rounded-full bg-red-700 text-white text-[11px] font-bold tabular-nums shrink-0">
          {cards.length}
        </span>
        <span className="flex-1 min-w-0 text-xs font-medium text-slate-400 truncate">{summary}</span>
        <span className="inline-flex items-center gap-1 text-xs font-semibold text-blue-700 whitespace-nowrap shrink-0">
          {open ? "ย่อ" : "ดูทั้งหมด"}
          <ChevronDown size={14} className={`transition-transform ${open ? "rotate-180" : ""}`} />
        </span>
      </button>

      {open && (
        <div className="max-h-[230px] overflow-auto dash-scroll border-t border-slate-100">
          <ProjectGroupedList
            cards={cards}
            boardMeta={boardMeta}
            onComplete={onComplete}
            onSnooze={onSnooze}
            onOpenCard={onOpenCard}
          />
        </div>
      )}
    </div>
  );
}
