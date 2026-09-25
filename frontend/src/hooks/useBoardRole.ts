import { useBoardStore } from "@/store/useBoardStore";

export type BoardRole = "owner" | "manager" | "member" | null;

// UI gating only — the backend re-checks in middleware/board_role.go.
function roleRank(role: BoardRole): number {
  switch (role) {
    case "owner":
      return 3;
    case "manager":
      return 2;
    case "member":
      return 1;
    default:
      return 0;
  }
}

export function useBoardRole(): BoardRole {
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const boardMembers = useBoardStore((s) => s.boardMembers);
  if (!currentUserId) return null;
  const me = boardMembers.find((m) => m.user_id === currentUserId);
  if (!me) return null;
  return (me.role as BoardRole) ?? null;
}

export function useCanManageBoard(): boolean {
  return roleRank(useBoardRole()) >= roleRank("manager");
}

export function useCanDeleteBoard(): boolean {
  return useBoardRole() === "owner";
}

export function useCanInviteMembers(): boolean {
  return roleRank(useBoardRole()) >= roleRank("manager");
}

export function useCanManageTags(): boolean {
  return roleRank(useBoardRole()) >= roleRank("manager");
}
