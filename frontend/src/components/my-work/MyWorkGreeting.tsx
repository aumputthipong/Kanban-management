"use client";

import { Sun } from "lucide-react";

interface MyWorkGreetingProps {
  fullName?: string | null;
}

function greetingByHour(hour: number): string {
  if (hour < 12) return "สวัสดีตอนเช้า";
  if (hour < 17) return "สวัสดีตอนบ่าย";
  return "สวัสดีตอนเย็น";
}

function thaiDate(now: Date): string {
  const weekday = now.toLocaleDateString("th-TH", { weekday: "long" });
  const day = now.getDate();
  const month = now.toLocaleDateString("th-TH", { month: "short" });
  const year = now.getFullYear() + 543;
  return `${weekday} · ${day} ${month} ${year}`;
}

// The hero stat cards on the right already carry the today/week/overdue counts,
// so the greeting stays minimal: just the date and a calm welcome line. No
// duplicated summary, no colored emphasis — keeps the header uncluttered.
export function MyWorkGreeting({ fullName }: MyWorkGreetingProps) {
  const now = new Date();
  const display = fullName?.split(" ")[0] ?? "คุณ";

  return (
    <header className="min-w-0">
      <div className="flex items-center gap-2 text-xs font-medium text-slate-500 mb-2">
        <Sun size={13} className="text-slate-400" />
        <span className="whitespace-nowrap">{thaiDate(now)}</span>
      </div>
      <h1 className="text-[2rem] leading-tight font-bold tracking-[-0.02em] text-slate-900">
        {greetingByHour(now.getHours())}, {display}
      </h1>
    </header>
  );
}
