"use client";

import { Calendar, CalendarDays, CalendarRange, Inbox } from "lucide-react";
import { DashboardPanel } from "./DashboardPanel";
import { HeroTodayPanel } from "./HeroTodayPanel";
import { MyWorkEmptyState } from "./MyWorkEmptyState";
import { OverdueStrip } from "./OverdueStrip";
import { ProjectGroupedList } from "./ProjectGroupedList";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard, MyWorkCounts } from "@/types/myWork";

interface DashboardGridProps {
  cards: MyWorkCard[];
  counts: MyWorkCounts;
  boardMeta: Map<string, BoardMeta>;
  doneToday: number;
  onOpenCard: (card: MyWorkCard) => void;
}

export function DashboardGrid({
  cards,
  counts,
  boardMeta,
  doneToday,
  onOpenCard,
}: DashboardGridProps) {
  const rowProps = { onOpenCard };

  // Whole inbox empty — one calm empty state instead of four empty panels.
  if (cards.length === 0) {
    return (
      <div className="min-h-0 lg:flex-1 flex items-center justify-center dash-reveal d3">
        <MyWorkEmptyState filter="all" />
      </div>
    );
  }

  // Today hero is primary; overdue is a quiet strip; the rail holds the rest.
  const today = cards.filter((c) => c.group === "today");
  const overdue = cards.filter((c) => c.group === "overdue");
  const upcoming = cards
    .filter((c) => c.group === "this_week" || c.group === "later")
    .sort((a, b) => (a.due_date ?? "").localeCompare(b.due_date ?? ""));
  const noDate = cards.filter((c) => c.group === "no_date");

  return (
    <div className="grid gap-[18px] min-h-0 lg:flex-1 grid-cols-1 lg:[grid-template-columns:minmax(0,1.9fr)_minmax(300px,1fr)]">
      {/* LEFT: Today (primary) + collapsed overdue */}
      <div className="flex flex-col gap-3.5 min-h-0">
        <HeroTodayPanel
          cards={today}
          doneToday={doneToday}
          boardMeta={boardMeta}
          className="dash-reveal d3 flex-1"
          {...rowProps}
        />
        <OverdueStrip
          cards={overdue}
          boardMeta={boardMeta}
          className="dash-reveal d4"
          {...rowProps}
        />
      </div>

      {/* RIGHT RAIL: secondary + tertiary */}
      <div className="grid gap-[18px] min-h-0 lg:[grid-template-rows:auto_minmax(0,1fr)]">
        <DashboardPanel
          icon={<Calendar size={13} />}
          iconTone="tint"
          title="กำหนดส่งที่จะถึง"
          count={upcoming.length > 0 ? upcoming.length : undefined}
          scrollBody={upcoming.length > 0}
          className="dash-reveal d3"
        >
          {upcoming.length > 0 ? (
            <ProjectGroupedList cards={upcoming} boardMeta={boardMeta} {...rowProps} />
          ) : (
            <div>
              <UpcomingClearedRow icon={<CalendarDays size={15} />} label="สัปดาห์นี้" />
              <UpcomingClearedRow icon={<CalendarRange size={15} />} label="เดือนนี้" />
            </div>
          )}
        </DashboardPanel>

        <DashboardPanel
          icon={<Inbox size={13} />}
          iconTone="neutral"
          title="ไม่มีวันที่"
          count={counts.no_date}
          className="min-h-0 dash-reveal d4"
        >
          {noDate.length === 0 ? (
            <ClearedState text="ทุกงานมีกำหนดส่งแล้ว" sub="ไม่มีงานค้างไว้โดยไม่มีวันที่" />
          ) : (
            <ProjectGroupedList cards={noDate} boardMeta={boardMeta} slim {...rowProps} />
          )}
        </DashboardPanel>
      </div>
    </div>
  );
}

function ClearedState({ text, sub }: { text: string; sub: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-5 py-8 text-center h-full">
      <span className="w-10 h-10 rounded-xl bg-emerald-50 text-emerald-700 flex items-center justify-center">
        <Inbox size={20} />
      </span>
      <span className="text-sm font-bold text-slate-900">{text}</span>
      <span className="text-xs text-slate-400 max-w-[200px]">{sub}</span>
    </div>
  );
}

function UpcomingClearedRow({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <div className="flex items-center justify-between px-[18px] py-3 border-b border-slate-100 last:border-b-0">
      <span className="flex items-center gap-2.5 text-[13px] font-semibold text-slate-600">
        <span className="text-slate-400">{icon}</span>
        {label}
      </span>
      <span className="text-xs font-bold text-emerald-700 bg-emerald-50 rounded-full px-2.5 py-0.5 whitespace-nowrap">
        ไม่มีคิว
      </span>
    </div>
  );
}
