"use client";

import { useEffect, useRef, useState } from "react";
import { Clock } from "lucide-react";

interface SnoozeMenuProps {
  /** Fires with the new due date (YYYY-MM-DD) + a Thai label for the toast. */
  onSnooze: (dueDate: string, label: string) => void;
}

function isoOffset(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return [
    d.getFullYear(),
    String(d.getMonth() + 1).padStart(2, "0"),
    String(d.getDate()).padStart(2, "0"),
  ].join("-");
}

// Two presets only — +1 / +7 map to intuitive labels. The "+3 days" preset was
// dropped (a magic number with no mental model) and so was the custom date
// picker (it duplicated the card modal's due-date field and made "any day"
// deferral a one-click habit). Reschedule here is a quick triage nudge, not a
// full editor — pick a real date in the card if you need precision.
const OPTIONS: { offset: number; label: string }[] = [
  { offset: 1, label: "พรุ่งนี้" },
  { offset: 7, label: "สัปดาห์หน้า" },
];

export function SnoozeMenu({ onSnooze }: SnoozeMenuProps) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  // Click outside closes. Pointerdown rather than click so the menu dismisses
  // before the row's parent Link navigates (the row is a Next.js Link).
  useEffect(() => {
    if (!open) return;
    function onDown(e: PointerEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onDown);
    return () => document.removeEventListener("pointerdown", onDown);
  }, [open]);

  const choose = (offset: number, label: string) => (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen(false);
    onSnooze(isoOffset(offset), label);
  };

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        title="เลื่อนวันครบกำหนด"
        aria-label="เลื่อนวันครบกำหนด"
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          setOpen((v) => !v);
        }}
        className="w-6 h-6 rounded-sm flex items-center justify-center text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
      >
        <Clock size={13} />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 top-7 z-20 min-w-44 rounded-lg border border-slate-200 bg-white shadow-md py-1 text-xs"
        >
          {OPTIONS.map((o) => (
            <MenuItem key={o.offset} onClick={choose(o.offset, o.label)} label={o.label} />
          ))}
        </div>
      )}
    </div>
  );
}

function MenuItem({ onClick, label }: { onClick: (e: React.MouseEvent) => void; label: string }) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      className="w-full text-left px-3 py-1.5 text-slate-700 hover:bg-slate-50"
    >
      {label}
    </button>
  );
}
