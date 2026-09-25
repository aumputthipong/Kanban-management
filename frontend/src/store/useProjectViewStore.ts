import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ProjectViewMode = "grid" | "list";

interface ProjectViewState {
  viewMode: ProjectViewMode;
  setViewMode: (mode: ProjectViewMode) => void;
}

// Rehydrates after mount, so the first paint uses the default.
export const useProjectViewStore = create<ProjectViewState>()(
  persist(
    (set) => ({
      viewMode: "grid",
      setViewMode: (mode) => set({ viewMode: mode }),
    }),
    { name: "project-view-mode" },
  ),
);
