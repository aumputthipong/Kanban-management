"use client";

import { AlertTriangle, CalendarDays, Inbox, ListChecks, Sun } from "lucide-react";
import type { MyWorkCounts, MyWorkFilter } from "@/types/myWork";

interface FilterChipBarProps {
  active: MyWorkFilter;
  counts: MyWorkCounts;
  onChange: (next: MyWorkFilter) => void;
}

interface ChipDef {
  key: MyWorkFilter;
  label: string;
  count: number;
  icon: React.ReactNode;
}

// Today leads; overdue is pushed last and de-emphasized (no red) so it stays a
// quiet reminder rather than dominating the bar.
export function FilterChipBar({ active, counts, onChange }: FilterChipBarProps) {
  const chips: ChipDef[] = [
    {
      key: "today",
      label: "วันนี้",
      count: counts.today,
      icon: <Sun size={12} />,
    },
    {
      key: "this_week",
      label: "สัปดาห์นี้",
      count: counts.this_week,
      icon: <CalendarDays size={12} />,
    },
    {
      key: "no_date",
      label: "ไม่มีวันที่",
      count: counts.no_date,
      icon: <Inbox size={12} />,
    },
    {
      key: "all",
      label: "ทั้งหมด",
      count: counts.total,
      icon: <ListChecks size={12} />,
    },
    {
      key: "overdue",
      label: "เลยกำหนด",
      count: counts.overdue,
      icon: <AlertTriangle size={12} />,
    },
  ];

  return (
    <div className="flex items-center gap-2 flex-wrap" role="tablist">
      {chips.map((c) => {
        const isActive = active === c.key;
        return (
          <button
            key={c.key}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(c.key)}
            className={`inline-flex items-center gap-1.5 h-8 px-3 rounded-full border text-xs font-semibold whitespace-nowrap transition-colors ${
              isActive
                ? "bg-blue-800 text-white border-blue-800 shadow-sm"
                : "bg-white text-slate-600 border-slate-200 hover:text-slate-900 hover:border-slate-300"
            }`}
          >
            {c.icon}
            <span>{c.label}</span>
            <span
              className={`tabular-nums text-[11px] font-bold min-w-5 px-1.5 py-0.5 rounded-full text-center ${
                isActive ? "bg-white/20 text-white" : "bg-slate-100 text-slate-500"
              }`}
            >
              {c.count}
            </span>
          </button>
        );
      })}
    </div>
  );
}
