import { useBoardStore } from "@/store/useBoardStore";
import { useBoardWebSocket } from "@/contexts/BoardWebSocketContext";

export function useColumnActions() {
  const { sendMessage } = useBoardWebSocket();

  const handleRenameColumn = (columnId: string, title: string) => {
    useBoardStore.getState().renameColumnInStore(columnId, title);
    sendMessage({
      type: "COLUMN_RENAMED",
      payload: { column_id: columnId, title },
    });
  };

  const handleDeleteColumn = (columnId: string) => {
    useBoardStore.getState().removeColumnFromStore(columnId);
    sendMessage({
      type: "COLUMN_DELETED",
      payload: { column_id: columnId },
    });
  };

  const handleUpdateColumn = (
    columnId: string,
    title: string,
    category: "TODO" | "DONE",
    color: string | null,
  ) => {
    useBoardStore.getState().updateColumnInStore(columnId, { title, category, color });
    sendMessage({
      type: "COLUMN_UPDATED",
      payload: { column_id: columnId, title, category, color: color ?? "" },
    });
  };

  const handleAddColumn = (
    title: string,
    category: "TODO" | "DONE" = "TODO",
    color: string | null = null,
  ) => {
    if (!title.trim()) return;
    sendMessage({
      type: "COLUMN_CREATED",
      payload: {
        title: title.trim(),
        category,
        ...(color ? { color } : {}),
      },
    });
  };

  return { handleRenameColumn, handleDeleteColumn, handleUpdateColumn, handleAddColumn };
}
