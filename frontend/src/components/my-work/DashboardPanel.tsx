"use client";

import { ArrowRight } from "lucide-react";

type IconTone = "danger" | "neutral" | "tint";

interface DashboardPanelProps {
  icon: React.ReactNode;
  iconTone: IconTone;
  title: string;
  danger?: boolean;
  count?: number;
  /** Optional header action rendered on the right ("ดูทั้งหมด →"). */
  link?: { label: string; onClick: () => void };
  /** When false, the body is rendered flush (no internal scroll) — used by the
   *  upcoming panel's "cleared" summary rows. */
  scrollBody?: boolean;
  className?: string;
  children: React.ReactNode;
}

const ICON_TONE: Record<IconTone, string> = {
  danger: "bg-red-50 text-red-700",
  neutral: "bg-slate-100 text-slate-600",
  tint: "bg-blue-50 text-blue-700",
};

export function DashboardPanel({
  icon,
  iconTone,
  title,
  danger = false,
  count,
  link,
  scrollBody = true,
  className = "",
  children,
}: DashboardPanelProps) {
  return (
    <section
      className={`bg-white border border-slate-200 rounded-xl shadow-sm flex flex-col min-h-0 overflow-hidden ${className}`}
    >
      <div className="flex items-center gap-2.5 px-[18px] py-3 border-b border-slate-100 shrink-0">
        <span
          aria-hidden
          className={`w-[22px] h-[22px] rounded-lg flex items-center justify-center shrink-0 ${ICON_TONE[iconTone]}`}
        >
          {icon}
        </span>
        <span className={`text-sm font-bold tracking-tight whitespace-nowrap ${danger ? "text-red-700" : "text-slate-900"}`}>
          {title}
        </span>
        {count != null && (
          <span className="inline-flex items-center justify-center min-w-[21px] h-[19px] px-1.5 rounded-full bg-slate-100 text-[11px] font-bold tabular-nums text-slate-500">
            {count}
          </span>
        )}
        {link && (
          <button
            type="button"
            onClick={link.onClick}
            className="ml-auto inline-flex items-center gap-1 text-xs font-semibold text-blue-700 hover:gap-1.5 transition-all whitespace-nowrap"
          >
            {link.label}
            <ArrowRight size={14} />
          </button>
        )}
      </div>
      <div className={`flex-1 min-h-0 ${scrollBody ? "overflow-auto dash-scroll" : ""}`}>
        {children}
      </div>
    </section>
  );
}
