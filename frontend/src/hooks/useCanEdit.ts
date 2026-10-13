import { useBoardStore } from "@/store/useBoardStore";
import { useBoardRole } from "@/hooks/useBoardRole";
import { canDeleteCard, canEditCard } from "@/lib/permissions";
import type { Card } from "@/types/board";

/** UI gating only — the backend re-checks every write. */
export function useCanEdit(card: Card): boolean {
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const role = useBoardRole();
  if (!currentUserId) return false;
  return canEditCard(role, card.created_by === currentUserId, card.assignee_id === currentUserId);
}

export function useCanDeleteCard(card: Card): boolean {
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const role = useBoardRole();
  if (!currentUserId) return false;
  return canDeleteCard(role, card.created_by === currentUserId);
}
