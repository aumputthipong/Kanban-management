import { apiClient } from "@/lib/apiClient";

export interface InviteLink {
  token: string;
  expires_at: string;
}

export const inviteApi = {
  /** null when there is no active link (204). */
  getActive: (boardId: string) =>
    apiClient<InviteLink | null>(`/boards/${boardId}/invites`),

  /** Revokes any previous link server-side. */
  create: (boardId: string) =>
    apiClient<InviteLink>(`/boards/${boardId}/invites`, { data: {} }),

  revoke: (boardId: string) =>
    apiClient<null>(`/boards/${boardId}/invites`, { method: "DELETE" }),

  accept: (token: string) =>
    apiClient<{ board_id: string }>(
      `/invites/${encodeURIComponent(token)}/accept`,
      { data: {} },
    ),
};
