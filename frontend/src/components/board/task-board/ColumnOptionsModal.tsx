"use client";

import { useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { Check, CircleCheck, Trash2, X } from "lucide-react";
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

// ── D1 "Solid Cap" colour derivations ──────────────────────────────────────────
// Shared between the column (Column.tsx) and this modal's live preview so the
// preview mirrors the real thing. `accent` is the column identity colour.

/** Column identity colour: the custom colour, or a neutral/emerald category fallback. */
export function columnAccentColor(hex: string | null, isDone: boolean): string {
  return hex ?? (isDone ? "#10b981" : "#94a3b8");
}

/** Solid-cap fill: deepen the identity colour toward ink so white cap text stays
 *  legible across the light palette (raw pastels fail white contrast). */
export function columnCapColor(accent: string): string {
  return `color-mix(in oklab, ${accent} 68%, #0f172a)`;
}

/** Column body: a faint wash of the identity hue; `hot` = drag-over highlight. */
export function columnBodyBg(accent: string, hot = false): string {
  return `color-mix(in oklab, ${accent} ${hot ? 12 : 6}%, white)`;
}

/** Column body border: a light tint of the identity hue; `hot` = drag-over. */
export function columnBodyBorder(accent: string, hot = false): string {
  return `color-mix(in oklab, ${accent} ${hot ? 42 : 20}%, white)`;
}

// ── props ─────────────────────────────────────────────────────────────────────
interface ColumnOptionsModalProps {
  open: boolean;
  /** "create" hides Delete + the card warning and titles the panel "New column". */
  mode?: "edit" | "create";
  initialTitle: string;
  initialCategory: "TODO" | "DONE";
  initialColor: string | null;
  cardCount: number;
  onSave: (
    title: string,
    category: "TODO" | "DONE",
    color: string | null,
  ) => void;
  /** Omitted in create mode (nothing to delete yet). */
  onDelete?: () => void;
  onClose: () => void;
}

// ── component ─────────────────────────────────────────────────────────────────
export function ColumnOptionsModal({
  open,
  mode = "edit",
  initialTitle,
  initialCategory,
  initialColor,
  cardCount,
  onSave,
  onDelete,
  onClose,
}: ColumnOptionsModalProps) {
  const isCreate = mode === "create";
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
  // Mirror the real column's D1 "Solid Cap" formulae so the preview is truthful.
  const accent = columnAccentColor(selectedHex, category === "DONE");
  const capColor = columnCapColor(accent);
  const bodyBg = columnBodyBg(accent);

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
            <h2 className="text-sm font-bold text-slate-800">
              {isCreate ? "New column" : "Column options"}
            </h2>
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
            </div>

            {/* ── color ── identity, with a live header preview ── */}
            <div className="flex flex-col gap-2">
              <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
                Color
              </label>

              {/* Live preview of the actual column — D1 Solid Cap */}
              <div className="rounded-lg border border-slate-100 overflow-hidden">
                <div
                  className="flex items-center gap-2 px-3 h-9"
                  style={{ backgroundColor: capColor }}
                >
                  {category === "DONE" && (
                    <CircleCheck size={14} className="text-white shrink-0" />
                  )}
                  <span className="flex-1 min-w-0 truncate text-sm font-bold text-white">
                    {title.trim() || "ชื่อคอลัมน์"}
                  </span>
                  <span className="min-w-5 rounded-full bg-white/25 px-2 py-0.5 text-center text-xs font-bold text-white">
                    {cardCount}
                  </span>
                </div>
                <div className="px-3 py-2.5" style={{ backgroundColor: bodyBg }}>
                  <div className="h-2 w-2/3 rounded bg-white/80" />
                </div>
              </div>

              {/* Compact swatch row — colour only (names live in the tooltip)
                  to keep the modal short. */}
              <div className="flex flex-wrap gap-2">
                {COLUMN_COLOR_PALETTE.map(({ key, hex, label }) => {
                  const isSel = color === key;
                  return (
                    <button
                      key={String(key)}
                      onClick={() => setColor(key)}
                      title={label}
                      aria-label={label}
                      className={`w-7 h-7 rounded-full flex items-center justify-center ring-2 ring-offset-2 ring-offset-white transition ${
                        isSel ? "ring-slate-700" : "ring-transparent hover:ring-slate-300"
                      }`}
                      style={{ backgroundColor: hex ?? "#e2e8f0" }}
                    >
                      {isSel && (
                        <Check
                          size={13}
                          strokeWidth={3}
                          className={key ? "text-white" : "text-slate-600"}
                        />
                      )}
                    </button>
                  );
                })}
              </div>
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
            {isCreate ? (
              <span />
            ) : (
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
            )}

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
                {isCreate ? "Create" : "Save"}
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
          onDelete?.();
        }}
        onCancel={() => setConfirmDeleteOpen(false)}
      />
    </>,
    document.body,
  );
}
