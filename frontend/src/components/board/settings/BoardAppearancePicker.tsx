// components/board/settings/BoardAppearancePicker.tsx
"use client";

import { Loader2 } from "lucide-react";
import {
  BOARD_COLORS,
  BOARD_ICONS,
  boardColor,
  BoardGlyph,
  type BoardIconKey,
} from "@/lib/boardAppearance";

interface BoardAppearancePickerProps {
  name: string;
  color: string;
  icon: string;
  canManage: boolean;
  /** Which control is mid-save (so we can show a spinner without blocking). */
  saving: "color" | "icon" | null;
  onPickColor: (color: string) => void;
  onPickIcon: (icon: BoardIconKey) => void;
}

/**
 * Live identity preview + colour swatches + icon grid. Selections save
 * immediately (optimistic) via the parent; the chosen value is reflected here
 * the instant it is clicked. Mirrors the project-list card glyph so what the
 * user picks is what they see on the dashboard.
 */
export function BoardAppearancePicker({
  name,
  color,
  icon,
  canManage,
  saving,
  onPickColor,
  onPickIcon,
}: BoardAppearancePickerProps) {
  const accent = boardColor(color);

  return (
    <div className="flex flex-col gap-3.5">
      {/* preview */}
      <div className="flex items-center gap-3.5 p-3.5 rounded-lg border border-slate-200 bg-[#FBFCFE]">
        <div
          className="w-[46px] h-[46px] rounded-lg flex items-center justify-center text-white shrink-0 transition-colors"
          style={{ background: accent }}
        >
          <BoardGlyph icon={icon} size={24} />
        </div>
        <div>
          <div className="text-[15px] font-bold tracking-tight text-slate-900">
            {name || "บอร์ดไม่มีชื่อ"}
          </div>
          <div className="text-xs text-slate-400 mt-0.5">
            ตัวอย่างที่แสดงในรายการโปรเจกต์และแถบด้านข้าง
          </div>
        </div>
      </div>

      {/* colour swatches */}
      <div className="flex items-center gap-2.5">
        {BOARD_COLORS.map((c) => (
          <button
            key={c}
            type="button"
            title="เลือกสีนี้"
            disabled={!canManage || saving === "color"}
            onClick={() => onPickColor(c)}
            style={{ background: c, color: c }}
            className={`w-[30px] h-[30px] rounded-full transition-transform hover:scale-110 disabled:opacity-60 ${
              accent === c ? "ring-2 ring-offset-2 ring-offset-white ring-current" : ""
            }`}
          />
        ))}
        {saving === "color" && (
          <Loader2 size={15} className="animate-spin text-slate-400" />
        )}
      </div>

      {/* icon grid */}
      <div className="flex items-center gap-2">
        {(Object.keys(BOARD_ICONS) as BoardIconKey[]).map((key) => {
          const G = BOARD_ICONS[key];
          const on = key === icon;
          return (
            <button
              key={key}
              type="button"
              title="เลือกไอคอนนี้"
              disabled={!canManage || saving === "icon"}
              onClick={() => onPickIcon(key)}
              className={`w-[42px] h-[42px] rounded-lg border flex items-center justify-center transition disabled:opacity-60 ${
                on
                  ? "bg-indigo-50 border-blue-200 text-blue-800"
                  : "bg-[#FBFCFE] border-slate-200 text-slate-600 hover:border-slate-300"
              }`}
            >
              <G size={20} />
            </button>
          );
        })}
        {saving === "icon" && (
          <Loader2 size={15} className="animate-spin text-slate-400" />
        )}
      </div>
    </div>
  );
}
