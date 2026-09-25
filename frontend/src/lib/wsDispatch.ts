import { useBoardStore } from "@/store/useBoardStore";
import { useActivityStore } from "@/store/useActivityStore";
import { WS_EVENT } from "@/types/wsEvents";
import type { BoardMember } from "@/types/board";

export interface WebSocketMessage {
  type: string;
  payload: unknown;
}

// Broadcasts carry assignee_id only — resolve the name here, never default to null.
function resolveAssigneeName(assigneeId: string | null | undefined): string | null {
  if (!assigneeId) return null;
  const { boardMembers } = useBoardStore.getState();
  return boardMembers.find((m) => m.user_id === assigneeId)?.full_name ?? null;
}

export function applyWsMessage({ type, payload }: WebSocketMessage): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- shape is per-type, set by the Go emitters
  const p = payload as any;
  const board = useBoardStore.getState();

  switch (type) {
    case WS_EVENT.CardMoved:
      board.moveCard(p.card_id, p.new_column_id, p.position, p.is_done, p.completed_at);
      return;
    case WS_EVENT.CardCreated:
      board.addCardToStore({ ...p, assignee_name: resolveAssigneeName(p.assignee_id) });
      return;
    case WS_EVENT.CardDeleted:
      board.removeCardFromStore(p.card_id);
      return;
    case WS_EVENT.CardUpdated: {
      const { card_id, ...rest } = p;
      board.updateCard({ id: card_id, ...rest, assignee_name: resolveAssigneeName(rest.assignee_id) });
      return;
    }
    case WS_EVENT.CardSubtasksUpdated:
      // Full list, not a delta — safe to re-apply over our own optimistic edit.
      board.setSubtasksToCard(p.card_id, p.subtasks);
      return;
    case WS_EVENT.TagDeleted:
      board.removeTagFromBoard(p.tag_id);
      return;
    case WS_EVENT.ColumnCreated:
      board.addColumnToStore({
        id: p.id, title: p.title, position: p.position, category: p.category,
        color: p.color ?? null, cards: [],
      });
      return;
    case WS_EVENT.ColumnDeleted:
      board.removeColumnFromStore(p.column_id);
      return;
    case WS_EVENT.ColumnUpdated:
      board.updateColumnInStore(p.column_id, {
        title: p.title, category: p.category, color: p.color || null,
      });
      return;
    case WS_EVENT.BoardMembersUpdated: {
      const members: BoardMember[] = p.members;
      board.setBoardMembers(members);
      if (board.currentUserId && !members.some((m) => m.user_id === board.currentUserId)) {
        board.setRemovedFromBoard(true);
      }
      return;
    }
    case WS_EVENT.ActivityCreated:
      useActivityStore.getState().prependActivity(p);
      return;
  }
}
