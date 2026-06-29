"use client";

// ItemActionMenu — the row's secondary actions, folded into a single "⋯"
// overflow menu so the row reads as one clear primary ("ส่งขึ้น Board") plus a
// quiet "more" affordance, instead of a row of ambiguous grey icons.
//
// Each item carries a Thai text label (the bare icons were unreadable without
// hovering for a tooltip). Batch "select to send" is NOT here — it's a
// dedicated select-mode toggled from the toolbar (checkboxes appear on every
// row), so the per-row menu only holds the single-item actions.
import { useEffect, useRef, useState } from "react";
import {
  Ban,
  ChevronDown,
  ChevronRight,
  MoreHorizontal,
  Pencil,
  Trash2,
  type LucideIcon,
} from "lucide-react";
import type { PlanningItemType } from "@/types/planning";
import {
  TYPE_CHIP,
  TYPE_CHIP_ACTIVE,
  TYPE_CYCLE,
  TYPE_ICON,
  TYPE_LONG,
} from "./planningTypeMeta";

interface Props {
  currentType: PlanningItemType;
  dropped: boolean;
  promoted: boolean;
  expanded: boolean;
  hasDetails: boolean;
  onRename: () => void;
  onChangeType: (t: PlanningItemType) => void;
  onToggleDropped: () => void;
  onDelete: () => void;
  onToggleExpanded: () => void;
}

export function ItemActionMenu({
  currentType,
  dropped,
  promoted,
  expanded,
  hasDetails,
  onRename,
  onChangeType,
  onToggleDropped,
  onDelete,
  onToggleExpanded,
}: Props) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  // Dismiss on outside mousedown (before the click lands) so clicking the row
  // to close the menu doesn't also re-focus / edit the row.
  useEffect(() => {
    if (!open) return;
    const onMouseDown = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [open]);

  const run = (fn: () => void) => (e: React.MouseEvent) => {
    e.stopPropagation();
    setOpen(false);
    fn();
  };

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setOpen((v) => !v);
        }}
        title="ตัวเลือกเพิ่มเติม"
        aria-label="ตัวเลือกเพิ่มเติม"
        aria-expanded={open}
        className={`rounded p-1 text-slate-400 opacity-60 transition-all group-hover:opacity-100 hover:bg-slate-200 hover:text-slate-700 ${
          open ? "bg-slate-200 text-slate-700 opacity-100" : ""
        }`}
      >
        <MoreHorizontal size={16} />
      </button>

      {open && (
        <div
          className="absolute right-0 top-full z-20 mt-1 w-48 rounded-md border border-slate-200 bg-white py-1 shadow-lg"
          onClick={(e) => e.stopPropagation()}
        >
          <MenuItem icon={Pencil} onClick={run(onRename)} disabled={promoted}>
            เปลี่ยนชื่อ
          </MenuItem>

          {/* Change type — moved out of the row chip into the menu so every row
              action lives here. Hidden on promoted items (retype is blocked
              once an item has a board card). */}
          {!promoted && (
            <div className="px-3 py-1.5">
              <p className="mb-1 text-[10px] font-semibold uppercase tracking-wider text-slate-400">
                เปลี่ยนประเภท
              </p>
              <div className="flex gap-1">
                {TYPE_CYCLE.map((t) => {
                  const active = t === currentType;
                  const Icon = TYPE_ICON[t];
                  return (
                    <button
                      key={t}
                      type="button"
                      onClick={run(() => {
                        if (t !== currentType) onChangeType(t);
                      })}
                      title={TYPE_LONG[t]}
                      className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase transition-colors ${
                        active ? TYPE_CHIP_ACTIVE[t] : TYPE_CHIP[t]
                      }`}
                    >
                      <Icon size={11} />
                      {t}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          <div className="my-1 border-t border-slate-100" />
          <MenuItem
            icon={Ban}
            onClick={run(onToggleDropped)}
            disabled={promoted}
            active={dropped}
          >
            {dropped ? "เอากลับมา" : "พักไว้ก่อน"}
          </MenuItem>
          <MenuItem
            icon={expanded ? ChevronDown : ChevronRight}
            onClick={run(onToggleExpanded)}
            active={hasDetails}
          >
            {expanded ? "Hide details" : "Show details"}
          </MenuItem>
          <div className="my-1 border-t border-slate-100" />
          <MenuItem icon={Trash2} onClick={run(onDelete)} danger>
            Delete item
          </MenuItem>
        </div>
      )}
    </div>
  );
}

function MenuItem({
  icon: Icon,
  onClick,
  disabled,
  active,
  danger,
  children,
}: {
  icon: LucideIcon;
  onClick: (e: React.MouseEvent) => void;
  disabled?: boolean;
  active?: boolean;
  danger?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        danger
          ? "text-red-600 hover:bg-red-50"
          : active
            ? "text-indigo-700 hover:bg-indigo-50"
            : "text-slate-600 hover:bg-slate-50"
      }`}
    >
      <Icon size={14} className="shrink-0" />
      {children}
    </button>
  );
}
