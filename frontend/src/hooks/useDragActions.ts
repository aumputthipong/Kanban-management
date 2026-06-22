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
 * @dnd-kit handlers for the board's drag-and-drop. Three concerns:
 *
 *   1. **Cross-column preview** during drag — moves the card optimistically in
 *      the store on `dragOver` so the user sees it land. A `useRef` tracks the
 *      last column we moved into so we don't keep re-firing the move on every
 *      pointer event (would loop with React's update cycle).
 *
 *   2. **Final commit** on `dragEnd` — recomputes the final position from the
 *      midpoint of the over-card to decide before/after placement, applies it
 *      locally, then broadcasts `CARD_MOVED` over WS. The server is the source
 *      of truth: if it rejects the move it broadcasts the corrected state and
 *      everyone (including this client) reconciles via `moveCard`'s
 *      idempotent semantics.
 *
 *   3. **Position math** is delegated to `utils/boardPosition` which uses the
 *      64k-gap strategy described in `docs/DATABASE.md` so reorders stay cheap.
 */
export function useDragActions() {
  // Select the action only — Zustand actions are stable, so this avoids
  // subscribing to board state. Consumers (useBoardActions) would otherwise
  // re-render on EVERY store mutation (subtask toggle, WS activity, drag),
  // which re-rendered the whole card modal on each interaction.
  const moveCard = useBoardStore((s) => s.moveCard);
  const { sendMessage } = useBoardWebSocket();

  // Tracks the column we most recently moved the card into during a drag.
  // ref (not state) — we don't want the React update cycle on every pointer move.
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

    // ref guard: ถ้าเราย้ายไป column นี้แล้ว ข้ามได้เลย (ป้องกัน loop)
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

    // Determine whether to place BEFORE or AFTER the over card.
    // Compare translated center of the dragged card vs center of the over card —
    // if the drag is past the midpoint, treat it as "after". Handles both
    // same-column downward moves and cross-column drops onto a bottom card.
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

  // Re-export POSITION_GAP so useCardActions can use it without re-importing
  return { handleDragStart, handleDragOver, handleDragEnd, POSITION_GAP };
}
