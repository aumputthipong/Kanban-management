import { useBoardStore } from "@/store/useBoardStore";
import { atLeast, type BoardRole as Role } from "@/lib/permissions";

export type BoardRole = Role | null;

export function useBoardRole(): BoardRole {
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const boardMembers = useBoardStore((s) => s.boardMembers);
  if (!currentUserId) return null;
  const me = boardMembers.find((m) => m.user_id === currentUserId);
  if (!me) return null;
  return (me.role as Role) ?? null;
}

export function useCanManageBoard(): boolean {
  return atLeast(useBoardRole(), "manager");
}

export function useCanDeleteBoard(): boolean {
  return useBoardRole() === "owner";
}

export function useCanInviteMembers(): boolean {
  return atLeast(useBoardRole(), "manager");
}

export function useCanManageTags(): boolean {
  return atLeast(useBoardRole(), "manager");
}
