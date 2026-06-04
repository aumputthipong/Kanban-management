// components/board/settings/GeneralSection.tsx
"use client";

import { useState } from "react";
import {
  Clock,
  Check,
  X,
  Loader2,
  LayoutGrid,
  Rocket,
  Target,
  Zap,
  Bug,
} from "lucide-react";
import { SectionCard, SettingRow, MockBadge } from "./SettingsParts";

const SWATCHES = ["#1E40AF", "#0D9488", "#7C3AED", "#B45309", "#BE185D", "#0F172A"];

const GLYPHS = {
  board: LayoutGrid,
  rocket: Rocket,
  target: Target,
  bolt: Zap,
  bug: Bug,
} as const;
type GlyphKey = keyof typeof GLYPHS;

interface GeneralSectionProps {
  initialTitle: string;
  canManage: boolean;
  /** Real save — PATCH /boards/:id { title }. Resolves on success. */
  onSaveTitle: (title: string) => Promise<void>;
  onSaved: () => void;
}

/**
 * Board name is the only field with a backend (`title`). Description and the
 * icon/colour picker are visual mockups — they keep local state for the demo
 * but persist nothing, so each wears a MockBadge.
 */
export function GeneralSection({
  initialTitle,
  canManage,
  onSaveTitle,
  onSaved,
}: GeneralSectionProps) {
  const [name, setName] = useState(initialTitle ?? "");
  const [savedName, setSavedName] = useState(initialTitle ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Mockup-only local state
  const [desc, setDesc] = useState("บอร์ดหลักสำหรับติดตามงานพัฒนาและ Sprint ของทีม");
  const [color, setColor] = useState("#1E40AF");
  const [glyph, setGlyph] = useState<GlyphKey>("board");

  const nameDirty = name.trim() !== savedName.trim();
  const PreviewGlyph = GLYPHS[glyph];

  const handleSaveName = async () => {
    const clean = name.trim();
    if (!clean) {
      setError("ชื่อบอร์ดต้องไม่ว่าง");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSaveTitle(clean);
      setSavedName(clean);
      onSaved();
    } catch {
      setError("บันทึกไม่สำเร็จ ลองอีกครั้ง");
    } finally {
      setSaving(false);
    }
  };

  return (
    <SectionCard
      id="sec-general"
      icon={<Clock size={15} />}
      title="ทั่วไป"
      description="ชื่อ คำอธิบาย และรูปลักษณ์ของบอร์ดที่สมาชิกทุกคนมองเห็น"
    >
      {/* Board name — REAL */}
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
                  setError(null);
                }}
                className={`h-[42px] w-[320px] px-3.5 rounded-lg border bg-[#FBFCFE] text-sm text-slate-900 outline-none transition focus:bg-white focus:ring-2 focus:ring-blue-800/10 disabled:opacity-60 ${
                  error ? "border-red-400" : "border-slate-200 focus:border-blue-300"
                }`}
              />
              <button
                type="button"
                title="บันทึกชื่อ"
                disabled={!canManage || !nameDirty || saving}
                onClick={handleSaveName}
                className="w-[42px] h-[42px] rounded-lg flex items-center justify-center bg-blue-800 text-white hover:bg-blue-900 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                {saving ? (
                  <Loader2 size={18} className="animate-spin" />
                ) : (
                  <Check size={18} />
                )}
              </button>
              <button
                type="button"
                title="ยกเลิก"
                disabled={!canManage || !nameDirty || saving}
                onClick={() => {
                  setName(savedName);
                  setError(null);
                }}
                className="w-[42px] h-[42px] rounded-lg flex items-center justify-center border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
              >
                <X size={18} />
              </button>
            </div>
            {error && <span className="text-xs text-red-600">{error}</span>}
          </div>
        }
      />

      {/* Description — MOCKUP */}
      <SettingRow
        stacked
        label={
          <>
            คำอธิบาย <MockBadge />
          </>
        }
        help="บอกเป้าหมายหรือขอบเขตของบอร์ดสั้น ๆ ช่วยให้สมาชิกใหม่เข้าใจบริบท"
        control={
          <div className="flex flex-col gap-1">
            <textarea
              value={desc}
              maxLength={160}
              disabled={!canManage}
              onChange={(e) => setDesc(e.target.value)}
              placeholder="เช่น บอร์ดติดตามงานพัฒนา Sprint ของทีม Engineering…"
              className="w-full min-h-[78px] px-3.5 py-2.5 rounded-lg border border-slate-200 bg-[#FBFCFE] text-sm text-slate-900 leading-relaxed outline-none resize-y transition focus:bg-white focus:border-blue-300 focus:ring-2 focus:ring-blue-800/10 disabled:opacity-60"
            />
            <span className="self-end text-[11.5px] font-semibold text-slate-400">
              {desc.length}/160
            </span>
          </div>
        }
      />

      {/* Icon & colour — MOCKUP */}
      <SettingRow
        stacked
        label={
          <>
            ไอคอน &amp; สีประจำบอร์ด <MockBadge />
          </>
        }
        help="ใช้แยกบอร์ดนี้จากบอร์ดอื่นในแถบด้านข้างและรายการทั้งหมด"
        control={
          <div className="flex flex-col gap-3.5">
            <div className="flex items-center gap-3.5 p-3.5 rounded-lg border border-slate-200 bg-[#FBFCFE]">
              <div
                className="w-[46px] h-[46px] rounded-lg flex items-center justify-center text-white shrink-0 transition-colors"
                style={{ background: color }}
              >
                <PreviewGlyph size={24} />
              </div>
              <div>
                <div className="text-[15px] font-bold tracking-tight text-slate-900">
                  {name || "Untitled board"}
                </div>
                <div className="text-xs text-slate-400 mt-0.5">
                  ตัวอย่างที่แสดงในแถบด้านข้าง
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2.5">
              {SWATCHES.map((c) => (
                <button
                  key={c}
                  type="button"
                  disabled={!canManage}
                  onClick={() => setColor(c)}
                  style={{ background: c, color: c }}
                  className={`w-[30px] h-[30px] rounded-full transition-transform hover:scale-110 disabled:opacity-60 ${
                    color === c
                      ? "ring-2 ring-offset-2 ring-offset-white ring-current"
                      : ""
                  }`}
                />
              ))}
            </div>

            <div className="flex items-center gap-2">
              {(Object.keys(GLYPHS) as GlyphKey[]).map((key) => {
                const G = GLYPHS[key];
                const on = key === glyph;
                return (
                  <button
                    key={key}
                    type="button"
                    disabled={!canManage}
                    onClick={() => setGlyph(key)}
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
            </div>
          </div>
        }
      />
    </SectionCard>
  );
}
