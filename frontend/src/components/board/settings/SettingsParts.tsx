"use client";

import { ReactNode } from "react";
import { ChevronDown, FlaskConical } from "lucide-react";

/**
 * Shared building blocks for board settings. Visual language follows the
 * frontend/design.md tokens; no new accent colours are introduced here.
 */

/**
 * Marks a control with no backend yet — local state for the demo, persists nothing.
 */
export function MockBadge({ className = "" }: { className?: string }) {
  return (
    <span
      title="ส่วนนี้เป็นตัวอย่าง (mockup) — ยังไม่ได้เชื่อมต่อกับ backend"
      className={`inline-flex items-center gap-1.5 h-6 px-2 rounded-md bg-slate-100 border border-slate-200 text-slate-500 text-[11px] font-semibold whitespace-nowrap ${className}`}
    >
      <FlaskConical size={12} />
      ตัวอย่าง · ยังไม่ได้เชื่อมระบบ
    </span>
  );
}

interface SectionCardProps {
  id: string;
  icon: ReactNode;
  title: string;
  description: string;
  /** Renders a MockBadge next to the title when true. */
  mock?: boolean;
  danger?: boolean;
  children: ReactNode;
  footer?: ReactNode;
}

/** A titled settings card. `scroll-mt-20` keeps the sticky-rail jump aligned. */
export function SectionCard({
  id,
  icon,
  title,
  description,
  mock = false,
  danger = false,
  children,
  footer,
}: SectionCardProps) {
  return (
    <section
      id={id}
      className={`scroll-mt-20 bg-white rounded-xl shadow-sm overflow-hidden border ${
        danger ? "border-red-200" : "border-slate-200"
      }`}
    >
      <div
        className={`px-5 py-4 border-b ${
          danger
            ? "bg-red-50/60 border-red-200"
            : "border-slate-100"
        }`}
      >
        <div className="flex items-center gap-2.5 flex-wrap">
          <span
            className={`rounded-md flex items-center justify-center shrink-0 ${
              danger
                ? "bg-white border border-red-200 text-red-600"
                : "bg-indigo-50 text-blue-800"
            }`}
            style={{ width: 26, height: 26 }}
          >
            {icon}
          </span>
          <h2
            className={`text-base font-bold tracking-tight ${
              danger ? "text-red-700" : "text-slate-900"
            }`}
          >
            {title}
          </h2>
          {mock && <MockBadge />}
        </div>
        <p className="text-[13px] text-slate-500 mt-1.5 leading-relaxed">
          {description}
        </p>
      </div>
      <div>{children}</div>
      {footer}
    </section>
  );
}

interface SettingRowProps {
  label: ReactNode;
  help?: ReactNode;
  control: ReactNode;
  /** Stacks control below the label (full-width controls like textarea). */
  stacked?: boolean;
}

/** One label/help + control line inside a SectionCard. */
export function SettingRow({ label, help, control, stacked = false }: SettingRowProps) {
  if (stacked) {
    return (
      <div className="flex flex-col gap-3 px-5 py-4 border-t border-slate-100 first:border-t-0">
        <div className="min-w-0">
          <div className="text-sm font-semibold text-slate-900 flex items-center gap-2 flex-wrap">
            {label}
          </div>
          {help && (
            <p className="text-[12.5px] text-slate-500 mt-1 leading-relaxed max-w-[440px]">
              {help}
            </p>
          )}
        </div>
        <div>{control}</div>
      </div>
    );
  }

  return (
    <div className="flex items-start gap-6 px-5 py-4 border-t border-slate-100 first:border-t-0">
      <div className="flex-1 min-w-0 pt-0.5">
        <div className="text-sm font-semibold text-slate-900 flex items-center gap-2 flex-wrap">
          {label}
        </div>
        {help && (
          <p className="text-[12.5px] text-slate-500 mt-1 leading-relaxed max-w-[420px]">
            {help}
          </p>
        )}
      </div>
      <div className="shrink-0 flex flex-col items-end gap-2">{control}</div>
    </div>
  );
}

/** iOS-style toggle. Controlled. */
export function Toggle({
  checked,
  onChange,
  disabled = false,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative w-[42px] h-6 rounded-full transition-colors shrink-0 disabled:opacity-50 disabled:cursor-not-allowed ${
        checked ? "bg-blue-800" : "bg-slate-300"
      }`}
    >
      <span
        className={`absolute top-[3px] left-[3px] w-[18px] h-[18px] rounded-full bg-white shadow transition-transform ${
          checked ? "translate-x-[18px]" : ""
        }`}
      />
    </button>
  );
}

interface SegmentedProps<T extends string> {
  value: T;
  options: { value: T; label: ReactNode }[];
  onChange: (next: T) => void;
  disabled?: boolean;
}

/** Pill segmented control. Controlled. */
export function Segmented<T extends string>({
  value,
  options,
  onChange,
  disabled = false,
}: SegmentedProps<T>) {
  return (
    <div className="inline-flex bg-slate-100 border border-slate-200 rounded-lg p-[3px] gap-[3px]">
      {options.map((opt) => {
        const on = opt.value === value;
        return (
          <button
            key={opt.value}
            type="button"
            disabled={disabled}
            onClick={() => onChange(opt.value)}
            className={`h-[34px] px-3.5 rounded-md text-[13px] font-semibold inline-flex items-center gap-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${
              on
                ? "bg-white text-blue-800 shadow-sm"
                : "text-slate-600 hover:text-slate-900"
            }`}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}

/** Styled native select with chevron. */
export function SelectField({
  value,
  options,
  onChange,
  disabled = false,
}: {
  value: string;
  options: string[];
  onChange: (next: string) => void;
  disabled?: boolean;
}) {
  return (
    <div className="relative inline-block">
      <select
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        className="appearance-none h-[42px] pl-3.5 pr-9 border border-slate-200 rounded-lg bg-white text-sm font-semibold text-slate-900 cursor-pointer hover:border-slate-300 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:border-blue-300 focus:ring-2 focus:ring-blue-800/10"
      >
        {options.map((o) => (
          <option key={o} value={o}>
            {o}
          </option>
        ))}
      </select>
      <ChevronDown
        size={15}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
      />
    </div>
  );
}
