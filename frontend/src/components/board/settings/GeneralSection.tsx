// components/board/settings/GeneralSection.tsx
"use client";

import { useState } from "react";
import { Clock, Check, X, Loader2 } from "lucide-react";
import { SectionCard, SettingRow } from "./SettingsParts";
import { BoardAppearancePicker } from "./BoardAppearancePicker";
import {
  DEFAULT_BOARD_COLOR,
  DEFAULT_BOARD_ICON,
  type BoardIconKey,
} from "@/lib/boardAppearance";

type BoardField = "title" | "description" | "color" | "icon";

interface GeneralSectionProps {
  initialTitle: string;
  initialDescription: string;
  initialColor: string;
  initialIcon: string;
  canManage: boolean;
  /** Real save — PATCH /boards/:id { [field]: value }. Resolves on success. */
  onSaveField: (field: BoardField, value: string) => Promise<void>;
  onSaved: () => void;
}

const DESC_MAX = 160;

/**
 * Name, description, and the icon/colour identity — all backed by the boards
 * table now. Name + description commit via a save button (dirty-gated); colour
 * and icon save optimistically the instant they are picked and revert if the
 * request fails.
 */
export function GeneralSection({
  initialTitle,
  initialDescription,
  initialColor,
  initialIcon,
  canManage,
  onSaveField,
  onSaved,
}: GeneralSectionProps) {
  const [name, setName] = useState(initialTitle ?? "");
  const [savedName, setSavedName] = useState(initialTitle ?? "");
  const [savingName, setSavingName] = useState(false);
  const [nameError, setNameError] = useState<string | null>(null);

  const [desc, setDesc] = useState(initialDescription ?? "");
  const [savedDesc, setSavedDesc] = useState(initialDescription ?? "");
  const [savingDesc, setSavingDesc] = useState(false);

  const [color, setColor] = useState(initialColor || DEFAULT_BOARD_COLOR);
  const [icon, setIcon] = useState<string>(initialIcon || DEFAULT_BOARD_ICON);
  const [savingAppearance, setSavingAppearance] = useState<"color" | "icon" | null>(null);

  const nameDirty = name.trim() !== savedName.trim();
  const descDirty = desc !== savedDesc;

  const handleSaveName = async () => {
    const clean = name.trim();
    if (!clean) {
      setNameError("ชื่อบอร์ดต้องไม่ว่าง");
      return;
    }
    setSavingName(true);
    setNameError(null);
    try {
      await onSaveField("title", clean);
      setSavedName(clean);
      onSaved();
    } catch {
      setNameError("บันทึกไม่สำเร็จ ลองอีกครั้ง");
    } finally {
      setSavingName(false);
    }
  };

  const handleSaveDesc = async () => {
    setSavingDesc(true);
    try {
      await onSaveField("description", desc);
      setSavedDesc(desc);
      onSaved();
    } finally {
      setSavingDesc(false);
    }
  };

  // Optimistic: reflect the pick immediately, revert if the PATCH fails.
  const pickColor = async (next: string) => {
    const prev = color;
    setColor(next);
    setSavingAppearance("color");
    try {
      await onSaveField("color", next);
      onSaved();
    } catch {
      setColor(prev);
    } finally {
      setSavingAppearance(null);
    }
  };

  const pickIcon = async (next: BoardIconKey) => {
    const prev = icon;
    setIcon(next);
    setSavingAppearance("icon");
    try {
      await onSaveField("icon", next);
      onSaved();
    } catch {
      setIcon(prev);
    } finally {
      setSavingAppearance(null);
    }
  };

  return (
    <SectionCard
      id="sec-general"
      icon={<Clock size={15} />}
      title="ทั่วไป"
      description="ชื่อ คำอธิบาย และรูปลักษณ์ของบอร์ดที่สมาชิกทุกคนมองเห็น"
    >
      {/* Board name */}
      <SettingRow
        label="ชื่อบอร์ด"
        help="ชื่อที่แสดงบนหัวบอร์ดและในรายการโปรเจกต์"
        control={
          <div className="flex flex-col items-end gap-1.5">
            <div className="flex items-center gap-2.5">
              <input
                value={name}
                disabled={!canManage}
                onChange={(e) => {
                  setName(e.target.value);
                  setNameError(null);
                }}
                className={`h-[42px] w-[320px] px-3.5 rounded-lg border bg-[#FBFCFE] text-sm text-slate-900 outline-none transition focus:bg-white focus:ring-2 focus:ring-blue-800/10 disabled:opacity-60 ${
                  nameError ? "border-red-400" : "border-slate-200 focus:border-blue-300"
                }`}
              />
              <button
                type="button"
                title="Save name"
                disabled={!canManage || !nameDirty || savingName}
                onClick={handleSaveName}
                className="w-[42px] h-[42px] rounded-lg flex items-center justify-center bg-blue-800 text-white hover:bg-blue-900 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                {savingName ? <Loader2 size={18} className="animate-spin" /> : <Check size={18} />}
              </button>
              <button
                type="button"
                title="Cancel"
                disabled={!canManage || !nameDirty || savingName}
                onClick={() => {
                  setName(savedName);
                  setNameError(null);
                }}
                className="w-[42px] h-[42px] rounded-lg flex items-center justify-center border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                <X size={18} />
              </button>
            </div>
            {nameError && <span className="text-xs text-red-600">{nameError}</span>}
          </div>
        }
      />

      {/* Description */}
      <SettingRow
        stacked
        label="คำอธิบาย"
        help="บอกเป้าหมายหรือขอบเขตของบอร์ดสั้น ๆ ช่วยให้สมาชิกใหม่เข้าใจบริบท"
        control={
          <div className="flex flex-col gap-2">
            <textarea
              value={desc}
              maxLength={DESC_MAX}
              disabled={!canManage}
              onChange={(e) => setDesc(e.target.value)}
              placeholder="เช่น บอร์ดติดตามงานพัฒนา Sprint ของทีม Engineering…"
              className="w-full min-h-[78px] px-3.5 py-2.5 rounded-lg border border-slate-200 bg-[#FBFCFE] text-sm text-slate-900 leading-relaxed outline-none resize-y transition focus:bg-white focus:border-blue-300 focus:ring-2 focus:ring-blue-800/10 disabled:opacity-60"
            />
            <div className="flex items-center justify-between">
              <span className="text-[11.5px] font-semibold text-slate-400">
                {desc.length}/{DESC_MAX}
              </span>
              {canManage && descDirty && (
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setDesc(savedDesc)}
                    disabled={savingDesc}
                    className="h-8 px-3 rounded-lg border border-slate-200 text-[13px] font-semibold text-slate-600 hover:bg-slate-50 disabled:opacity-40 transition"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleSaveDesc}
                    disabled={savingDesc}
                    className="h-8 px-3.5 rounded-lg bg-blue-800 text-white text-[13px] font-semibold hover:bg-blue-900 disabled:opacity-40 inline-flex items-center gap-1.5 transition"
                  >
                    {savingDesc ? <Loader2 size={14} className="animate-spin" /> : <Check size={14} />}
                    Save description
                  </button>
                </div>
              )}
            </div>
          </div>
        }
      />

      {/* Icon & colour */}
      <SettingRow
        stacked
        label="ไอคอน & สีประจำบอร์ด"
        help="ใช้แยกบอร์ดนี้จากบอร์ดอื่นในรายการโปรเจกต์และแถบด้านข้าง บันทึกทันทีเมื่อเลือก"
        control={
          <BoardAppearancePicker
            name={name}
            color={color}
            icon={icon}
            canManage={canManage}
            saving={savingAppearance}
            onPickColor={pickColor}
            onPickIcon={pickIcon}
          />
        }
      />
    </SectionCard>
  );
}
