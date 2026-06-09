"use client";

// ItemTypePopover — the type chip on each row, with a 3-option popover
// for converting REQ ↔ DEC ↔ Q. Extracted from ItemRow during the polish
// pass because the popover (with its click-outside ref + 3 inner buttons)
// was the single largest visual block in a file that had grown past 400
// lines after F2/F3/F5/F6.
//
// The promoted-item path is "disabled chip with explanatory tooltip" —
// promoted items have a corresponding Kanban card whose semantics depend
// on the original type, so retype is intentionally blocked here and in
// the backend (handler returns 400).
import { useEffect, useRef, useState } from "react";
import type { PlanningItemType } from "@/types/planning";
import {
  TYPE_CHIP,
  TYPE_CHIP_ACTIVE,
  TYPE_CYCLE,
  TYPE_ICON,
  TYPE_LONG,
  TYPE_TOOLTIP,
} from "./planningTypeMeta";

interface Props {
  type: PlanningItemType;
  disabled: boolean;
  onChange: (next: PlanningItemType) => void;
}

export function ItemTypePopover({ type, disabled, onChange }: Props) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  // Close the menu on any outside mousedown so the dismiss happens BEFORE
  // the outside element's click handler — clicking elsewhere on the row
  // shouldn't toggle row focus or enter edit mode on the very click that
  // dismisses this popover.
  useEffect(() => {
    if (!open) return;
    const onMouseDown = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [open]);

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          if (disabled) return;
          setOpen((v) => !v);
        }}
        disabled={disabled}
        title={
          disabled
            ? "ส่งเข้า Board แล้ว เปลี่ยนประเภทไม่ได้"
            : `${TYPE_TOOLTIP[type]} · คลิกเพื่อเปลี่ยน`
        }
        className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase ${TYPE_CHIP[type]} ${
          disabled ? "cursor-not-allowed opacity-60" : "hover:brightness-95"
        }`}
      >
        {(() => {
          const Icon = TYPE_ICON[type];
          return <Icon size={11} />;
        })()}
        {type}
      </button>
      {open && (
        <div
          className="absolute left-0 top-full z-10 mt-1 flex gap-1 rounded-md border border-slate-200 bg-white p-1 shadow-md"
          onClick={(e) => e.stopPropagation()}
        >
          {TYPE_CYCLE.map((t) => {
            const active = t === type;
            return (
              <button
                key={t}
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  setOpen(false);
                  if (t !== type) onChange(t);
                }}
                className={`whitespace-nowrap rounded border px-2 py-0.5 text-[10px] font-bold uppercase transition-colors ${
                  active ? TYPE_CHIP_ACTIVE[t] : TYPE_CHIP[t]
                }`}
                title={TYPE_LONG[t]}
              >
                {t}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
