import { create } from "zustand";

// Ephemeral "scroll to + briefly highlight this card" signal. Set when arriving
// on a board via a deep link (e.g. open-in-board from My Work with ?card=<id>);
// the matching TaskCard scrolls itself into view, shows a short ring, then
// clears the target. Not persisted — it's a one-shot navigation cue.
interface CardHighlightState {
  targetId: string | null;
  setTarget: (id: string | null) => void;
}

export const useCardHighlightStore = create<CardHighlightState>((set) => ({
  targetId: null,
  setTarget: (id) => set({ targetId: id }),
}));
