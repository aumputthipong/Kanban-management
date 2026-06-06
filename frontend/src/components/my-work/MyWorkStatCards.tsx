"use client";

import { AlertTriangle, CalendarDays, Sun } from "lucide-react";

interface MyWorkStatCardsProps {
  overdue: number;
  today: number;
  thisWeek: number;
}

interface CardProps {
  value: number;
  label: string;
  icon: React.ReactNode;
  tone: "primary" | "neutral" | "muted";
}

const TONE: Record<CardProps["tone"], { card: string; bar: string; num: string; lbl: string; icon: string }> = {
  // Today is the hero stat — the only one that carries the brand accent.
  // Flat surface-tint wash (no gradient) keeps it calm.
  primary: {
    card: "border-blue-200 bg-blue-50",
    bar: "bg-blue-800",
    num: "text-blue-800",
    lbl: "text-blue-800",
    icon: "",
  },
  neutral: {
    card: "border-slate-200 bg-white",
    bar: "bg-slate-300",
    num: "text-slate-900",
    lbl: "text-slate-600",
    icon: "",
  },
  // Overdue is intentionally muted — a quiet reminder, not an alarm. Only the
  // tiny icon keeps a subtle red tint.
  muted: {
    card: "border-slate-200 bg-white hover:bg-slate-50 transition-colors",
    bar: "bg-slate-300",
    num: "text-slate-600",
    lbl: "text-slate-400",
    icon: "text-red-700/70",
  },
};

function StatCard({ value, label, icon, tone }: CardProps) {
  const t = TONE[tone];
  return (
    <div className={`relative w-[104px] px-3 pt-[11px] pb-2.5 rounded-lg border shadow-sm overflow-hidden ${t.card}`}>
      <span aria-hidden className={`absolute left-0 top-0 bottom-0 w-[3px] ${t.bar}`} />
      <span className={`block text-2xl font-bold leading-none tabular-nums ${t.num}`}>{value}</span>
      <span className={`flex items-center gap-1.5 mt-1.5 text-xs font-semibold whitespace-nowrap ${t.lbl}`}>
        <span className={t.icon}>{icon}</span>
        {label}
      </span>
    </div>
  );
}

export function MyWorkStatCards({ overdue, today, thisWeek }: MyWorkStatCardsProps) {
  return (
    <div className="flex gap-2.5 shrink-0">
      <StatCard value={today} label="วันนี้" tone="primary" icon={<Sun size={13} />} />
      <StatCard value={thisWeek} label="สัปดาห์นี้" tone="neutral" icon={<CalendarDays size={13} />} />
      <StatCard value={overdue} label="เลยกำหนด" tone="muted" icon={<AlertTriangle size={13} />} />
    </div>
  );
}
