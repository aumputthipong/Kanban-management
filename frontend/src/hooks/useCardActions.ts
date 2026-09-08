import { useBoardStore } from "@/store/useBoardStore";
import { POSITION_GAP } from "@/utils/boardPosition";
import { apiClient, ApiError } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";
import type { Card, CardUpdateForm, Column } from "@/types/board";

// Restores the board to `snapshot` and tells the user the write did not land. 403 is
// skipped because apiClient has already toasted it.
function revertWith(snapshot: Column[], message: string) {
  return (err: unknown) => {
    useBoardStore.getState().setColumns(snapshot);
    if (err instanceof ApiError && err.status === 403) return;
    useToastStore.getState().show({ message, duration: 4000 });
  };
}

export function useCardActions() {

  const handleToggleDone = (card: Card) => {
    const snapshot = useBoardStore.getState().columns;
    const isDone = !card.is_done;
    // Tick the box immediately; the move into (or out of) the DONE column arrives
    // with the broadcast, which this client receives like any other.
    useBoardStore.getState().updateCard({ ...card, is_done: isDone });
    apiClient(`/cards/${card.id}/done`, { method: "PATCH", data: { is_done: isDone } })
      .catch(revertWith(snapshot, "อัปเดตสถานะไม่สำเร็จ"));
  };

  // opts lets the Create Task modal seed assignee/priority/due/description/subtasks
  // in one shot; omitting it sends the original quick-add payload. Subtasks are titles
  // only — the backend creates them alongside the card in one transaction.
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

    // Nothing is applied optimistically: the server assigns the id, and a card
    // without one cannot be edited or dragged. The response carries it.
    apiClient<{ id: string }>("/cards", {
      method: "POST",
      data: {
        column_id: columnId,
        title,
        position: newPosition,
        ...(opts?.assigneeId ? { assignee_id: opts.assigneeId } : {}),
        ...(opts?.priority ? { priority: opts.priority } : {}),
        ...(opts?.dueDate ? { due_date: opts.dueDate } : {}),
        ...(opts?.description ? { description: opts.description } : {}),
        ...(opts?.subtasks && opts.subtasks.length > 0 ? { subtasks: opts.subtasks } : {}),
      },
    })
      .then((created) => {
        useBoardStore.getState().addCardToStore({
          id: created.id,
          column_id: columnId,
          title,
          position: newPosition,
          description: opts?.description ?? null,
          due_date: opts?.dueDate ?? null,
          assignee_id: opts?.assigneeId ?? null,
          assignee_name:
            useBoardStore
              .getState()
              .boardMembers.find((m) => m.user_id === opts?.assigneeId)?.full_name ?? null,
          priority: (opts?.priority as Card["priority"]) ?? null,
          estimated_hours: null,
          is_done: false,
          completed_at: null,
          created_at: null,
          created_by: useBoardStore.getState().currentUserId,
          total_subtasks: opts?.subtasks?.length ?? 0,
          completed_subtasks: 0,
        });
      })
      .catch((err) => {
        if (err instanceof ApiError && err.status === 403) return;
        useToastStore.getState().show({ message: "สร้างการ์ดไม่สำเร็จ", duration: 4000 });
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

    const snapshot = useBoardStore.getState().columns;
    useBoardStore.getState().moveCard(cardId, toColumnId, newPosition);
    apiClient(`/cards/${cardId}/move`, {
      method: "PATCH",
      data: { column_id: toColumnId, position: newPosition },
    }).catch(revertWith(snapshot, "ย้ายการ์ดไม่สำเร็จ"));
  };

  const handleDeleteCard = (cardId: string) => {
    const snapshot = useBoardStore.getState().columns;
    useBoardStore.getState().removeCardFromStore(cardId);
    apiClient(`/cards/${cardId}`, { method: "DELETE" }).catch(
      revertWith(snapshot, "ลบการ์ดไม่สำเร็จ"),
    );
  };

  const handleUpdateCard = (cardId: string, form: CardUpdateForm) => {
    const { boardMembers, columns, updateCard } = useBoardStore.getState();
    const newAssigneeId = form.assignee_id || null;
    const original = columns.flatMap((c) => c.cards).find((c) => c.id === cardId);
    // Only fields that actually changed become activity-log entries.
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

    // acceptance_criteria / implementation_note are sent only when they changed:
    // the backend COALESCEs them, so omitting preserves the existing value. A
    // title-only edit must not clobber AC that PromoteItem copied in.
    type CardPatchBody = {
      title: string;
      description: string | null;
      due_date: string | null;
      assignee_id: string | null;
      priority: string | null;
      estimated_hours: number | null;
      tag_ids: string[];
      changed_fields: string[];
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
      changed_fields: changedFields,
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
    // apiClient, not raw fetch: raw fetch only rejects on network errors, so a 4xx
    // used to vanish silently. Toast anything apiClient has not already toasted.
    apiClient(`/cards/${cardId}`, { method: "PATCH", data: body }).catch((err) => {
      if (err instanceof ApiError && err.status === 403) return; // apiClient already toasted
      useToastStore.getState().show({
        message: "บันทึกการ์ดไม่สำเร็จ — ลองอีกครั้ง",
        duration: 4000,
      });
    });
  };

  return { handleToggleDone, handleAddCard, handleChangeColumn, handleDeleteCard, handleUpdateCard };
}
