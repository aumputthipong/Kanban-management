// hooks/useBoardData.ts
import { useState, useEffect } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import { apiClient, ApiError } from "@/lib/apiClient";
import type { Board, Column, BoardMember } from "@/types/board";

interface MeResponse {
  user_id?: string;
}

/**
 * Bootstraps a board view by hydrating `useBoardStore` in three parallel
 * fetches: columns + cards, current user, member list. Should be called
 * once at the board page root — subsequent updates flow through WebSocket
 * broadcasts, not refetches.
 *
 * Returns `{ isLoading, error }` for the page to render skeleton / 404 /
 * generic error states. The string `"NOT_FOUND"` is used as a sentinel for
 * 404 (board missing or not a member) so the page can show a tailored
 * message instead of the generic error UI.
 *
 * Uses `apiClient` (not raw fetch) so 401 → silent token refresh works the
 * same way as the rest of the app. me/members failures fall back to
 * sensible defaults instead of failing the whole bootstrap.
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
        // GET /boards/:id is columns-only — the board's title/icon/color come
        // from the list endpoint (same source the settings page reads). Fall
        // back to leaving boardMeta null so the header shows its default label.
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
