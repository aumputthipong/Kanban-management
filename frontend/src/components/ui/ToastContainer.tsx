"use client";

import { useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { useToastStore } from "@/store/useToastStore";

// False on the server and on the first client render so the hydrated tree matches,
// true afterwards. Use this, not a bare `typeof window` check, to gate client-only DOM
// access — the bare check renders differently on each side and trips hydration.
const subscribe = () => () => {};
function useIsClient(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => true,   // client snapshot
    () => false,  // server snapshot
  );
}

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);
  const isClient = useIsClient();

  // createPortal needs document.body. Until we're past the hydration commit
  // we render nothing — same on server and client first render — so React
  // can match up the trees.
  if (!isClient) return null;

  return createPortal(
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-10000 flex flex-col gap-2 pointer-events-none">
      {toasts.map((t) => (
        <div
          key={t.id}
          className="pointer-events-auto flex items-center gap-3 px-4 py-2.5 bg-slate-900 text-white rounded-lg shadow-lg text-sm animate-in fade-in slide-in-from-bottom-2"
        >
          <span>{t.message}</span>
          {t.actionLabel && t.onAction && (
            <button
              onClick={() => {
                t.onAction?.();
                dismiss(t.id);
              }}
              className="px-2 py-0.5 text-blue-300 hover:text-blue-200 font-semibold uppercase text-xs tracking-wide cursor-pointer"
            >
              {t.actionLabel}
            </button>
          )}
          <button
            onClick={() => dismiss(t.id)}
            className="text-slate-400 hover:text-white transition-colors cursor-pointer"
          >
            <X size={14} />
          </button>
        </div>
      ))}
    </div>,
    document.body,
  );
}
