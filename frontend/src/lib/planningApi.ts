import { apiClient } from "@/lib/apiClient";
import type {
  CardSource,
  PlanningComment,
  PlanningSessionSummary,
  PlanningSessionDetail,
  PlanningItem,
  PlanningItemType,
  PlanningItemStatus,
} from "@/types/planning";

export const planningApi = {
  listSessions: (boardId: string) =>
    apiClient<PlanningSessionSummary[]>(
      `/boards/${boardId}/planning/sessions`,
    ),

  createSession: (
    boardId: string,
    data: { title: string; label?: string; meeting_at?: string },
  ) =>
    apiClient<PlanningSessionSummary>(
      `/boards/${boardId}/planning/sessions`,
      { data },
    ),

  getSession: (sessionId: string) =>
    apiClient<PlanningSessionDetail>(
      `/planning/sessions/${sessionId}`,
    ),

  updateSession: (
    sessionId: string,
    data: { title?: string; label?: string | null; meeting_at?: string | null },
  ) =>
    apiClient<PlanningSessionSummary>(
      `/planning/sessions/${sessionId}`,
      { method: "PATCH", data },
    ),

  deleteSession: (sessionId: string) =>
    apiClient<null>(`/planning/sessions/${sessionId}`, {
      method: "DELETE",
    }),

  createItem: (
    sessionId: string,
    data: { type: PlanningItemType; title: string; description?: string | null },
  ) =>
    apiClient<PlanningItem>(
      `/planning/sessions/${sessionId}/items`,
      { data },
    ),

  updateItem: (
    itemId: string,
    data: {
      type?: PlanningItemType;
      title?: string;
      description?: string | null;
      status?: PlanningItemStatus;
      position?: number;
      acceptance_criteria?: string | null;
      implementation_note?: string | null;
    },
  ) =>
    apiClient<PlanningItem>(`/planning/items/${itemId}`, {
      method: "PATCH",
      data,
    }),

  deleteItem: (itemId: string) =>
    apiClient<null>(`/planning/items/${itemId}`, { method: "DELETE" }),

  promoteItem: (itemId: string) =>
    apiClient<{ item: PlanningItem; card_id: string }>(
      `/planning/items/${itemId}/promote`,
      { method: "POST", data: {} },
    ),

  // null when the card wasn't promoted from planning.
  getCardSource: (cardId: string) =>
    apiClient<CardSource | null>(`/cards/${cardId}/source`),

  // Includes soft-deleted rows (body = null) so the thread doesn't shift.
  listComments: (itemId: string) =>
    apiClient<PlanningComment[]>(`/planning/items/${itemId}/comments`),

  createComment: (itemId: string, body: string) =>
    apiClient<PlanningComment>(`/planning/items/${itemId}/comments`, {
      data: { body },
    }),

  editComment: (commentId: string, body: string) =>
    apiClient<PlanningComment>(`/planning/comments/${commentId}`, {
      method: "PATCH",
      data: { body },
    }),

  deleteComment: (commentId: string) =>
    apiClient<null>(`/planning/comments/${commentId}`, { method: "DELETE" }),
};
