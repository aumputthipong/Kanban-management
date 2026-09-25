import { create } from "zustand";

// One-shot "scroll to + highlight" cue for ?card=<id> deep links. Not persisted.
interface CardHighlightState {
  targetId: string | null;
  setTarget: (id: string | null) => void;
}

export const useCardHighlightStore = create<CardHighlightState>((set) => ({
  targetId: null,
  setTarget: (id) => set({ targetId: id }),
}));
