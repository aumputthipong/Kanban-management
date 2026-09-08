import { DragEndEvent, DragOverEvent } from "@dnd-kit/core";
import { useRef } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import { apiClient, ApiError } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";
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

    // Snapshot before the commit, not before the dragOver preview: the preview has
    // already moved the card, and reverting to pre-drag state would fight the pointer.
    const snapshot = useBoardStore.getState().columns;
    moveCard(activeCardId, overColumnId, newPosition);

    apiClient(`/cards/${activeCardId}/move`, {
      method: "PATCH",
      data: { column_id: overColumnId, position: newPosition },
    }).catch((err) => {
      useBoardStore.getState().setColumns(snapshot);
      if (err instanceof ApiError && err.status === 403) return;
      useToastStore.getState().show({ message: "ย้ายการ์ดไม่สำเร็จ", duration: 4000 });
    });
  };

  return { handleDragStart, handleDragOver, handleDragEnd, POSITION_GAP };
}
