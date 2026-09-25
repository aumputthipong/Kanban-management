"use client";
import { useEffect } from "react";
import { API_URL } from "@/lib/constants";
import { useActivityStore } from "@/store/useActivityStore";

export function useActivityFeed(boardId: string | undefined, limit = 30) {
  const activities = useActivityStore((s) => s.activities);
  const loading = useActivityStore((s) => s.loading);
  const error = useActivityStore((s) => s.error);

  useEffect(() => {
    if (!boardId) return;
    const store = useActivityStore.getState();
    store.reset();
    store.setLoading(true);

    let cancelled = false;
    (async () => {
      try {
        const res = await fetch(`${API_URL}/boards/${boardId}/activities?limit=${limit}`, {
          credentials: "include",
        });
        if (!res.ok) throw new Error(`Failed to fetch activities: ${res.status}`);
        const data = await res.json();
        if (cancelled) return;
        useActivityStore.getState().setActivities(Array.isArray(data) ? data : []);
      } catch (err) {
        if (cancelled) return;
        useActivityStore.getState().setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        if (!cancelled) useActivityStore.getState().setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [boardId, limit]);

  return { activities, loading, error };
}
