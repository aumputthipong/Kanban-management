"use client";

import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import { isSameMonth } from "date-fns";
import { MonthYearPicker } from "./MonthYearPicker";

export type CalendarView = "day" | "week" | "month" | "agenda";

interface Props {
  currentDate: Date;
  today: Date;
  view: CalendarView;
  onPrev: () => void;
  onNext: () => void;
  onToday: () => void;
  /** Jump the calendar to a chosen month/year (from the title picker). */
  onSelectDate: (date: Date) => void;
  onViewChange: (v: CalendarView) => void;
  onNewTask?: () => void;
}

const VIEWS: { key: CalendarView; label: string; enabled: boolean }[] = [
  { key: "day", label: "Day", enabled: false },
  { key: "week", label: "Week", enabled: false },
  { key: "month", label: "Month", enabled: true },
  { key: "agenda", label: "Agenda", enabled: false },
];

export function CalendarHeader({
  currentDate,
  today,
  view,
  onPrev,
  onNext,
  onToday,
  onSelectDate,
  onViewChange,
  onNewTask,
}: Props) {
  const isViewingCurrentMonth = isSameMonth(currentDate, today);

  return (
    <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
      {/* Month navigator — prev / picker / next / Today all in one cluster so
          changing month is obvious and lives in a single place. */}
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onPrev}
          aria-label="Previous month"
          className="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
        >
          <ChevronLeft size={20} />
        </button>
        <MonthYearPicker currentDate={currentDate} today={today} onSelect={onSelectDate} />
        <button
          type="button"
          onClick={onNext}
          aria-label="Next month"
          className="rounded-md p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
        >
          <ChevronRight size={20} />
        </button>
        <button
          type="button"
          onClick={onToday}
          disabled={isViewingCurrentMonth}
          className="ml-1 rounded-md border border-slate-200 px-3 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-400 disabled:hover:bg-white"
        >
          Today
        </button>
      </div>

      <div className="flex items-center gap-2">
        <div role="tablist" className="flex rounded-md border border-slate-200 bg-white p-0.5">
          {VIEWS.map((v) => {
            const active = v.key === view;
            return (
              <button
                key={v.key}
                role="tab"
                aria-selected={active}
                disabled={!v.enabled}
                onClick={() => v.enabled && onViewChange(v.key)}
                title={v.enabled ? v.label : `${v.label} — coming soon`}
                className={`rounded px-3 py-1 text-xs font-medium transition-colors ${
                  active
                    ? "bg-slate-900 text-white"
                    : v.enabled
                      ? "text-slate-600 hover:bg-slate-50"
                      : "cursor-not-allowed text-slate-300"
                }`}
              >
                {v.label}
              </button>
            );
          })}
        </div>

        {onNewTask && (
          <button
            type="button"
            onClick={onNewTask}
            className="inline-flex items-center gap-1 rounded-md bg-blue-800 px-3 py-1.5 text-xs font-semibold text-white hover:bg-blue-900"
          >
            <Plus size={14} />
            New Task
          </button>
        )}
      </div>
    </div>
  );
}
