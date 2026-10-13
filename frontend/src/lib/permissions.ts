// Mirrors backend/internal/core/permissions.go for UI gating only; the backend decides.

export type BoardRole = "owner" | "manager" | "member";

const RANK: Record<BoardRole, number> = { owner: 3, manager: 2, member: 1 };

export function atLeast(role: BoardRole | null, min: BoardRole): boolean {
  return role !== null && RANK[role] >= RANK[min];
}

export function canEditCard(role: BoardRole | null, isCreator: boolean, isAssignee: boolean): boolean {
  return isCreator || isAssignee || atLeast(role, "manager");
}

export function canDeleteCard(role: BoardRole | null, isCreator: boolean): boolean {
  return isCreator || atLeast(role, "manager");
}

export function canManageColumns(role: BoardRole | null): boolean {
  return atLeast(role, "manager");
}

export function canManageMember(actor: BoardRole | null, target: BoardRole): boolean {
  if (actor === "owner") return target !== "owner";
  if (actor === "manager") return target === "member";
  return false;
}

export function grantableRoles(actor: BoardRole | null): BoardRole[] {
  if (actor === "owner") return ["manager", "member"];
  if (actor === "manager") return ["member"];
  return [];
}
