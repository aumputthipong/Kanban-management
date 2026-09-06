import { useBoardStore } from "@/store/useBoardStore";

/**
 * Caller's role on the viewed board, or null if not a member. Mirrors core.BoardRole;
 * docs/ARCHITECTURE.md has the permission matrix.
 */
export type BoardRole = "owner" | "manager" | "member" | null;

/**
 * Numeric rank so `>=` comparisons work without listing every role.
 * UI gating only — the backend re-checks in internal/middleware/board_role.go.
 */
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

/**
 * Reads the current user's role on the active board from `useBoardStore`.
 * Returns null while the board is still loading or if the user isn't a
 * member. Use this with `useCanManageBoard` etc. for declarative UI gating.
 */
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
