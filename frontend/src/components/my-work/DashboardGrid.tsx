"use client";

import { Calendar, CalendarDays, CalendarRange, Inbox } from "lucide-react";
import { MyWorkEmptyState } from "./MyWorkEmptyState";
import { ProjectGroupedList } from "./ProjectGroupedList";
import { TaskNote } from "./TaskNote";
import type { BoardMeta } from "./boardMeta";
import { Skeleton } from "@/components/ui/Skeleton";
import type { MyWorkCard, MyWorkCounts, MyWorkNoteTab } from "@/types/myWork";

const MAIN_CELL = "min-h-0 flex flex-col px-6 py-6 lg:px-8 lg:col-start-1 lg:row-start-1";
const RAIL_CELL =
  "min-h-0 bg-white border-t border-slate-200 px-5 py-6 lg:border-t-0 lg:border-l lg:overflow-y-auto dash-scroll lg:col-start-2 lg:row-start-1";

interface DashboardGridProps {
  header: React.ReactNode;
  cards: MyWorkCard[];
  counts: MyWorkCounts;
  boardMeta: Map<string, BoardMeta>;
  doneToday: number;
  onOpenCard: (card: MyWorkCard) => void;
  onComplete: (cardId: string) => void;
  tab: MyWorkNoteTab;
  onTabChange: (tab: MyWorkNoteTab) => void;
}

export function DashboardGrid({
  header,
  cards,
  counts,
  boardMeta,
  doneToday,
  onOpenCard,
  onComplete,
  tab,
  onTabChange,
}: DashboardGridProps) {
  if (cards.length === 0) {
    return (
      <div className={`${MAIN_CELL} lg:col-span-2`}>
        <section className="relative rounded-2xl border border-slate-200 bg-white overflow-hidden">
          <span aria-hidden className="absolute inset-x-0 top-0 h-0.75 bg-slate-900" />
          <div className="px-6 py-5">{header}</div>
          <div className="border-t border-slate-200 p-6">
            <MyWorkEmptyState filter="all" />
          </div>
        </section>
      </div>
    );
  }

  const today = cards.filter((c) => c.group === "today");
  const overdue = cards.filter((c) => c.group === "overdue");
  const upcoming = cards
    .filter((c) => c.group === "this_week" || c.group === "later")
    .sort((a, b) => (a.due_date ?? "").localeCompare(b.due_date ?? ""));
  const noDate = cards.filter((c) => c.group === "no_date");

  return (
    <>
      <div className={MAIN_CELL}>
        <TaskNote
          header={header}
          tab={tab}
          onTabChange={onTabChange}
          today={today}
          overdue={overdue}
          doneToday={doneToday}
          boardMeta={boardMeta}
          onOpenCard={onOpenCard}
          onComplete={onComplete}
        />
      </div>

      <aside className={RAIL_CELL}>
        <RailSection icon={<Calendar size={15} />} title="กำหนดส่งที่จะถึง" count={upcoming.length}>
          {upcoming.length > 0 ? (
            <ProjectGroupedList cards={upcoming} boardMeta={boardMeta} onOpenCard={onOpenCard} />
          ) : (
            <div className="flex flex-col gap-2">
              <ClearedRow icon={<CalendarDays size={14} />} label="สัปดาห์นี้" />
              <ClearedRow icon={<CalendarRange size={14} />} label="เดือนนี้" />
            </div>
          )}
        </RailSection>

        <RailSection icon={<Inbox size={15} />} title="ไม่มีวันที่" count={counts.no_date} className="mt-7">
          {noDate.length > 0 ? (
            <ProjectGroupedList cards={noDate} boardMeta={boardMeta} onOpenCard={onOpenCard} />
          ) : (
            <p className="text-[13px] text-slate-400">ทุกงานมีกำหนดส่งแล้ว</p>
          )}
        </RailSection>
      </aside>
    </>
  );
}

export function DashboardGridLoading() {
  return (
    <>
      <div className={MAIN_CELL}>
        <section className="relative rounded-2xl border border-slate-200 bg-white overflow-hidden">
          <span aria-hidden className="absolute inset-x-0 top-0 h-0.75 bg-slate-900" />
          <div className="flex flex-wrap items-center justify-between gap-6 px-6 py-5">
            <div className="flex-1 basis-72 space-y-2">
              <Skeleton className="h-3.5 w-40" />
              <Skeleton className="h-8 w-80" />
              <Skeleton className="h-2 max-w-md mt-4 rounded-full" />
            </div>
            <div className="flex gap-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-18.5 w-26 rounded-lg" />
              ))}
            </div>
          </div>
          <div className="border-t border-slate-200 px-6 py-3">
            <Skeleton className="h-6 w-56" />
          </div>
          <div className="border-t border-slate-200 p-6 space-y-3">
            <Skeleton className="h-24 rounded-lg" />
            <Skeleton className="h-24 rounded-lg" />
          </div>
        </section>
      </div>
      <aside className={RAIL_CELL}>
        <Skeleton className="h-4 w-32 mb-3" />
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10 rounded-lg" />
          <Skeleton className="h-10 rounded-lg" />
        </div>
        <Skeleton className="h-4 w-24 mt-7 mb-3" />
        <Skeleton className="h-28 rounded-lg" />
      </aside>
    </>
  );
}

function RailSection({
  icon,
  title,
  count,
  className = "",
  children,
}: {
  icon: React.ReactNode;
  title: string;
  count: number;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <section className={className}>
      <h2 className="flex items-center gap-2 mb-3 text-sm font-semibold text-slate-900">
        <span aria-hidden className="text-slate-500">
          {icon}
        </span>
        {title}
        {count > 0 && (
          <span className="ml-auto rounded-full bg-blue-50 px-2 py-0.5 text-xs font-semibold tabular-nums text-blue-700">
            {count} งาน
          </span>
        )}
      </h2>
      {children}
    </section>
  );
}

function ClearedRow({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <div className="flex items-center gap-2.5 h-10 px-3 rounded-lg border border-slate-200 bg-slate-50 text-[13px] text-slate-600">
      <span aria-hidden className="text-slate-400">
        {icon}
      </span>
      {label}
      <span className="ml-auto rounded-sm bg-slate-200/70 px-2 py-0.5 text-xs text-slate-500">ไม่มีคิว</span>
    </div>
  );
}
