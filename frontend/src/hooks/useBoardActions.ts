import { useDragActions } from "./useDragActions";
import { useCardActions } from "./useCardActions";
import { useColumnActions } from "./useColumnActions";
import { useSubtaskActions } from "./useSubtaskActions";

/**
 * Facade hook that bundles every mutation a board view needs (drag-and-drop
 * handlers, card CRUD, column CRUD, subtask CRUD) under a single name.
 *
 * Use this from board page components rather than importing each hook
 * individually — it keeps the call site short and lets us add new action
 * groups without touching every consumer. Each underlying hook is
 * independently testable.
 */
export function useBoardActions(boardId: string) {
  return {
    ...useDragActions(),
    ...useCardActions(boardId),
    ...useColumnActions(),
    ...useSubtaskActions(),
  };
}
