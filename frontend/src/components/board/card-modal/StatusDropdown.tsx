"use client";

import { useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { ChevronDown } from "lucide-react";
import { getColumnColorHex } from "../task-board/ColumnOptionsModal";
import type { Column } from "@/types/board";

interface StatusDropdownProps {
  columns: Column[];
  currentColumnId: string;
  onChange: (newColumnId: string) => void;
  disabled?: boolean;
}

export function StatusDropdown({
  columns,
  currentColumnId,
  onChange,
  disabled,
}: StatusDropdownProps) {
  const [open, setOpen] = useState(false);
  const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({});
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const current = columns.find((c) => c.id === currentColumnId);
  const todoCols = columns.filter((c) => c.category !== "DONE");
  const doneCols = columns.filter((c) => c.category === "DONE");

  useEffect(() => {
    if (!open || !btnRef.current) return;
    const r = btnRef.current.getBoundingClientRect();
    setMenuStyle({
      position: "fixed",
      top: r.bottom + 4,
      left: r.left,
      minWidth: Math.max(r.width, 180),
      zIndex: 99999,
    });
  }, [open]);

  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (
        menuRef.current?.contains(e.target as Node) ||
        btnRef.current?.contains(e.target as Node)
      )
        return;
      setOpen(false);
    };
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  const currentHex = getColumnColorHex(current?.color);

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        disabled={disabled}
        onClick={() => setOpen((v) => !v)}
        aria-label="Status"
        className="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-full border border-slate-200 bg-white text-xs font-medium tracking-wide text-slate-900 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed cursor-pointer"
      >
        <span
          className="w-1.25 h-1.25 rounded-full shrink-0"
          style={{ backgroundColor: currentHex ?? "#94a3b8" }}
        />
        <span className="truncate max-w-32">{current?.title ?? "—"}</span>
        {!disabled && <ChevronDown size={12} className="text-slate-600" />}
      </button>

      {open &&
        createPortal(
          <div
            ref={menuRef}
            style={menuStyle}
            className="bg-white border border-slate-200 rounded-lg shadow-xl overflow-hidden py-1"
          >
            {todoCols.length > 0 && (
              <>
                <div className="px-3 py-1 text-[10px] font-bold text-slate-400 uppercase tracking-wider">
                  To do
                </div>
                {todoCols.map((col) => (
                  <StatusOption
                    key={col.id}
                    col={col}
                    current={currentColumnId}
                    onSelect={(id) => {
                      onChange(id);
                      setOpen(false);
                    }}
                  />
                ))}
              </>
            )}
            {doneCols.length > 0 && (
              <>
                <div className="px-3 py-1 mt-1 text-[10px] font-bold text-slate-400 uppercase tracking-wider border-t border-slate-100">
                  Done
                </div>
                {doneCols.map((col) => (
                  <StatusOption
                    key={col.id}
                    col={col}
                    current={currentColumnId}
                    onSelect={(id) => {
                      onChange(id);
                      setOpen(false);
                    }}
                  />
                ))}
              </>
            )}
          </div>,
          document.body,
        )}
    </>
  );
}

function StatusOption({
  col,
  current,
  onSelect,
}: {
  col: Column;
  current: string;
  onSelect: (id: string) => void;
}) {
  const hex = getColumnColorHex(col.color);
  const active = col.id === current;
  return (
    <button
      type="button"
      onClick={() => onSelect(col.id)}
      className={`w-full text-left px-3 py-1.5 text-xs flex items-center gap-2 transition-colors cursor-pointer ${
        active ? "bg-blue-50 font-semibold text-blue-700" : "hover:bg-slate-50 text-slate-700"
      }`}
    >
      <span
        className="w-2 h-2 rounded-full shrink-0"
        style={{ backgroundColor: hex ?? "#94a3b8" }}
      />
      <span className="truncate">{col.title}</span>
    </button>
  );
}
