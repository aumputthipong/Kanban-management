"use client";

import { useMemo, useState } from "react";
import { ChevronRight } from "lucide-react";
import type { Card } from "@/types/board";
import type { MemberOwnership, OwnershipColumn } from "@/hooks/useBoardOwnership";
import { useBoardStore } from "@/store/useBoardStore";
import { getColumnColorHex } from "@/components/board/task-board/ColumnOptionsModal";
import { avatarColor, initials } from "./activityFormat";
import { HeldCardRow } from "./HeldCardRow";

// Colour for a column with no explicit colour set (Default) — neutral slate.
const BAR_FALLBACK = "#cbd5e1";
// Dashed placeholder bar for members holding nothing.
const EMPTY_BAR =
  "repeating-linear-gradient(90deg,#EDF0F5,#EDF0F5 6px,transparent 6px,transparent 12px)";

interface MemberOwnershipRowProps {
  member: MemberOwnership;
  columns: OwnershipColumn[];
  /** Per-column number cells (only in the 2..MAX "sweet spot"). */
  numericMode: boolean;
  /** Mini distribution bar (>=2 columns; a single segment conveys nothing). */
  showBar: boolean;
  onSelectCard: (card: Card) => void;
}

export function MemberOwnershipRow({ member, columns, numericMode, showBar, onSelectCard }: MemberOwnershipRowProps) {
  const idle = member.totalHeld === 0;
  const [expanded, setExpanded] = useState(false);
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const isYou = member.userId === currentUserId;

  const columnTitleById = useMemo(
    () => new Map(columns.map((c) => [c.id, c.title])),
    [columns],
  );

  return (
    <>
      <tr
        className={`border-t border-slate-100 transition-colors ${
          idle ? "" : "cursor-pointer hover:bg-slate-50/70"
        }`}
        onClick={idle ? undefined : () => setExpanded((v) => !v)}
        aria-expanded={idle ? undefined : expanded}
      >
        {/* member + mini workload bar */}
        <td className="py-3 pl-4 pr-3">
          <div className="flex items-center gap-2.5">
            <ChevronRight
              size={15}
              className={`shrink-0 text-slate-300 transition-transform ${
                idle ? "invisible" : ""
              } ${expanded ? "rotate-90" : ""}`}
            />
            <span
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-[13px] font-bold ${avatarColor(member.name)}`}
            >
              {initials(member.name)}
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="truncate text-sm font-semibold text-slate-900">
                  {member.name}
                </span>
                {isYou && (
                  <span className="shrink-0 rounded-sm bg-indigo-50 px-1.5 py-px text-[10px] font-bold uppercase tracking-wide text-blue-800">
                    คุณ
                  </span>
                )}
              </div>
              {showBar &&
                (idle ? (
                  <div
                    className={`mt-1.5 h-1.5 rounded-full ${numericMode ? "max-w-[240px]" : "max-w-[460px]"}`}
                    style={{ backgroundImage: EMPTY_BAR }}
                  />
                ) : (
                  <div
                    className={`mt-1.5 flex h-1.5 overflow-hidden rounded-full bg-slate-100 ${numericMode ? "max-w-[240px]" : "max-w-[460px]"}`}
                  >
                    {columns.map((col) => {
                      const count = member.countByColumn[col.id] ?? 0;
                      if (count === 0) return null;
                      return (
                        <span
                          key={col.id}
                          title={`${col.title} · ${count}`}
                          style={{
                            width: `${(count / member.totalHeld) * 100}%`,
                            backgroundColor: getColumnColorHex(col.color) ?? BAR_FALLBACK,
                          }}
                        />
                      );
                    })}
                  </div>
                ))}
            </div>
          </div>
        </td>

        {/* per-column counts — only in the 2..MAX sweet spot */}
        {numericMode &&
          columns.map((col) => {
            const count = member.countByColumn[col.id] ?? 0;
            return (
              <td key={col.id} className="px-2 py-3 text-center">
                {count === 0 ? (
                  <span className="text-slate-300">–</span>
                ) : (
                  <span className="text-sm font-semibold tabular-nums text-slate-900">
                    {count}
                  </span>
                )}
              </td>
            );
          })}

        {/* total */}
        <td className="py-3 pl-3 pr-4 text-right">
          {idle ? (
            <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-xs font-bold text-emerald-700">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
              ว่าง
            </span>
          ) : (
            <span className="inline-flex items-baseline gap-1 rounded-full bg-slate-900 px-2.5 py-1 text-[13px] font-bold tabular-nums text-white">
              {member.totalHeld}
              <span className="text-[11px] font-semibold text-white/70">งาน</span>
            </span>
          )}
        </td>
      </tr>

      {expanded && !idle && (
        <tr className="border-t border-slate-100 bg-slate-50/50">
          <td colSpan={numericMode ? columns.length + 2 : 2} className="px-2 py-1.5 pl-8">
            <div className="flex flex-col">
              {member.cards.map((card) => (
                <HeldCardRow
                  key={card.id}
                  card={card}
                  columnTitle={columnTitleById.get(card.column_id) ?? ""}
                  onSelect={onSelectCard}
                />
              ))}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}
