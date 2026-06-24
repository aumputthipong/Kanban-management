import { useBoardStore } from "@/store/useBoardStore";
import { useBoardWebSocket } from "@/contexts/BoardWebSocketContext";
import { POSITION_GAP } from "@/utils/boardPosition";
import { apiClient, ApiError } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";
import { WS_EVENT } from "@/types/wsEvents";
import type { Card, CardUpdateForm } from "@/types/board";

export function useCardActions(boardId: string) {
  const { sendMessage } = useBoardWebSocket();

  const handleToggleDone = (card: Card) => {
    sendMessage({
      type: WS_EVENT.CardDoneToggled,
      payload: {
        card_id: card.id,
        board_id: boardId,
        is_done: !card.is_done,
      },
    });
  };

  // opts lets the Create Task modal seed assignee / priority / due date /
  // description / subtasks in one shot. Callers that omit opts (or pass null /
  // empty fields) send just column_id + title + position — the original
  // quick-add WS payload. Subtasks are titles only; the backend creates them
  // alongside the card in one transaction (the card id doesn't exist yet).
  const handleAddCard = (
    columnId: string,
    title: string,
    opts?: {
      assigneeId?: string | null;
      priority?: string | null;
      dueDate?: string | null;
      description?: string | null;
      subtasks?: string[];
    },
  ) => {
    const { columns } = useBoardStore.getState();
    const col = columns.find((c) => c.id === columnId);
    const sorted = col ? [...col.cards].sort((a, b) => a.position - b.position) : [];
    const lastCard = sorted[sorted.length - 1];
    const newPosition = lastCard ? lastCard.position + POSITION_GAP : POSITION_GAP;

    sendMessage({
      type: WS_EVENT.CardCreated,
      payload: {
        column_id: columnId,
        title,
        position: newPosition,
        ...(opts?.assigneeId ? { assignee_id: opts.assigneeId } : {}),
        ...(opts?.priority ? { priority: opts.priority } : {}),
        ...(opts?.dueDate ? { due_date: opts.dueDate } : {}),
        ...(opts?.description ? { description: opts.description } : {}),
        ...(opts?.subtasks && opts.subtasks.length > 0 ? { subtasks: opts.subtasks } : {}),
      },
    });
  };

  const handleChangeColumn = (cardId: string, toColumnId: string) => {
    const freshColumns = useBoardStore.getState().columns;
    const currentCol = freshColumns.find((col) =>
      col.cards.some((c) => c.id === cardId),
    );
    if (!currentCol || currentCol.id === toColumnId) return;

    const targetCol = freshColumns.find((c) => c.id === toColumnId);
    if (!targetCol) return;

    const sorted = [...targetCol.cards].sort((a, b) => a.position - b.position);
    const last = sorted[sorted.length - 1];
    const newPosition = last ? last.position + POSITION_GAP : POSITION_GAP;

    useBoardStore.getState().moveCard(cardId, toColumnId, newPosition);

    sendMessage({
      type: WS_EVENT.CardMoved,
      payload: {
        card_id: cardId,
        old_column_id: currentCol.id,
        new_column_id: toColumnId,
        position: newPosition,
      },
    });
  };

  const handleDeleteCard = (cardId: string) => {
    sendMessage({
      type: WS_EVENT.CardDeleted,
      payload: { card_id: cardId },
    });
  };

  const handleUpdateCard = (cardId: string, form: CardUpdateForm) => {
    const { boardMembers, columns, updateCard } = useBoardStore.getState();
    const newAssigneeId = form.assignee_id || null;
    const original = columns.flatMap((c) => c.cards).find((c) => c.id === cardId);
    // Diff form vs current card to build changed_fields — only fields that actually changed
    // are sent as activity log entries (tags included).
    const changedFields: string[] = [];
    if (original) {
      const newEstimated = form.estimated_hours ? parseFloat(form.estimated_hours) : null;
      if (form.title !== original.title) changedFields.push("title");
      if ((form.description || null) !== (original.description ?? null)) changedFields.push("description");
      if ((form.due_date || null) !== (original.due_date ?? null)) changedFields.push("due_date");
      if (newAssigneeId !== (original.assignee_id ?? null)) changedFields.push("assignee_id");
      if ((form.priority || null) !== (original.priority ?? null)) changedFields.push("priority");
      if (newEstimated !== (original.estimated_hours ?? null)) changedFields.push("estimated_hours");
      const oldTagIds = new Set((original.tags ?? []).map((t) => t.id));
      const newTagIds = new Set(form.tags.map((t) => t.id));
      const tagsChanged =
        oldTagIds.size !== newTagIds.size ||
        [...newTagIds].some((id) => !oldTagIds.has(id));
      if (tagsChanged) changedFields.push("tags");
    }
    const newAssigneeName = newAssigneeId
      ? (boardMembers.find((m) => m.user_id === newAssigneeId)?.full_name ?? null)
      : null;

    // Optimistic store update (includes tags)
    updateCard({
      ...original!,
      title: form.title,
      description: form.description || null,
      due_date: form.due_date || null,
      assignee_id: newAssigneeId,
      assignee_name: newAssigneeName,
      priority: (form.priority as Card["priority"]) || null,
      estimated_hours: form.estimated_hours ? parseFloat(form.estimated_hours) : null,
      tags: form.tags,
      acceptance_criteria: form.acceptance_criteria || null,
      implementation_note: form.implementation_note || null,
    });

    // Persist via REST (handles tag_ids through service layer). For
    // acceptance_criteria / implementation_note we only include the field
    // in the body when it actually changed — backend uses COALESCE on
    // these two, so an omitted field preserves the existing value (matters
    // for cards that PromoteItem copied AC into; a "title only" edit must
    // not clobber the source acceptance criteria).
    type CardPatchBody = {
      title: string;
      description: string | null;
      due_date: string | null;
      assignee_id: string | null;
      priority: string | null;
      estimated_hours: number | null;
      tag_ids: string[];
      acceptance_criteria?: string;
      implementation_note?: string;
    };
    const body: CardPatchBody = {
      title: form.title,
      description: form.description || null,
      due_date: form.due_date || null,
      assignee_id: newAssigneeId,
      priority: form.priority || null,
      estimated_hours: form.estimated_hours ? parseFloat(form.estimated_hours) : null,
      tag_ids: form.tags.map((t) => t.id),
    };
    if (original) {
      if (form.acceptance_criteria !== (original.acceptance_criteria ?? "")) {
        body.acceptance_criteria = form.acceptance_criteria;
        changedFields.push("acceptance_criteria");
      }
      if (form.implementation_note !== (original.implementation_note ?? "")) {
        body.implementation_note = form.implementation_note;
        changedFields.push("implementation_note");
      }
    }
    // The optimistic store update + WS broadcast below already reflect the
    // change. Persist through apiClient (not raw fetch): a raw fetch only
    // rejects on a network error, so a 4xx/5xx response used to be swallowed
    // and the edit silently vanished on next reload. apiClient throws on any
    // non-2xx (and toasts on 403); surface a toast for the rest so a failed
    // save is visible and the user can retry.
    apiClient(`/cards/${cardId}`, { method: "PATCH", data: body }).catch((err) => {
      if (err instanceof ApiError && err.status === 403) return; // apiClient already toasted
      useToastStore.getState().show({
        message: "บันทึกการ์ดไม่สำเร็จ — ลองอีกครั้ง",
        duration: 4000,
      });
    });

    sendMessage({
      type: WS_EVENT.CardUpdated,
      payload: {
        card_id: cardId,
        title: form.title,
        description: form.description || null,
        due_date: form.due_date || null,
        assignee_id: newAssigneeId,
        assignee_name: newAssigneeName,
        priority: form.priority || null,
        estimated_hours: form.estimated_hours ? parseFloat(form.estimated_hours) : null,
        changed_fields: changedFields,
      },
    });
  };

  return { handleToggleDone, handleAddCard, handleChangeColumn, handleDeleteCard, handleUpdateCard };
}
