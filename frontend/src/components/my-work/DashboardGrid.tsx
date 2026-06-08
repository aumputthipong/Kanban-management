"use client";

import { AlertTriangle, Calendar, CalendarDays, CalendarRange, Inbox, Sun } from "lucide-react";
import { CompactRow } from "./CompactRow";
import { DashboardPanel } from "./DashboardPanel";
import { HeroTodayPanel } from "./HeroTodayPanel";
import { MyWorkEmptyState } from "./MyWorkEmptyState";
import { OverdueStrip } from "./OverdueStrip";
import type { MyWorkCard, MyWorkCounts, MyWorkFilter } from "@/types/myWork";

interface DashboardGridProps {
  filter: MyWorkFilter;
  cards: MyWorkCard[];
  counts: MyWorkCounts;
  doneToday: number;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string, label: string) => void;
}

const FOCUS_META: Record<
  Exclude<MyWorkFilter, "all">,
  { title: string; icon: React.ReactNode; tone: "danger" | "neutral" | "tint"; danger?: boolean; slim?: boolean }
> = {
  overdue: { title: "เลยกำหนด", icon: <AlertTriangle size={13} />, tone: "danger", danger: true },
  today: { title: "วันนี้", icon: <Sun size={13} />, tone: "tint" },
  this_week: { title: "สัปดาห์นี้", icon: <CalendarDays size={13} />, tone: "tint" },
  no_date: { title: "ไม่มีวันที่", icon: <Inbox size={13} />, tone: "neutral", slim: true },
};

export function DashboardGrid({
  filter,
  cards,
  counts,
  doneToday,
  onComplete,
  onSnooze,
}: DashboardGridProps) {
  const rowProps = { onComplete, onSnooze };

  // ── Focused single-group view (any chip other than "ทั้งหมด") ──
  if (filter !== "all") {
    const meta = FOCUS_META[filter];
    return (
      <div className="grid grid-cols-1 min-h-0 lg:flex-1">
        <DashboardPanel
          icon={meta.icon}
          iconTone={meta.tone}
          title={meta.title}
          danger={meta.danger}
          count={cards.length}
          className="min-h-0 dash-reveal d3"
        >
          {cards.length === 0 ? (
            <div className="p-6">
              <MyWorkEmptyState filter={filter} />
            </div>
          ) : (
            cards.map((c) => <CompactRow key={c.id} card={c} slim={meta.slim} {...rowProps} />)
          )}
        </DashboardPanel>
      </div>
    );
  }

  // ── Overview (ทั้งหมด): Today hero is primary; overdue is a quiet strip ──
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
          className="dash-reveal d3 flex-1"
          {...rowProps}
        />
        <OverdueStrip cards={overdue} className="dash-reveal d4" {...rowProps} />
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
            upcoming.map((c) => <CompactRow key={c.id} card={c} {...rowProps} />)
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
            noDate.map((c) => <CompactRow key={c.id} card={c} slim {...rowProps} />)
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
