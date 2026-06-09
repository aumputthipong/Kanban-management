// planningTypeMeta.ts — visual metadata for the three planning item types
// (REQ / DEC / Q). Shared between the capture input's segmented control and
// each item row's small chip so the colour and Thai label stay consistent
// across the two surfaces.

import { Gavel, HelpCircle, Target, type LucideIcon } from "lucide-react";
import type { PlanningItemType } from "@/types/planning";

// Leading glyph on each type chip — Requirement = target (what we're aiming
// for), Decision = gavel (what's been ruled), Question = help circle. Mirrors
// the planning redesign so REQ/DEC/Q read at a glance, not just by colour.
export const TYPE_ICON: Record<PlanningItemType, LucideIcon> = {
  REQ: Target,
  DEC: Gavel,
  Q: HelpCircle,
};

// Full Thai label used in tooltips on the small REQ/DEC/Q chips so a hover
// reveals "what does this code actually mean" — keeps row density compact
// while making the chip self-documenting for new users.
export const TYPE_TOOLTIP: Record<PlanningItemType, string> = {
  REQ: "Requirement — สิ่งที่ต้องทำ",
  DEC: "Decision — ที่ตกลงกัน",
  Q: "Question — คำถามที่ยังตอบไม่ได้",
};

// Full Thai label used in the segmented control next to the input. The
// abbreviated codes are kept only on row chips (density) and in the
// sidebar count where users have time to read.
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

// Solid (active) styles for the segmented control — match the filled
// priority chip pattern from the calendar pill so the visual vocabulary
// stays consistent across the app.
export const TYPE_CHIP_ACTIVE: Record<PlanningItemType, string> = {
  REQ: "bg-red-600 text-white border-red-600",
  DEC: "bg-blue-600 text-white border-blue-600",
  Q: "bg-amber-500 text-white border-amber-500",
};

// Iteration order — also used as the cycle for click-to-change-type on
// the row chip.
export const TYPE_CYCLE: PlanningItemType[] = ["REQ", "DEC", "Q"];
