import { useDragActions } from "./useDragActions";
import { useCardActions } from "./useCardActions";
import { useColumnActions } from "./useColumnActions";
import { useSubtaskActions } from "./useSubtaskActions";

/**
 * Facade bundling every mutation a board view needs — drag-and-drop, card, column and
 * subtask CRUD — so call sites stay short and new action groups do not touch consumers.
 */
export function useBoardActions(boardId: string) {
  return {
    ...useDragActions(),
    ...useCardActions(boardId),
    ...useColumnActions(),
    ...useSubtaskActions(),
  };
}
