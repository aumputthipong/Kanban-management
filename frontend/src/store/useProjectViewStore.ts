import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ProjectViewMode = "grid" | "list";

interface ProjectViewState {
  viewMode: ProjectViewMode;
  setViewMode: (mode: ProjectViewMode) => void;
}

// Persists the project-list layout choice (grid vs list) in localStorage,
// mirroring useSidebarStore — same persist middleware, its own key. The
// value rehydrates after mount, so the first paint uses the default and then
// snaps to the remembered choice (identical behaviour to the sidebar).
export const useProjectViewStore = create<ProjectViewState>()(
  persist(
    (set) => ({
      viewMode: "grid",
      setViewMode: (mode) => set({ viewMode: mode }),
    }),
    { name: "project-view-mode" },
  ),
);
