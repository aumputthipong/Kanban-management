"use client";

import { AlertTriangle, CalendarDays, Sun } from "lucide-react";
import type { MyWorkNoteTab } from "@/types/myWork";

interface MyWorkStatCardsProps {
  overdue: number;
  today: number;
  thisWeek: number;
  activeTab: MyWorkNoteTab;
  onSelectTab: (tab: MyWorkNoteTab) => void;
}

const TILE = "flex min-w-26 flex-col items-start gap-1 rounded-lg border bg-white px-4 py-3 text-left";
const FOCUS = "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary";

// Outlined tiles on the white container. Today and Overdue are shortcuts to the
// matching note tab; the open tab's tile carries the primary ring.
export function MyWorkStatCards({ overdue, today, thisWeek, activeTab, onSelectTab }: MyWorkStatCardsProps) {
  const tabTile = (tab: MyWorkNoteTab) =>
    `${TILE} ${FOCUS} transition-colors ${
      activeTab === tab ? "border-primary ring-1 ring-primary" : "border-slate-200 hover:border-slate-300"
    }`;

  return (
    <div className="flex gap-3 shrink-0">
      <button type="button" aria-pressed={activeTab === "today"} onClick={() => onSelectTab("today")} className={tabTile("today")}>
        <span className="text-2xl leading-8 font-bold tabular-nums text-primary">{today}</span>
        <Label icon={<Sun size={13} />} text="วันนี้" />
      </button>
      <div className={`${TILE} border-slate-200`}>
        <span className="text-2xl leading-8 font-bold tabular-nums text-slate-900">{thisWeek}</span>
        <Label icon={<CalendarDays size={13} />} text="สัปดาห์นี้" />
      </div>
      <button
        type="button"
        aria-pressed={activeTab === "overdue"}
        onClick={() => onSelectTab("overdue")}
        className={tabTile("overdue")}
      >
        <span className={`text-2xl leading-8 font-bold tabular-nums ${overdue > 0 ? "text-danger" : "text-slate-900"}`}>
          {overdue}
        </span>
        <Label icon={<AlertTriangle size={13} />} text="เลยกำหนด" />
      </button>
    </div>
  );
}

function Label({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <span className="flex items-center gap-1.5 text-xs font-medium text-slate-600 whitespace-nowrap">
      <span aria-hidden className="text-slate-400">
        {icon}
      </span>
      {text}
    </span>
  );
}
