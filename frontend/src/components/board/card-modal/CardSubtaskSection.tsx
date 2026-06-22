"use client";

import { memo, useState } from "react";
import { CheckSquare, Plus } from "lucide-react";
import type { Subtask } from "@/types/board";
import { SubtaskItem } from "../task-board/subtask/SubtaskItem";
import { useBoardActions } from "@/hooks/useBoardActions";

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
  const total = list.length;
  const completed = list.filter((st) => st.is_done).length;
  const progressPercent = total === 0 ? 0 : Math.round((completed / total) * 100);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const title = newSubtaskTitle.trim();
    if (!title || !onAddSubtask) return;
    onAddSubtask(cardId, title);
    setNewSubtaskTitle("");
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-[10px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
          <CheckSquare size={12} className="text-blue-500" />
          Subtasks
        </h3>
        {total > 0 && (
          <span className="text-xs font-medium text-slate-500 bg-slate-100 px-2 py-0.5 rounded-full">
            {progressPercent}%
          </span>
        )}
      </div>

      {total > 0 && (
        <div className="w-full bg-slate-100 rounded-full h-1.5 mb-3 overflow-hidden">
          {/* No transition. Animating width triggers layout on every frame
              and the 500ms duration made each tick feel lagged — by the time
              the bar caught up the user had ticked the next box. Optimistic
              state already updates instantly; let the bar match. */}
          <div
            className="bg-blue-500 h-1.5 rounded-full"
            style={{ width: `${progressPercent}%` }}
          />
        </div>
      )}

      <div className="flex flex-col gap-2 mb-3">
        {list.map((st) => (
          <SubtaskItem
            key={st.id}
            cardId={cardId}
            subtask={st}
            onToggle={handleToggleSubtask}
            onUpdateTitle={handleUpdateSubtaskTitle}
            onDelete={handleDeleteSubtask}
          />
        ))}
      </div>

      {canEdit && (
        <form onSubmit={handleSubmit} className="flex items-center gap-2">
          <div className="flex-1 relative">
            <input
              type="text"
              value={newSubtaskTitle}
              onChange={(e) => setNewSubtaskTitle(e.target.value)}
              placeholder="Add a new subtask..."
              className="w-full text-sm bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:bg-white focus:border-blue-400 focus:ring-2 focus:ring-blue-50 transition-all"
            />
          </div>
          <button
            type="submit"
            disabled={!newSubtaskTitle.trim()}
            className="p-2 bg-blue-50 text-primary rounded-lg hover:bg-blue-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            <Plus size={16} />
          </button>
        </form>
      )}
    </div>
  );
}

export const CardSubtaskSection = memo(CardSubtaskSectionImpl);
