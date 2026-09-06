import { useBoardStore } from "@/store/useBoardStore";
import type { Card } from "@/types/board";

/**
 * Can the current user edit this card — creator, assignee, or board owner/manager.
 * UI gating only; the backend re-checks and stays the source of truth.
 */
export function useCanEdit(card: Card): boolean {
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const boardMembers = useBoardStore((s) => s.boardMembers);

  if (!currentUserId) return false;

  if (card.created_by === currentUserId) return true;

  if (card.assignee_id === currentUserId) return true;

  const member = boardMembers.find((m) => m.user_id === currentUserId);
  if (member && (member.role === "owner" || member.role === "manager")) return true;

  return false;
}
