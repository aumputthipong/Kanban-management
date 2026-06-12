"use client";

// ItemTypeChip — display-only REQ/DEC/Q chip on each row (icon + English code).
// Changing the type is no longer triggered from this chip: that action moved
// into the row's "⋯" menu (alongside rename) so every row action lives in one
// predictable place. The chip is purely informational here; the full Thai
// meaning is still on hover via TYPE_TOOLTIP.
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
