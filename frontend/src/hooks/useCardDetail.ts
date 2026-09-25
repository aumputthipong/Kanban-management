import { useEffect, useState } from "react";
import { apiClient } from "@/lib/apiClient";
import type { Card } from "@/types/board";

export function useCardDetail(cardId: string) {
  const [detail, setDetail] = useState<Card | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    apiClient<Card>(`/cards/${cardId}`)
      .then((c) => {
        if (!cancelled) {
          setDetail(c);
          setIsLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [cardId]);

  return { detail, setDetail, isLoading };
}
