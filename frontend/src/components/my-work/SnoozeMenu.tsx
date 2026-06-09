"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
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

// Rough menu height (2 items + padding) — used to decide whether to flip the
// menu above the button when there's little room below.
const MENU_EST_HEIGHT = 84;

interface MenuPos {
  /** Distance from the viewport right edge — aligns the menu to the button. */
  right: number;
  /** Either anchors the top (open downward) or bottom (open upward). */
  top?: number;
  bottom?: number;
}

export function SnoozeMenu({ onSnooze }: SnoozeMenuProps) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState<MenuPos | null>(null);
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // The menu renders in a portal on document.body (so the panels' overflow
  // can't clip it), positioned `fixed` against the button. Because it's fixed,
  // it can't follow a scroll — so any scroll/resize closes it. Outside-pointer
  // and Escape also close. Pointerdown (not click) so the menu dismisses before
  // the row's parent Link can navigate.
  useEffect(() => {
    if (!open) return;
    function onDown(e: PointerEvent) {
      const t = e.target as Node;
      if (btnRef.current?.contains(t) || menuRef.current?.contains(t)) return;
      setOpen(false);
    }
    function close() {
      setOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("pointerdown", onDown);
    // capture=true so a scroll on any internal panel (not just window) closes it
    window.addEventListener("scroll", close, true);
    window.addEventListener("resize", close);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onDown);
      window.removeEventListener("scroll", close, true);
      window.removeEventListener("resize", close);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const toggle = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (open || !btnRef.current) {
      setOpen(false);
      return;
    }
    const r = btnRef.current.getBoundingClientRect();
    const right = window.innerWidth - r.right;
    const flipUp = window.innerHeight - r.bottom < MENU_EST_HEIGHT;
    setPos(
      flipUp
        ? { right, bottom: window.innerHeight - r.top + 6 }
        : { right, top: r.bottom + 6 },
    );
    setOpen(true);
  };

  const choose = (offset: number, label: string) => (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen(false);
    onSnooze(isoOffset(offset), label);
  };

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        title="เลื่อนวันครบกำหนด"
        aria-label="เลื่อนวันครบกำหนด"
        aria-expanded={open}
        onClick={toggle}
        className="w-6 h-6 rounded-sm flex items-center justify-center text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
      >
        <Clock size={13} />
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={menuRef}
            role="menu"
            style={{
              position: "fixed",
              right: pos.right,
              top: pos.top,
              bottom: pos.bottom,
              zIndex: 9999,
            }}
            className="min-w-44 rounded-lg border border-slate-200 bg-white shadow-lg py-1 text-xs"
          >
            {OPTIONS.map((o) => (
              <MenuItem key={o.offset} onClick={choose(o.offset, o.label)} label={o.label} />
            ))}
          </div>,
          document.body,
        )}
    </>
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
