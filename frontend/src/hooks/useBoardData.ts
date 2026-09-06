import { useState, useEffect } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import { apiClient, ApiError } from "@/lib/apiClient";
import type { Board, Column, BoardMember } from "@/types/board";

interface MeResponse {
  user_id?: string;
}

/**
 * Bootstraps a board view by hydrating `useBoardStore` in three parallel fetches.
 * Call once at the board page root — later updates arrive over WebSocket, not refetches.
 * Returns `{ isLoading, error }`; error is the sentinel "NOT_FOUND" for a 404 so the
 * page can distinguish a missing board from a generic failure.
 */
export function useBoardData(boardId: string) {
  const { setColumns, setCurrentUser, setBoardMembers, setBoardMeta, setLoading } =
    useBoardStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!boardId) return;
    let cancelled = false;

    const fetchAll = async () => {
      setIsLoading(true);
      setLoading(true);
      setError(null);
      try {
        const [boardRes, meRes, membersRes, listRes] = await Promise.allSettled([
          apiClient<Column[]>(`/boards/${boardId}`),
          apiClient<MeResponse>(`/auth/me`),
          apiClient<BoardMember[]>(`/boards/${boardId}/members`),
          apiClient<Board[]>(`/boards`),
        ]);

        if (cancelled) return;

        if (boardRes.status === "rejected") {
          const err = boardRes.reason;
          if (err instanceof ApiError && err.status === 404) {
            setError("NOT_FOUND");
            return;
          }
          throw err;
        }

        setColumns(boardRes.value);
        if (meRes.status === "fulfilled" && meRes.value?.user_id) {
          setCurrentUser(meRes.value.user_id);
        }
        // GET /boards/:id is columns-only — title/icon/color come from the list
        // endpoint. Leaving boardMeta null makes the header fall back to its default.
        if (listRes.status === "fulfilled" && Array.isArray(listRes.value)) {
          const meta = listRes.value.find((b) => b.id === boardId);
          setBoardMeta(
            meta ? { title: meta.title, color: meta.color, icon: meta.icon } : null,
          );
        }
        const members =
          membersRes.status === "fulfilled" && Array.isArray(membersRes.value)
            ? membersRes.value.filter(Boolean)
            : [];
        setBoardMembers(members);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : "Unknown error");
      } finally {
        if (!cancelled) {
          setIsLoading(false);
          setLoading(false);
        }
      }
    };

    fetchAll();
    return () => {
      cancelled = true;
    };
  }, [boardId, setColumns, setCurrentUser, setBoardMembers, setBoardMeta, setLoading]);

  return { isLoading, error };
}
