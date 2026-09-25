// undefined = loading, null = no source (the backend returns null, not 404).
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
