"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { isSameMonth } from "date-fns";
import { MonthYearPicker } from "./MonthYearPicker";

interface Props {
  currentDate: Date;
  today: Date;
  onPrev: () => void;
  onNext: () => void;
  onToday: () => void;
  /** Jump the calendar to a chosen month/year (from the title picker). */
  onSelectDate: (date: Date) => void;
}

// The calendar only renders a month grid. Day/Week/Agenda view switching was
// never implemented, so there is no view selector here — just month navigation.
export function CalendarHeader({
  currentDate,
  today,
  onPrev,
  onNext,
  onToday,
  onSelectDate,
}: Props) {
  const isViewingCurrentMonth = isSameMonth(currentDate, today);

  return (
    <div className="mb-4 flex flex-wrap items-center gap-2">
      {/* Month navigator — prev / picker / next / Today all in one cluster so
          changing month is obvious and lives in a single place. */}
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
  );
}
