"use client";

import { Users } from "lucide-react";
import type { Card } from "@/types/board";
import { useBoardOwnership } from "@/hooks/useBoardOwnership";
import { getColumnColorHex } from "@/components/board/task-board/ColumnOptionsModal";
import { MemberOwnershipRow } from "./MemberOwnershipRow";

// Neutral dot for a column with no explicit colour set (Default).
const DOT_FALLBACK = "#cbd5e1";

interface TeamOwnershipListProps {
  onSelectCard: (card: Card) => void;
}

export function TeamOwnershipList({ onSelectCard }: TeamOwnershipListProps) {
  const { columns, members } = useBoardOwnership();

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="flex items-center gap-2.5 border-b border-slate-100 px-4 py-3.5">
        <span className="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-50 text-blue-800">
          <Users size={16} />
        </span>
        <h2 className="text-sm font-bold tracking-tight text-slate-900">ใครถืออะไรอยู่</h2>
        <span className="ml-auto text-xs font-semibold text-slate-400">{members.length} คน</span>
      </div>

      {members.length === 0 ? (
        <p className="px-4 py-8 text-center text-sm text-slate-400">
          ยังไม่มีสมาชิกในบอร์ดนี้
        </p>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr className="bg-slate-50/70">
                  <th className="px-4 py-2 text-left text-[10.5px] font-bold uppercase tracking-wide text-slate-400">
                    สมาชิก
                  </th>
                  {columns.map((col) => (
                    <th
                      key={col.id}
                      className="whitespace-nowrap px-2 py-2 text-[10.5px] font-bold uppercase tracking-wide text-slate-400"
                    >
                      <span className="inline-flex items-center gap-1.5">
                        <span
                          className="h-[7px] w-[7px] rounded-sm"
                          style={{ backgroundColor: getColumnColorHex(col.color) ?? DOT_FALLBACK }}
                        />
                        {col.title}
                      </span>
                    </th>
                  ))}
                  <th className="px-4 py-2 text-right text-[10.5px] font-bold uppercase tracking-wide text-slate-400">
                    รวม
                  </th>
                </tr>
              </thead>
              <tbody>
                {members.map((member) => (
                  <MemberOwnershipRow
                    key={member.userId}
                    member={member}
                    columns={columns}
                    onSelectCard={onSelectCard}
                  />
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-slate-100 bg-slate-50/70 px-4 py-2.5">
            <span className="text-[11px] text-slate-400">
              นับเฉพาะงานที่ยังไม่เสร็จและมีผู้รับผิดชอบ
            </span>
            <span className="ml-auto flex flex-wrap items-center gap-3">
              {columns.map((col) => (
                <span
                  key={col.id}
                  className="inline-flex items-center gap-1.5 text-[11px] font-semibold text-slate-500"
                >
                  <span
                    className="h-2.5 w-2.5 rounded-sm"
                    style={{ backgroundColor: getColumnColorHex(col.color) ?? DOT_FALLBACK }}
                  />
                  {col.title}
                </span>
              ))}
            </span>
          </div>
        </>
      )}
    </section>
  );
}
