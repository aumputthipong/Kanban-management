"use client";

import { memo, useState } from "react";
import { Plus } from "lucide-react";
import type { Subtask } from "@/types/board";
import { SubtaskItem } from "../task-board/subtask/SubtaskItem";
import { useBoardActions } from "@/hooks/useBoardActions";
import { SubtaskProgress } from "./modalParts";

interface CardSubtaskSectionProps {
  cardId: string;
  boardId: string;
  subtasks: Subtask[] | undefined;
  canEdit: boolean;
  onAddSubtask?: (cardId: string, title: string) => void;
}

function CardSubtaskSectionImpl({
  cardId,
  boardId,
  subtasks,
  canEdit,
  onAddSubtask,
}: CardSubtaskSectionProps) {
  const [newSubtaskTitle, setNewSubtaskTitle] = useState("");
  const {
    handleToggleSubtask,
    handleDeleteSubtask,
    handleUpdateSubtaskTitle,
  } = useBoardActions(boardId);

  const list = subtasks ?? [];
  const completed = list.filter((st) => st.is_done).length;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const title = newSubtaskTitle.trim();
    if (!title || !onAddSubtask) return;
    onAddSubtask(cardId, title);
    setNewSubtaskTitle("");
  };

  return (
    <div>
      <SubtaskProgress done={completed} total={list.length} showPercent />

      {list.length > 0 && (
        <div className="flex flex-col gap-1">
          {list.map((st) => (
            <SubtaskItem
              key={st.id}
              cardId={cardId}
              subtask={st}
              onToggle={handleToggleSubtask}
              onUpdateTitle={handleUpdateSubtaskTitle}
              onDelete={handleDeleteSubtask}
              canEdit={canEdit}
            />
          ))}
        </div>
      )}

      {canEdit && (
        <form onSubmit={handleSubmit} className="flex items-center gap-2 mt-2">
          <input
            type="text"
            aria-label="New subtask"
            value={newSubtaskTitle}
            onChange={(e) => setNewSubtaskTitle(e.target.value)}
            placeholder="เพิ่ม subtask..."
            className="flex-1 min-w-0 px-3 py-2 text-sm text-slate-900 bg-white border border-slate-200 rounded-md focus:outline-none focus:border-primary focus:ring-3 focus:ring-surface-tint transition-colors placeholder:text-slate-600"
          />
          <button
            type="submit"
            disabled={!newSubtaskTitle.trim()}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-sm font-medium text-slate-900 bg-white border border-slate-200 rounded-md hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Plus size={13} strokeWidth={2.4} />
            Add
          </button>
        </form>
      )}
    </div>
  );
}

export const CardSubtaskSection = memo(CardSubtaskSectionImpl);
