"use client";

import { useEffect, useRef, useState } from "react";
import { Calendar, Clock } from "lucide-react";

interface SnoozeMenuProps {
  onSnooze: (dueDate: string) => void;
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

  const choose = (offset: number) => (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen(false);
    onSnooze(isoOffset(offset));
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
          <MenuItem onClick={choose(1)} label="พรุ่งนี้" />
          <MenuItem onClick={choose(3)} label="อีก 3 วัน" />
          <MenuItem onClick={choose(7)} label="สัปดาห์หน้า" />
          <DateInputItem onPick={(iso) => { setOpen(false); onSnooze(iso); }} />
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

function DateInputItem({ onPick }: { onPick: (iso: string) => void }) {
  return (
    <label className="flex items-center gap-2 px-3 py-1.5 text-slate-700 hover:bg-slate-50 cursor-pointer border-t border-slate-100">
      <Calendar size={12} className="text-slate-400" />
      <span>เลือกวัน...</span>
      <input
        type="date"
        className="ml-auto text-[11px] border border-slate-200 rounded px-1 py-0.5"
        onClick={(e) => e.stopPropagation()}
        onChange={(e) => {
          if (e.target.value) onPick(e.target.value);
        }}
      />
    </label>
  );
}
