import { create } from "zustand";
import type { Activity } from "@/types/activity";

interface ActivityState {
  activities: Activity[];
  loading: boolean;
  error: string | null;
  setActivities: (items: Activity[]) => void;
  prependActivity: (item: Activity) => void;
  setLoading: (v: boolean) => void;
  setError: (v: string | null) => void;
  reset: () => void;
}

/**
 * Activity feed for the viewed board. Separate from `useBoardStore` because it grows
 * append-only and is reset on every board change. `prependActivity` (called on
 * ACTIVITY_CREATED) de-duplicates on id and caps the list at 100.
 */
export const useActivityStore = create<ActivityState>((set) => ({
  activities: [],
  loading: false,
  error: null,
  setActivities: (items) => set({ activities: items }),
  prependActivity: (item) =>
    set((s) =>
      s.activities.some((a) => a.id === item.id)
        ? s
        : { activities: [item, ...s.activities].slice(0, 100) },
    ),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),
  reset: () => set({ activities: [], loading: false, error: null }),
}));
