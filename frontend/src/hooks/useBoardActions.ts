import { useDragActions } from "./useDragActions";
import { useCardActions } from "./useCardActions";
import { useColumnActions } from "./useColumnActions";
import { useSubtaskActions } from "./useSubtaskActions";

export function useBoardActions(boardId: string) {
  return {
    ...useDragActions(),
    ...useCardActions(),
    ...useColumnActions(boardId),
    ...useSubtaskActions(),
  };
}
