import { useState, useEffect } from "react";
import type { BoardMember } from "@/types/board";
import { apiClient } from "@/lib/apiClient";
import { useBoardStore } from "@/store/useBoardStore";

export function useBoardMembers(boardId: string) {
  const [members, setMembers] = useState<BoardMember[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isAdding, setIsAdding] = useState(false);
  const [loadingId, setLoadingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { setBoardMembers } = useBoardStore();

  useEffect(() => {
    let cancelled = false;
    setIsLoading(true);
    const loadData = async () => {
      try {
        const membersData = await apiClient(`/boards/${boardId}/members`);
        if (cancelled) return;
        setMembers(Array.isArray(membersData) ? membersData.filter(Boolean) : []);
      } catch {
        if (!cancelled) setError("Failed to load members.");
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    };

    loadData();
    return () => {
      cancelled = true;
    };
  }, [boardId]);

  // Invite by exact email — no user list is fetched/exposed (privacy). Returns
  // an error message to show inline (e.g. "user not found") or null on success.
  const addMember = async (email: string, role: string): Promise<string | null> => {
    setIsAdding(true);
    try {
      await apiClient(`/boards/${boardId}/members`, {
        data: { email, role },
      });
      // Re-fetch the full list so the UI reflects the actual DB state
      const fresh: BoardMember[] = await apiClient(`/boards/${boardId}/members`);
      const cleaned = Array.isArray(fresh) ? fresh.filter(Boolean) : [];
      setMembers(cleaned);
      setBoardMembers(cleaned); // sync Zustand so MemberFilterBar also updates
      return null;
    } catch (err) {
      return err instanceof Error ? err.message : "เพิ่มสมาชิกไม่สำเร็จ";
    } finally {
      setIsAdding(false);
    }
  };
  const removeMember = async (userId: string) => {
    setLoadingId(userId);
    try {
      // apiClient handles credentials and error-status checking.
      await apiClient(`/boards/${boardId}/members/${userId}`, {
        method: "DELETE",
      });

      setMembers((prev) => prev.filter((m) => m.user_id !== userId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setLoadingId(null);
    }
  };

  const changeRole = async (userId: string, role: string) => {
    setLoadingId(userId);
    try {
      // apiClient sets the headers and parses the JSON response.
      await apiClient(`/boards/${boardId}/members/${userId}`, {
        method: "PATCH",
        data: { role },
      });

      setMembers((prev) =>
        prev.map((m) => (m.user_id === userId ? { ...m, role: role as BoardMember["role"] } : m))
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setLoadingId(null);
    }
  };

  const leaveBoard = async () => {
    setError(null);
    try {
      await apiClient(`/boards/${boardId}/members/me`, { method: "DELETE" });
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
      return false;
    }
  };

  return {
    members,
    isLoading,
    isAdding,
    loadingId,
    error,
    addMember,
    removeMember,
    changeRole,
    leaveBoard,
  };
}