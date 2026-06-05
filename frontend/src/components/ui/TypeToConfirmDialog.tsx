"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AlertTriangle, Loader2 } from "lucide-react";

interface TypeToConfirmDialogProps {
  open: boolean;
  title: string;
  description?: React.ReactNode;
  /** The exact string the user must type to enable the confirm button. */
  confirmPhrase: string;
  /** Label above the input, e.g. "พิมพ์ชื่อบอร์ดเพื่อยืนยัน". */
  inputLabel: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * GitHub-style "type to confirm" dialog for irreversible actions. The confirm
 * button stays disabled until the user types `confirmPhrase` exactly — guards
 * against accidental permanent deletes. Always destructive styling.
 */
export function TypeToConfirmDialog({
  open,
  title,
  description,
  confirmPhrase,
  inputLabel,
  confirmLabel = "ลบถาวร",
  cancelLabel = "ยกเลิก",
  loading = false,
  onConfirm,
  onCancel,
}: TypeToConfirmDialogProps) {
  const [value, setValue] = useState("");
  const [wasOpen, setWasOpen] = useState(open);
  const inputRef = useRef<HTMLInputElement>(null);

  // Reset the typed value when the dialog transitions closed→open. This is the
  // setState-during-render pattern (React 19 forbids synchronous setState in an
  // effect body) — track the previous `open`, compare in render, reset before
  // returning JSX.
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) setValue("");
  }

  // Focus the input after the portal mounts (DOM side-effect, no setState).
  useEffect(() => {
    if (open) {
      const t = setTimeout(() => inputRef.current?.focus(), 50);
      return () => clearTimeout(t);
    }
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onCancel]);

  if (!open) return null;

  const matched = value === confirmPhrase;
  const canConfirm = matched && !loading;

  return createPortal(
    <div
      className="fixed inset-0 z-9999 flex items-center justify-center"
      onClick={loading ? undefined : onCancel}
    >
      <div className="absolute inset-0 bg-black/40" />

      <div
        className="relative z-10 w-full max-w-md mx-4 bg-white rounded-2xl shadow-2xl p-6 flex flex-col gap-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-3">
          <span className="shrink-0 w-9 h-9 rounded-lg bg-red-50 text-red-600 flex items-center justify-center">
            <AlertTriangle size={18} />
          </span>
          <div className="flex flex-col gap-1 min-w-0">
            <h2 className="text-base font-bold text-slate-900">{title}</h2>
            {description && (
              <div className="text-[13px] text-slate-500 leading-relaxed">
                {description}
              </div>
            )}
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <label className="text-[12.5px] font-semibold text-slate-600">
            {inputLabel}
          </label>
          <input
            ref={inputRef}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && canConfirm) onConfirm();
            }}
            disabled={loading}
            autoComplete="off"
            spellCheck={false}
            className="h-[42px] px-3.5 rounded-lg border border-slate-300 bg-white text-sm text-slate-900 outline-none transition focus:border-red-400 focus:ring-2 focus:ring-red-500/10 disabled:opacity-60"
          />
        </div>

        <div className="flex gap-2 justify-end pt-1">
          <button
            onClick={onCancel}
            disabled={loading}
            className="px-4 py-2 text-sm rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors disabled:opacity-40"
          >
            {cancelLabel}
          </button>
          <button
            onClick={onConfirm}
            disabled={!canConfirm}
            className="px-4 py-2 text-sm rounded-lg font-semibold text-white bg-red-600 hover:bg-red-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed inline-flex items-center gap-2"
          >
            {loading && <Loader2 size={15} className="animate-spin" />}
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
