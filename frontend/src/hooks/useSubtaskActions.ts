import { useCallback } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import { useToastStore } from "@/store/useToastStore";
import { API_URL } from "@/lib/constants";
import type { Subtask } from "@/types/board";

export function useSubtaskActions() {
  // Stable identity: consumers put this in effect deps, and a fresh function per
  // render would refetch on every one. It closes over nothing from this scope.
  const fetchSubtasks = useCallback(async (cardId: string) => {
    try {
      const response = await fetch(`${API_URL}/cards/${cardId}/subtasks`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });
      if (!response.ok) throw new Error("Failed to fetch subtasks");
      const data: Subtask[] = await response.json();
      useBoardStore.getState().setSubtasksToCard(cardId, data);
    } catch (error) {
      console.error("Error fetching subtasks:", error);
    }
  }, []);

  const handleAddSubtask = async (cardId: string, title: string) => {
    try {
      const { columns, updateCard } = useBoardStore.getState();
      const targetCard = columns.flatMap((c) => c.cards).find((c) => c.id === cardId);
      if (!targetCard) return;

      const currentSubtasks = targetCard.subtasks || [];
      const newPosition = currentSubtasks.length + 1;

      const response = await fetch(`${API_URL}/cards/${cardId}/subtasks`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ title, position: newPosition }),
      });

      if (!response.ok) throw new Error("Failed to create subtask");

      const newSubtask: Subtask = await response.json();

      updateCard({
        ...targetCard,
        subtasks: [...currentSubtasks, newSubtask],
        total_subtasks: targetCard.total_subtasks + 1,
      });
    } catch (error) {
      console.error("Error creating subtask:", error);
    }
  };

  const handleToggleSubtask = async (
    cardId: string,
    subtaskId: string,
    currentStatus: boolean,
  ) => {
    const newStatus = !currentStatus;
    useBoardStore.getState().updateSubtaskInCard(cardId, subtaskId, { is_done: newStatus });

    try {
      const response = await fetch(`${API_URL}/cards/${cardId}/subtasks/${subtaskId}`, {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ is_done: newStatus }),
      });
      if (!response.ok) throw new Error("Failed to update subtask");
    } catch (error) {
      console.error("Error updating subtask:", error);
      useBoardStore.getState().updateSubtaskInCard(cardId, subtaskId, { is_done: currentStatus });
    }
  };

  const handleDeleteSubtask = (cardId: string, subtaskId: string) => {
    const store = useBoardStore.getState();
    const card = store.columns
      .flatMap((c) => c.cards)
      .find((c) => c.id === cardId);
    const removed = card?.subtasks?.find((st) => st.id === subtaskId);
    if (!removed) return;

    // Optimistic remove from UI
    store.deleteSubtaskFromCard(cardId, subtaskId);

    let undone = false;
    const toastId = useToastStore.getState().show({
      message: `Subtask "${removed.title}" deleted`,
      actionLabel: "Undo",
      duration: 5000,
      onAction: () => {
        undone = true;
        // Restore: fetch fresh list from server
        fetchSubtasks(cardId);
      },
    });

    // Delay API call — gives user time to undo
    setTimeout(async () => {
      if (undone) return;
      try {
        const response = await fetch(
          `${API_URL}/cards/${cardId}/subtasks/${subtaskId}`,
          { method: "DELETE", credentials: "include" },
        );
        if (!response.ok) throw new Error("Failed to delete subtask");
      } catch (error) {
        console.error("Error deleting subtask:", error);
        useToastStore.getState().dismiss(toastId);
        fetchSubtasks(cardId);
      }
    }, 5000);
  };

  const handleUpdateSubtaskTitle = async (
    cardId: string,
    subtaskId: string,
    newTitle: string,
  ) => {
    if (!newTitle.trim()) return;
    useBoardStore.getState().updateSubtaskInCard(cardId, subtaskId, { title: newTitle.trim() });

    try {
      const response = await fetch(`${API_URL}/cards/${cardId}/subtasks/${subtaskId}`, {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ title: newTitle.trim() }),
      });
      if (!response.ok) throw new Error("Failed to update subtask title");
    } catch (error) {
      console.error("Error updating subtask title:", error);
      fetchSubtasks(cardId);
    }
  };

  return {
    fetchSubtasks,
    handleAddSubtask,
    handleToggleSubtask,
    handleDeleteSubtask,
    handleUpdateSubtaskTitle,
  };
}
