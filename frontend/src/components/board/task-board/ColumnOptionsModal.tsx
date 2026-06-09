"use client";

import { useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { Check, CircleCheck, Inbox, Trash2, X } from "lucide-react";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";

// ── colour palette ────────────────────────────────────────────────────────────
export const COLUMN_COLOR_PALETTE: {
  key: string | null;
  hex: string | null;
  label: string;
}[] = [
  { key: null, hex: null, label: "Default" },
  { key: "slate", hex: "#94a3b8", label: "Slate" },
  { key: "blue", hex: "#60a5fa", label: "Blue" },
  { key: "purple", hex: "#a78bfa", label: "Purple" },
  { key: "green", hex: "#34d399", label: "Green" },
  { key: "amber", hex: "#fbbf24", label: "Amber" },
  { key: "rose", hex: "#fb7185", label: "Rose" },
  { key: "pink", hex: "#f472b6", label: "Pink" },
  { key: "cyan", hex: "#22d3ee", label: "Cyan" },
];

export function getColumnColorHex(key?: string | null): string | null {
  return COLUMN_COLOR_PALETTE.find((c) => c.key === (key ?? null))?.hex ?? null;
}

// ── props ─────────────────────────────────────────────────────────────────────
interface ColumnOptionsModalProps {
  open: boolean;
  columnId: string;
  initialTitle: string;
  initialCategory: "TODO" | "DONE";
  initialColor: string | null;
  cardCount: number;
  onSave: (
    title: string,
    category: "TODO" | "DONE",
    color: string | null,
  ) => void;
  onDelete: () => void;
  onClose: () => void;
}

// ── component ─────────────────────────────────────────────────────────────────
export function ColumnOptionsModal({
  open,
  initialTitle,
  initialCategory,
  initialColor,
  cardCount,
  onSave,
  onDelete,
  onClose,
}: ColumnOptionsModalProps) {
  // State initialises from props once. Caller (Column.tsx) remounts this via
  // `key={`${columnId}-${open}`}` so each open starts fresh — no in-effect sync.
  const [title, setTitle] = useState(initialTitle);
  const [category, setCategory] = useState<"TODO" | "DONE">(initialCategory);
  const [color, setColor] = useState<string | null>(initialColor);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  // Escape to close
  useEffect(() => {
    if (!open) return;
    const h = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [open, onClose]);

  if (!open) return null;

  const handleSave = () => {
    const trimmed = title.trim();
    if (!trimmed) return;
    onSave(trimmed, category, color);
    onClose();
  };

  const selectedHex = getColumnColorHex(color);
  // Mirror the real column tint formula (Column.tsx) so the preview is truthful.
  const previewBg = selectedHex
    ? `color-mix(in srgb, ${selectedHex} 12%, white)`
    : "#f1f5f9";

  return createPortal(
    <>
      {/* backdrop */}
      <div className="fixed inset-0 z-[9998] bg-black/30" onClick={onClose} />

      {/* panel */}
      <div className="fixed inset-0 z-[9999] flex items-center justify-center pointer-events-none">
        <div
          className="pointer-events-auto w-full max-w-sm mx-4 bg-white rounded-2xl shadow-2xl flex flex-col overflow-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          {/* header */}
          <div className="flex items-center justify-between px-5 pt-5 pb-3 border-b border-slate-100">
            <h2 className="text-sm font-bold text-slate-800">Column options</h2>
            <button
              onClick={onClose}
              className="p-1 text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100 transition-colors"
            >
              <X size={16} />
            </button>
          </div>

          <div className="px-5 py-4 flex flex-col gap-5">
            {/* ── name ── */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
                Name
              </label>
              <input
                autoFocus
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleSave();
                }}
                className="text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 focus:border-blue-400 transition"
              />
            </div>

            {/* ── category ── */}
            <div className="flex flex-col gap-1.5">
              <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
                Category
              </label>
              <div className="flex gap-2">
                {(["TODO", "DONE"] as const).map((cat) => (
                  <button
                    key={cat}
                    onClick={() => setCategory(cat)}
                    className={`flex-1 py-2 text-xs font-semibold rounded-lg border transition-colors ${
                      category === cat
                        ? cat === "DONE"
                          ? "bg-emerald-500 text-white border-emerald-500"
                          : "bg-blue-500 text-white border-blue-500"
                        : "bg-white text-slate-600 border-slate-200 hover:border-slate-400"
                    }`}
                  >
                    {cat}
                  </button>
                ))}
              </div>
              <p className="text-[11px] leading-snug text-slate-400">
                ประเภทกำหนด{" "}
                <span className="font-semibold text-slate-500">พฤติกรรม</span> —
                คอลัมน์ DONE จะยุบเป็นแถบและการ์ดแสดงแบบเสร็จแล้ว
              </p>
            </div>

            {/* ── color ── identity, with a live header preview ── */}
            <div className="flex flex-col gap-2">
              <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
                Color
              </label>

              {/* Live preview of the actual column header */}
              <div
                className="rounded-lg border border-slate-100 px-3 py-2.5"
                style={{ backgroundColor: previewBg }}
              >
                <div className="flex items-center gap-2">
                  <span
                    className="w-2.5 h-2.5 rounded-full shrink-0"
                    style={{ backgroundColor: selectedHex ?? "#cbd5e1" }}
                  />
                  {category === "DONE" ? (
                    <CircleCheck size={15} className="text-emerald-600 shrink-0" />
                  ) : (
                    <Inbox size={15} className="text-slate-400 shrink-0" />
                  )}
                  <span className="text-sm font-bold text-slate-700 truncate">
                    {title.trim() || "ชื่อคอลัมน์"}
                  </span>
                  <span
                    className={`ml-auto text-xs font-bold px-2 py-0.5 rounded-full min-w-5 text-center ${
                      category === "DONE"
                        ? "bg-emerald-100 text-emerald-700"
                        : "bg-slate-200 text-slate-500"
                    }`}
                  >
                    {cardCount}
                  </span>
                </div>
              </div>

              {/* Curated swatch grid with labels + clear selected state */}
              <div className="grid grid-cols-3 gap-1.5">
                {COLUMN_COLOR_PALETTE.map(({ key, hex, label }) => {
                  const isSel = color === key;
                  return (
                    <button
                      key={String(key)}
                      onClick={() => setColor(key)}
                      className={`flex flex-col items-center gap-1 rounded-lg px-2 py-2 border transition-colors ${
                        isSel
                          ? "border-slate-300 bg-slate-50"
                          : "border-transparent hover:bg-slate-50"
                      }`}
                    >
                      <span
                        className={`relative w-6 h-6 rounded-full flex items-center justify-center ring-2 ring-offset-2 ring-offset-white transition ${
                          isSel ? "ring-slate-700" : "ring-transparent"
                        }`}
                        style={{ backgroundColor: hex ?? "#e2e8f0" }}
                      >
                        {isSel && (
                          <Check
                            size={12}
                            strokeWidth={3}
                            className={key ? "text-white" : "text-slate-600"}
                          />
                        )}
                      </span>
                      <span
                        className={`text-[10.5px] font-semibold ${
                          isSel ? "text-slate-700" : "text-slate-400"
                        }`}
                      >
                        {label}
                      </span>
                    </button>
                  );
                })}
              </div>

              <p className="text-[11px] leading-snug text-slate-400">
                สีคือ{" "}
                <span className="font-semibold text-slate-500">identity</span>{" "}
                ของคอลัมน์ ไม่ใช่สถานะ — ทุกสีคุมความสว่างใกล้กันเพื่อให้บอร์ดดูสงบ
              </p>
            </div>
          </div>

          {cardCount > 0 && (
            <div className="mx-5 mb-2 px-3 py-2 text-[11px] text-amber-700 bg-amber-50 border border-amber-200 rounded-lg">
              This column contains {cardCount} cards — please move them all out
              before deleting.
            </div>
          )}

          {/* footer */}
          <div className="flex items-center justify-between px-5 py-4 border-t border-slate-100">
            <button
              onClick={() => setConfirmDeleteOpen(true)}
              disabled={cardCount > 0}
              title={
                cardCount > 0
                  ? "กรุณาย้ายการ์ดออกให้หมดก่อนลบคอลัมน์"
                  : "Delete column"
              }
              className="flex items-center gap-1.5 text-xs font-semibold text-red-500 hover:text-red-600 hover:bg-red-50 px-3 py-2 rounded-lg transition-colors disabled:text-slate-300 disabled:hover:bg-transparent disabled:cursor-not-allowed"
            >
              <Trash2 size={14} /> Delete column
            </button>

            <div className="flex gap-2">
              <button
                onClick={onClose}
                className="px-4 py-2 text-sm rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSave}
                disabled={!title.trim()}
                className="px-4 py-2 text-sm rounded-lg bg-slate-900 text-white font-medium hover:bg-slate-700 disabled:opacity-40 transition-colors"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      </div>

      <ConfirmDialog
        open={confirmDeleteOpen}
        title="Delete column?"
        description="This column is empty and will be permanently removed."
        confirmLabel="Delete"
        destructive
        onConfirm={() => {
          setConfirmDeleteOpen(false);
          onClose();
          onDelete();
        }}
        onCancel={() => setConfirmDeleteOpen(false)}
      />
    </>,
    document.body,
  );
}
