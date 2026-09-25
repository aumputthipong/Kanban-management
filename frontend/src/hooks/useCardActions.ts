import { useBoardStore } from "@/store/useBoardStore";
import { POSITION_GAP } from "@/utils/boardPosition";
import { apiClient, ApiError } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";
import { buildCardFieldUpdate, type CardField } from "@/lib/cardPatch";
import type { Card, CardUpdateForm, Column } from "@/types/board";

// 403 is skipped — apiClient has already toasted it.
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
    // The DONE-column move arrives with the broadcast.
    useBoardStore.getState().updateCard({ ...card, is_done: isDone });
    apiClient(`/cards/${card.id}/done`, { method: "PATCH", data: { is_done: isDone } })
      .catch(revertWith(snapshot, "อัปเดตสถานะไม่สำเร็จ"));
  };

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

    // Not optimistic: a card has no id until the server responds.
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

  const handleUpdateCard = (cardId: string, form: CardUpdateForm, field: CardField) => {
    const { boardMembers, columns, updateCard } = useBoardStore.getState();
    const current = columns.flatMap((c) => c.cards).find((c) => c.id === cardId);
    const { body, patch } = buildCardFieldUpdate(form, field, boardMembers);
    // Patch the store's copy, not the form — other fields may be newer.
    if (current) updateCard({ ...current, ...patch });

    // apiClient, not raw fetch: raw fetch silently resolves on 4xx.
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
