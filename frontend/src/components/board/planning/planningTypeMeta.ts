// Visual metadata for the three planning item types, shared by the capture input's
// segmented control and each row chip so colour and label stay consistent.

import { Gavel, HelpCircle, Target, type LucideIcon } from "lucide-react";
import type { PlanningItemType } from "@/types/planning";

// Leading glyph per type, so REQ/DEC/Q read at a glance and not only by colour.
export const TYPE_ICON: Record<PlanningItemType, LucideIcon> = {
  REQ: Target,
  DEC: Gavel,
  Q: HelpCircle,
};

// Tooltip on the small REQ/DEC/Q chips, so density does not cost legibility.
export const TYPE_TOOLTIP: Record<PlanningItemType, string> = {
  REQ: "Requirement — สิ่งที่ต้องทำ",
  DEC: "Decision — ที่ตกลงกัน",
  Q: "Question — คำถามที่ยังตอบไม่ได้",
};

// Full label for the segmented control; the codes stay on row chips for density.
export const TYPE_LONG: Record<PlanningItemType, string> = {
  REQ: "Requirement",
  DEC: "Decision",
  Q: "Question",
};

// Soft chip palette — used on each row's small REQ/DEC/Q chip.
export const TYPE_CHIP: Record<PlanningItemType, string> = {
  REQ: "bg-red-50 text-red-700 border-red-200",
  DEC: "bg-blue-50 text-blue-700 border-blue-200",
  Q: "bg-amber-50 text-amber-700 border-amber-200",
};

// Solid (active) styles for the segmented control — matches the calendar pill.
export const TYPE_CHIP_ACTIVE: Record<PlanningItemType, string> = {
  REQ: "bg-red-600 text-white border-red-600",
  DEC: "bg-blue-600 text-white border-blue-600",
  Q: "bg-amber-500 text-white border-amber-500",
};

// Iteration order for the three types.
export const TYPE_CYCLE: PlanningItemType[] = ["REQ", "DEC", "Q"];
