// Fetches the planning origin of a card for the card detail modal's
// "source" section. Backend responds with JSON `null` (not 404) when the
// card wasn't promoted from planning, so the loaded state has three
// possible values: undefined (still loading), null (no source), or the
// CardSource object.
import { useEffect, useState } from "react";
import { planningApi } from "@/lib/planningApi";
import type { CardSource } from "@/types/planning";

interface State {
  source: CardSource | null | undefined;
  isLoading: boolean;
}

export function useCardSource(cardId: string | null): State {
  const [source, setSource] = useState<CardSource | null | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(false);
  // Track the cardId we last started a fetch for. When the consumer
  // navigates between cards (or closes the modal entirely), we reset
  // local state synchronously during render — calling setSource(undefined)
  // inside the effect tripped react-hooks/set-state-in-effect under
  // React 19, and the cascading render that pattern caused was the
  // actual problem the rule is trying to surface.
  const [trackedCardId, setTrackedCardId] = useState<string | null>(cardId);
  if (trackedCardId !== cardId) {
    setTrackedCardId(cardId);
    setSource(undefined);
    // Flip the loading flag synchronously during render — moving this out
    // of the effect was needed to satisfy react-hooks/set-state-in-effect.
    // Setting it during render is fine because the next render runs
    // immediately anyway (we just set state), and the effect below picks
    // up the new cardId on the same commit.
    setIsLoading(Boolean(cardId));
  }

  useEffect(() => {
    if (!cardId) return;
    let cancelled = false;
    planningApi
      .getCardSource(cardId)
      .then((result) => {
        if (!cancelled) setSource(result);
      })
      .catch(() => {
        // Treat fetch failure as "no source" so the modal stays usable.
        // Real auth/network issues already surface via apiClient toasts.
        if (!cancelled) setSource(null);
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [cardId]);

  return { source, isLoading };
}
