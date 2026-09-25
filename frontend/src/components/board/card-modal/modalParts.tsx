"use client";

import { Check } from "lucide-react";

export function FieldLabel({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <span className={`text-xs font-medium tracking-wide text-slate-600 ${className}`}>
      {children}
    </span>
  );
}

export function Dot({ className }: { className: string }) {
  return <span aria-hidden className={`inline-block w-1.25 h-1.25 rounded-full shrink-0 ${className}`} />;
}

export const PRIORITY_DOT: Record<string, string> = {
  high: "bg-red-600",
  medium: "bg-amber-500",
  low: "bg-emerald-500",
};

export const PRIORITY_LABEL: Record<string, string> = {
  high: "High",
  medium: "Medium",
  low: "Low",
};

export function MetaChip({ children, strong = false }: { children: React.ReactNode; strong?: boolean }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 h-6 px-2.5 rounded-full border border-slate-200 bg-white text-xs font-medium tracking-wide ${
        strong ? "text-slate-900" : "text-slate-600"
      }`}
    >
      {children}
    </span>
  );
}

export function SubtaskProgress({ done, total, showPercent = false }: { done: number; total: number; showPercent?: boolean }) {
  const pct = total === 0 ? 0 : Math.round((done / total) * 100);
  const complete = total > 0 && done === total;
  return (
    <>
      <div className="flex items-center gap-2 mb-2.5">
        <FieldLabel>Subtasks</FieldLabel>
        {total > 0 && (
          <span
            className={`ml-auto text-xs font-semibold tracking-wide px-2 py-0.5 rounded-full ${
              complete ? "bg-emerald-100 text-emerald-700" : "bg-surface-tint text-primary"
            }`}
          >
            {done} / {total}
            {showPercent && ` · ${pct}%`}
          </span>
        )}
      </div>
      {total > 0 && (
        // No width transition: it re-lays out every frame.
        <div className="h-0.5 mb-3 rounded-full bg-slate-200 overflow-hidden">
          <div
            className={`h-full ${complete ? "bg-emerald-700" : "bg-blue-700"}`}
            style={{ width: `${pct}%` }}
          />
        </div>
      )}
    </>
  );
}

export function subtaskRowClass(isDone: boolean): string {
  return `group flex items-center gap-2.5 px-2.5 py-1.5 rounded-md border text-sm transition-colors ${
    isDone ? "bg-emerald-100 border-transparent text-slate-900" : "bg-white border-slate-200 text-slate-900"
  }`;
}

export function SubtaskCheckbox({
  checked,
  disabled,
  label,
  onToggle,
}: {
  checked: boolean;
  disabled?: boolean;
  label: string;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={onToggle}
      className={`w-4 h-4 shrink-0 grid place-items-center rounded-sm border-2 bg-white transition-colors disabled:cursor-default ${
        checked
          ? "border-emerald-700 text-emerald-700"
          : "border-slate-400 text-transparent hover:border-emerald-700"
      }`}
    >
      <Check size={11} strokeWidth={3.2} />
    </button>
  );
}
