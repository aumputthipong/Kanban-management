import { useBoardStore } from "@/store/useBoardStore";
import { apiClient, ApiError } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";
import type { Column } from "@/types/board";

// Restores the board to `snapshot` and tells the user the write did not land. 403 is
// skipped because apiClient has already toasted it.
function revertWith(snapshot: Column[], message: string) {
  return (err: unknown) => {
    useBoardStore.getState().setColumns(snapshot);
    if (err instanceof ApiError && err.status === 403) return;
    useToastStore.getState().show({ message, duration: 4000 });
  };
}

export function useColumnActions(boardId: string) {
  const handleRenameColumn = (columnId: string, title: string) => {
    const snapshot = useBoardStore.getState().columns;
    const current = snapshot.find((c) => c.id === columnId);
    if (!current) return;

    useBoardStore.getState().updateColumnInStore(columnId, { title });
    // The endpoint sets title and category outright, so a rename has to resend the
    // category it already has rather than leave it out.
    apiClient(`/columns/${columnId}`, {
      method: "PATCH",
      data: { title, category: current.category, color: current.color },
    }).catch(revertWith(snapshot, "เปลี่ยนชื่อคอลัมน์ไม่สำเร็จ"));
  };

  const handleDeleteColumn = (columnId: string) => {
    const snapshot = useBoardStore.getState().columns;
    useBoardStore.getState().removeColumnFromStore(columnId);
    apiClient(`/columns/${columnId}`, { method: "DELETE" }).catch(
      revertWith(snapshot, "ลบคอลัมน์ไม่สำเร็จ"),
    );
  };

  const handleUpdateColumn = (
    columnId: string,
    title: string,
    category: "TODO" | "DONE",
    color: string | null,
  ) => {
    const snapshot = useBoardStore.getState().columns;
    useBoardStore.getState().updateColumnInStore(columnId, { title, category, color });
    apiClient(`/columns/${columnId}`, {
      method: "PATCH",
      data: { title, category, color },
    }).catch(revertWith(snapshot, "แก้ไขคอลัมน์ไม่สำเร็จ"));
  };

  // Create is the one action with nothing to apply optimistically: the server assigns
  // the id and position, so the column is added once the response carries them.
  const handleAddColumn = async (
    title: string,
    category: "TODO" | "DONE" = "TODO",
    color: string | null = null,
  ) => {
    const trimmed = title.trim();
    if (!trimmed) return;

    try {
      const column = await apiClient<Column>(`/boards/${boardId}/columns`, {
        method: "POST",
        data: { title: trimmed, category, ...(color ? { color } : {}) },
      });
      useBoardStore.getState().addColumnToStore({ ...column, cards: [] });
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) return;
      useToastStore.getState().show({ message: "สร้างคอลัมน์ไม่สำเร็จ", duration: 4000 });
    }
  };

  return { handleRenameColumn, handleDeleteColumn, handleUpdateColumn, handleAddColumn };
}
