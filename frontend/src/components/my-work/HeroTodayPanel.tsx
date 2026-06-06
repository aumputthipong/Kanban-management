"use client";

import { Sun } from "lucide-react";
import { CompactRow } from "./CompactRow";
import type { MyWorkCard } from "@/types/myWork";

interface HeroTodayPanelProps {
  cards: MyWorkCard[];
  /** Tasks completed in this session — drives the daily progress meter. */
  doneToday: number;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string) => void;
  className?: string;
}

export function HeroTodayPanel({
  cards,
  doneToday,
  onComplete,
  onSnooze,
  className = "",
}: HeroTodayPanelProps) {
  const total = cards.length + doneToday;
  const pct = total > 0 ? Math.round((doneToday / total) * 100) : 0;

  return (
    <section
      className={`bg-white border border-blue-200 rounded-xl shadow-sm flex flex-col min-h-0 overflow-hidden ${className}`}
    >
      <div className="flex items-center gap-2.5 px-[18px] pt-[15px] pb-3.5 border-b border-slate-200 bg-blue-50 shrink-0">
        <span
          aria-hidden
          className="w-[26px] h-[26px] rounded-lg bg-white text-blue-700 flex items-center justify-center shrink-0"
        >
          <Sun size={15} />
        </span>
        <div className="min-w-0">
          <div className="text-[11px] font-bold uppercase tracking-wider text-blue-700/80 leading-none mb-0.5">
            โฟกัส
          </div>
          <div className="text-[15px] font-bold tracking-tight text-slate-900">วันนี้ต้องทำ</div>
        </div>
        {total > 0 && (
          <div className="ml-auto flex items-center gap-2.5 shrink-0">
            <span className="text-xs font-bold text-slate-600 whitespace-nowrap">
              <b className="text-blue-700">{doneToday}</b>/{total} เสร็จ
            </span>
            <span className="w-22 h-1.5 rounded-full bg-blue-100 overflow-hidden">
              <span
                className="block h-full rounded-full bg-blue-700 transition-[width] duration-300"
                style={{ width: `${pct}%` }}
              />
            </span>
          </div>
        )}
      </div>

      <div className="flex-1 min-h-0 overflow-auto dash-scroll">
        {cards.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 px-5 py-10 text-center h-full">
            <span className="w-11 h-11 rounded-xl bg-blue-50 text-blue-700 flex items-center justify-center">
              <Sun size={22} />
            </span>
            <span className="text-sm font-bold text-slate-900">
              {doneToday > 0 ? "เคลียร์งานวันนี้หมดแล้ว" : "วันนี้ยังไม่มีงานที่ต้องทำ"}
            </span>
            <span className="text-xs text-slate-400 max-w-[220px]">
              {doneToday > 0
                ? "เยี่ยมมาก — พักหรือหยิบงานในคิวมาทำต่อได้เลย"
                : "กำหนด due date ให้งาน หรือดูงานในคิวด้านขวา"}
            </span>
          </div>
        ) : (
          cards.map((c) => (
            <CompactRow key={c.id} card={c} hero onComplete={onComplete} onSnooze={onSnooze} />
          ))
        )}
      </div>
    </section>
  );
}
