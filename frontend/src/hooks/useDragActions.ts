import { DragEndEvent, DragOverEvent } from "@dnd-kit/core";
import { useRef } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import { useBoardWebSocket } from "@/contexts/BoardWebSocketContext";
import { WS_EVENT } from "@/types/wsEvents";
import {
  POSITION_GAP,
  resolveOverFromColumns,
  calcPositionFromColumns,
} from "@/utils/boardPosition";

/**
 * @dnd-kit board drag handlers: optimistic cross-column preview on dragOver,
 * final position + CARD_MOVED broadcast on dragEnd. Reconcile contract and
 * position math: docs/ARCHITECTURE.md, "Optimistic UI pattern".
 */
export function useDragActions() {
  // Action only — selecting board state here re-renders the card modal on every mutation.
  const moveCard = useBoardStore((s) => s.moveCard);
  const { sendMessage } = useBoardWebSocket();

  // ref, not state — a re-render per pointer move would re-fire the dragOver move.
  const dragOverColumnRef = useRef<string | null>(null);

  const handleDragStart = () => {
    dragOverColumnRef.current = null;
  };

  const handleDragOver = (event: DragOverEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeCardId = active.id as string;
    const freshColumns = useBoardStore.getState().columns;

    const resolved = resolveOverFromColumns(freshColumns, over.id as string);
    if (!resolved) return;
    const { overColumnId, overCardId } = resolved;

    if (dragOverColumnRef.current === overColumnId) return;

    const currentCol = freshColumns.find((col) =>
      col.cards.some((c) => c.id === activeCardId),
    );
    if (!currentCol || currentCol.id === overColumnId) {
      dragOverColumnRef.current = overColumnId;
      return;
    }

    const tempPosition = calcPositionFromColumns(
      freshColumns,
      overColumnId,
      overCardId,
      activeCardId,
    );
    dragOverColumnRef.current = overColumnId;
    useBoardStore.getState().moveCard(activeCardId, overColumnId, tempPosition);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    dragOverColumnRef.current = null;

    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const activeCardId = active.id as string;
    const originalColumnId = active.data.current?.currentColumnId as string;

    const freshColumns = useBoardStore.getState().columns;
    const resolved = resolveOverFromColumns(freshColumns, over.id as string);
    if (!resolved) return;
    const { overColumnId, overCardId } = resolved;

    // Past the midpoint = place after; covers same-column downward moves and
    // cross-column drops onto a bottom card.
    let placeAfter = false;
    const activeTranslated = active.rect.current.translated;
    if (overCardId && over.rect && activeTranslated) {
      const overMidY = over.rect.top + over.rect.height / 2;
      const activeMidY = activeTranslated.top + activeTranslated.height / 2;
      placeAfter = activeMidY > overMidY;
    }

    const newPosition = calcPositionFromColumns(
      freshColumns,
      overColumnId,
      overCardId,
      activeCardId,
      placeAfter,
    );

    moveCard(activeCardId, overColumnId, newPosition);

    sendMessage({
      type: WS_EVENT.CardMoved,
      payload: {
        card_id: activeCardId,
        old_column_id: originalColumnId,
        new_column_id: overColumnId,
        position: newPosition,
      },
    });
  };

  return { handleDragStart, handleDragOver, handleDragEnd, POSITION_GAP };
}
