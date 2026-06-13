import { apiClient } from "@/lib/apiClient";

export interface InviteLink {
  token: string;
  /** ISO timestamp when the link stops working. */
  expires_at: string;
}

// Board invite links. create/get/revoke are manager-gated (board-scoped);
// accept is the public-ish join action (auth + valid token).
export const inviteApi = {
  /** Current active link for the board, or null when there isn't one (204). */
  getActive: (boardId: string) =>
    apiClient<InviteLink | null>(`/boards/${boardId}/invites`),

  /** (Re)generate the board's link — revokes any prior link server-side. */
  create: (boardId: string) =>
    apiClient<InviteLink>(`/boards/${boardId}/invites`, { data: {} }),

  /** Turn off the active link. */
  revoke: (boardId: string) =>
    apiClient<null>(`/boards/${boardId}/invites`, { method: "DELETE" }),

  /** Join the board the token points at. Returns the board id to navigate to. */
  accept: (token: string) =>
    apiClient<{ board_id: string }>(
      `/invites/${encodeURIComponent(token)}/accept`,
      { data: {} },
    ),
};
