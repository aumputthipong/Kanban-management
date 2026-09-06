import { create } from "zustand";

/**
 * One toast in the top-level `<Toaster>`. `actionLabel` and `onAction` are paired —
 * supply both for an undo affordance, neither for a plain notice.
 */
export interface Toast {
  id: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
  /** Auto-dismiss after this many ms. Pass `0` to keep the toast sticky. */
  duration: number;
}

interface ToastState {
  toasts: Toast[];
  show: (toast: Omit<Toast, "id" | "duration"> & { duration?: number }) => string;
  dismiss: (id: string) => void;
}

/**
 * Global queue for transient UI feedback. `show()` returns the id so a caller can
 * dismiss programmatically; `duration: 0` keeps a toast sticky.
 */
export const useToastStore = create<ToastState>((set) => ({
  toasts: [],
  show: ({ duration = 5000, ...rest }) => {
    const id = `toast_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`;
    set((state) => ({ toasts: [...state.toasts, { id, duration, ...rest }] }));
    if (duration > 0) {
      setTimeout(() => {
        set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) }));
      }, duration);
    }
    return id;
  },
  dismiss: (id) =>
    set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),
}));
