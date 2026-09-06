// Planning origin of a card, for the detail modal's "source" section. The backend
// returns JSON null (not 404) when there is none, so state is undefined (loading),
// null (no source), or the object.
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
  // Reset during render when the consumer switches cards, not in an effect (AGENTS.md).
  const [trackedCardId, setTrackedCardId] = useState<string | null>(cardId);
  if (trackedCardId !== cardId) {
    setTrackedCardId(cardId);
    setSource(undefined);
    setIsLoading(Boolean(cardId));
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
        // Treat a fetch failure as "no source" — apiClient already toasts real errors.
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
