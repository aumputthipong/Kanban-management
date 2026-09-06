"use client";

import { CheckCircle2 } from "lucide-react";
import type { PlanningItem } from "@/types/planning";

export interface SessionStats {
  REQ: number;
  DEC: number;
  Q: number;
  dropped: number;
  promoted: number;
  selected: number;
}

interface Props {
  stats: SessionStats;
  promotedItems: PlanningItem[];
}

// Right rail: a summary panel (total + per-type breakdown) above the list of titles
// already promoted. The breakdown repeats the chip counts on purpose — a labelled
// legend reads as a summary, not a filter control.
export function SessionSidebar({ stats, promotedItems }: Props) {
  // Matches the "All" filter: live plus promoted. Paused items get their own line.
  const total = stats.REQ + stats.DEC + stats.Q + stats.promoted;

  return (
    <aside className="w-full shrink-0 lg:w-64">
      <div className="rounded-lg border border-slate-200 p-4">
        <p className="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400">
          สรุปรายการ
        </p>
        <div className="mb-3 flex items-baseline gap-2">
          <span className="text-3xl font-bold tracking-tight text-slate-900 tabular-nums">
            {total}
          </span>
          <span className="text-xs text-slate-500">รายการในบันทึกนี้</span>
        </div>
        <ul className="space-y-1.5 text-xs">
          <SidebarCount label="สิ่งที่อยากได้" value={stats.REQ} dotClass="bg-red-500" />
          <SidebarCount label="ที่ตกลงแล้ว" value={stats.DEC} dotClass="bg-blue-500" />
          <SidebarCount
            label="คำถามค้าง · รอตอบ"
            value={stats.Q}
            dotClass="bg-amber-500"
          />
          <SidebarCount
            label="พักไว้ก่อน"
            value={stats.dropped}
            dotClass="bg-slate-300"
          />
        </ul>
      </div>

      {promotedItems.length > 0 && (
        <div className="mt-3 rounded-lg border border-slate-200 p-4">
          <p className="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400">
            ส่งเข้า Board แล้ว · {promotedItems.length}
          </p>
          <ul className="space-y-1.5 text-xs">
            {promotedItems.slice(0, 6).map((it) => (
              <li key={it.id} className="flex items-center gap-1.5 text-slate-600">
                <CheckCircle2 size={13} className="shrink-0 text-emerald-500" />
                <span className="truncate">{it.title}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </aside>
  );
}

function SidebarCount({
  label,
  value,
  dotClass,
}: {
  label: string;
  value: number;
  dotClass: string;
}) {
  return (
    <li className="flex items-center justify-between text-slate-600">
      <span className="flex items-center gap-2">
        <span className={`h-1.5 w-1.5 rounded-full ${dotClass}`} />
        {label}
      </span>
      <span className="font-semibold text-slate-800 tabular-nums">{value}</span>
    </li>
  );
}
