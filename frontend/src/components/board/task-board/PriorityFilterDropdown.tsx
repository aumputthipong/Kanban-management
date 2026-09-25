"use client";

import { useRef, useState, useEffect } from "react";
import { useBoardStore } from "@/store/useBoardStore";

// Config
export type Priority = "high" | "medium" | "low";

const PRIORITY_CONFIG: {
  value: Priority;
  label: string;
  bars: number;
  color: string;
  dimColor: string;
}[] = [
  { value: "high",   label: "High",   bars: 3, color: "#ef4444", dimColor: "#fca5a5" },
  { value: "medium", label: "Medium", bars: 2, color: "#f59e0b", dimColor: "#fcd34d" },
  { value: "low",    label: "Low",    bars: 1, color: "#3b82f6", dimColor: "#93c5fd" },
];

// Signal bars icon
export function SignalBars({
  priority,
  size = 14,
}: {
  priority: Priority | null;
  size?: number;
}) {
  const cfg = PRIORITY_CONFIG.find((p) => p.value === priority);
  const filled = cfg?.bars ?? 0;
  const activeColor = cfg?.color ?? "#94a3b8";
  const emptyColor = "#e2e8f0";

  const scale = size / 12;
  const barWidth = Math.round(3 * scale);
  const gap = Math.round(2 * scale);
  const heights = [4 * scale, 7 * scale, 11 * scale];
  const totalW = barWidth * 3 + gap * 2;

  return (
    <svg width={totalW} height={size} viewBox={`0 0 ${totalW} ${size}`} fill="none">
      {heights.map((h, i) => (
        <rect
          key={i}
          x={i * (barWidth + gap)}
          y={size - h}
          width={barWidth}
          height={h}
          rx={1}
          fill={i < filled ? activeColor : emptyColor}
        />
      ))}
    </svg>
  );
}

// Dropdown
export function PriorityFilterDropdown() {
  const filterPriorities = useBoardStore((s) => s.filterPriorities);
  const toggleFilterPriority = useBoardStore((s) => s.toggleFilterPriority);
  const clearFilterPriorities = useBoardStore((s) => s.clearFilterPriorities);

  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  const activeCount = filterPriorities.length;
  const isActive = activeCount > 0;

  return (
    <div className="relative" ref={ref}>
      {/* Trigger */}
      <button
        onClick={() => setOpen((v) => !v)}
        className={`cursor-pointer flex items-center gap-1.5 h-7 px-2.5 rounded-lg border text-xs font-medium transition-colors ${
          isActive
            ? "bg-blue-50 border-blue-300 text-blue-700"
            : "bg-white border-slate-200 text-slate-600 hover:border-slate-400 hover:text-slate-800"
        }`}
      >
        {isActive && filterPriorities.length === 1 ? (
          <SignalBars priority={filterPriorities[0] as Priority} size={13} />
        ) : (
          <SignalBars priority={null} size={13} />
        )}
        <span>Priority</span>
        {isActive && (
          <span className="flex items-center justify-center w-4 h-4 rounded-full bg-blue-500 text-white text-[9px] font-bold leading-none">
            {activeCount}
          </span>
        )}
        <svg
          className={`w-3 h-3 text-slate-400 transition-transform ${open ? "rotate-180" : ""}`}
          viewBox="0 0 10 10"
          fill="none"
        >
          <path
            d="M2 3.5L5 6.5L8 3.5"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
      </button>

      {/* Menu */}
      {open && (
        <div className="absolute top-9 left-0 z-50 w-44 bg-white border border-slate-200 rounded-xl shadow-lg py-1.5 overflow-hidden">
          {PRIORITY_CONFIG.map(({ value, label }) => {
            const checked = filterPriorities.includes(value);
            return (
              <button
                key={value}
                onClick={() => toggleFilterPriority(value)}
                className={`cursor-pointer flex items-center gap-2.5 w-full px-3 py-2 text-xs text-left transition-colors ${
                  checked
                    ? "bg-blue-50 text-blue-700"
                    : "text-slate-700 hover:bg-slate-50"
                }`}
              >
                <SignalBars priority={value} size={13} />
                <span className="flex-1 font-medium">{label}</span>
                {checked && (
                  <svg className="w-3.5 h-3.5 text-blue-500 shrink-0" viewBox="0 0 14 14" fill="none">
                    <path d="M2.5 7L5.5 10L11.5 4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </button>
            );
          })}

          {isActive && (
            <>
              <div className="my-1 border-t border-slate-100" />
              <button
                onClick={() => { clearFilterPriorities(); setOpen(false); }}
                className="cursor-pointer flex items-center gap-2 w-full px-3 py-1.5 text-xs text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors"
              >
                Clear filter
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}
