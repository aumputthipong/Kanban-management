"use client";

// Display-only REQ/DEC/Q chip. Changing the type moved into the row's overflow menu so
// every row action lives in one place — do not re-add a click handler here. The full
// Thai meaning stays on hover via TYPE_TOOLTIP.
import type { PlanningItemType } from "@/types/planning";
import { TYPE_CHIP, TYPE_ICON, TYPE_TOOLTIP } from "./planningTypeMeta";

export function ItemTypeChip({ type }: { type: PlanningItemType }) {
  const Icon = TYPE_ICON[type];
  return (
    <span
      title={TYPE_TOOLTIP[type]}
      className={`inline-flex shrink-0 items-center gap-1 rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase ${TYPE_CHIP[type]}`}
    >
      <Icon size={11} />
      {type}
    </span>
  );
}
