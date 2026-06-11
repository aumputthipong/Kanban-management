"use client";

import { useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronLeft, ChevronRight } from "lucide-react";
import { format, isSameMonth } from "date-fns";

interface Props {
  /** The month currently shown by the calendar. */
  currentDate: Date;
  today: Date;
  /** Jump the calendar to the first day of the picked month. */
  onSelect: (date: Date) => void;
}

// Short month labels (Jan…Dec) — derived once from date-fns so they follow the
// app locale instead of being hardcoded English strings.
const MONTHS = Array.from({ length: 12 }, (_, i) => format(new Date(2000, i, 1), "MMM"));

// Click the "MMMM yyyy" title to open a compact picker: a year stepper over a
// 12-month grid. Jumping to a far month/year takes one or two clicks instead of
// paging one month at a time.
export function MonthYearPicker({ currentDate, today, onSelect }: Props) {
  const [open, setOpen] = useState(false);
  // Year shown in the grid — browse years without committing a selection yet.
  const [viewYear, setViewYear] = useState(currentDate.getFullYear());
  const ref = useRef<HTMLDivElement | null>(null);

  // Re-sync the browsed year to the calendar each time the popover opens.
  const openPicker = () => {
    setViewYear(currentDate.getFullYear());
    setOpen(true);
  };

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const pick = (monthIndex: number) => {
    onSelect(new Date(viewYear, monthIndex, 1));
    setOpen(false);
  };

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => (open ? setOpen(false) : openPicker())}
        aria-haspopup="dialog"
        aria-expanded={open}
        className="group flex items-baseline gap-1.5 rounded-md px-2 py-1 text-2xl font-bold tracking-tight text-slate-900 hover:bg-slate-100 hover:text-blue-800"
      >
        {format(currentDate, "MMMM yyyy")}
        <ChevronDown
          size={18}
          className={`self-center text-slate-400 transition-transform group-hover:text-blue-800 ${open ? "rotate-180" : ""}`}
        />
      </button>

      {open && (
        <div
          role="dialog"
          aria-label="เลือกเดือนและปี"
          className="absolute left-0 top-full z-50 mt-2 w-64 rounded-lg border border-slate-200 bg-white p-3 shadow-xl"
        >
          <div className="mb-2 flex items-center justify-between">
            <button
              type="button"
              onClick={() => setViewYear((y) => y - 1)}
              aria-label="ปีก่อนหน้า"
              className="rounded p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="text-sm font-bold tabular-nums text-slate-900">{viewYear}</span>
            <button
              type="button"
              onClick={() => setViewYear((y) => y + 1)}
              aria-label="ปีถัดไป"
              className="rounded p-1 text-slate-500 hover:bg-slate-100 hover:text-slate-700"
            >
              <ChevronRight size={16} />
            </button>
          </div>

          <div className="grid grid-cols-3 gap-1">
            {MONTHS.map((label, i) => {
              const cellDate = new Date(viewYear, i, 1);
              const isSelected = isSameMonth(cellDate, currentDate);
              const isCurrent = isSameMonth(cellDate, today);
              return (
                <button
                  key={label}
                  type="button"
                  onClick={() => pick(i)}
                  className={`rounded-md py-1.5 text-xs font-medium transition-colors ${
                    isSelected
                      ? "bg-blue-800 text-white"
                      : isCurrent
                        ? "text-blue-800 ring-1 ring-blue-200 hover:bg-blue-50"
                        : "text-slate-700 hover:bg-slate-100"
                  }`}
                >
                  {label}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
