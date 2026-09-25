"use client";

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
