// components/kanban/TaskCard.tsx
"use client";

import { memo } from "react";
import { useSortable } from "@dnd-kit/sortable";
import {
  Calendar,
  Check,
  Clock,
  UserRound,
} from "lucide-react";
import { useState } from "react";
import dynamic from "next/dynamic";
import type { Card } from "@/types/board";
import type { FormState } from "../card-modal/CardDetailModal";

const CardDetailModal = dynamic(
  () =>
    import("../card-modal/CardDetailModal").then((m) => ({
      default: m.CardDetailModal,
    })),
  { ssr: false },
);
import { useBoardActions } from "@/hooks/useBoardActions";
import { useCanEdit } from "@/hooks/useCanEdit";
import { CSS } from "@dnd-kit/utilities";
import { formatThaiDate } from "@/utils/date_helper";
import { getAvatarColor } from "@/utils/avatar";
import { PriorityBadge } from "@/components/board/task-board/PriorityBadge";
import { TagChip } from "@/components/board/task-board/TagChip";

interface CardProps {
  card: Card;
  boardId: string;
  onDeleteCard: (cardId: string) => void;
  onSaveCard: (cardId: string, form: FormState) => void;
}

export const TaskCard = memo(function TaskCard({
  card,
  boardId,
  onDeleteCard,
  onSaveCard,
}: CardProps) {
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const { handleAddSubtask } = useBoardActions(boardId);
  const canEdit = useCanEdit(card);

  const totalSubtasks = card.total_subtasks ?? 0;
  const completedSubtasks = card.completed_subtasks ?? 0;

  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: card.id,
    data: { currentColumnId: card.column_id },
  });

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <>
      <div
        ref={setNodeRef}
        style={style}
        {...listeners}
        {...attributes}
        onClick={() => setIsDetailOpen(true)}
        className={`group relative p-4 rounded-xl border flex flex-col gap-3 transition-all duration-200 select-none ${
          isDragging
            ? "opacity-0 pointer-events-none"
            : card.is_done
              ? "bg-emerald-50 border-emerald-200 shadow-sm cursor-grab hover:border-emerald-300"
              : "bg-white border-slate-200 shadow-sm cursor-grab hover:shadow-md hover:border-blue-300"
        }`}
      >
        <div className="flex items-start gap-3">
          <div className="flex flex-col gap-1.5 flex-1 min-w-0">
            {(card.priority && !card.is_done) || (card.tags && card.tags.length > 0) ? (
              <div className="flex flex-wrap items-center gap-1">
                {card.priority && !card.is_done && (
                  <PriorityBadge priority={card.priority} />
                )}
                {card.tags && card.tags.length > 0 && (
                  <>
                    {card.tags.slice(0, 3).map((tag) => (
                      <TagChip key={tag.id} tag={tag} />
                    ))}
                    {card.tags.length > 3 && (
                      <span className="text-[10px] text-slate-400 font-semibold">
                        +{card.tags.length - 3}
                      </span>
                    )}
                  </>
                )}
              </div>
            ) : null}

            {/* Done = check icon + state-done tint (design.md: never strikethrough) */}
            <div className="flex items-start gap-1.5">
              {card.is_done && (
                <span className="mt-px shrink-0 w-[18px] h-[18px] rounded-full bg-emerald-600 text-white flex items-center justify-center">
                  <Check size={11} strokeWidth={3} />
                </span>
              )}
              <p
                className={`text-sm font-semibold leading-snug transition-all line-clamp-2 ${
                  card.is_done ? "text-emerald-900" : "text-slate-700"
                }`}
              >
                {card.title}
              </p>
            </div>
          </div>
        </div>

        {totalSubtasks > 0 && (
          <div className="flex items-center gap-2 pl-2">
            <div className="flex-1 bg-slate-100 rounded-full h-1 overflow-hidden">
              <div
                className={`h-1 rounded-full transition-all duration-300 ${
                  completedSubtasks === totalSubtasks
                    ? "bg-emerald-500"
                    : "bg-blue-400"
                }`}
                style={{
                  width: `${Math.round((completedSubtasks / totalSubtasks) * 100)}%`,
                }}
              />
            </div>
            <span
              className={`text-[10px] font-semibold ${
                completedSubtasks === totalSubtasks
                  ? "text-emerald-500"
                  : "text-slate-400"
              }`}
            >
              {completedSubtasks}/{totalSubtasks}
            </span>
          </div>
        )}

        {/* Footer — due date, estimated hours, assignee */}
        <div className="flex items-center justify-between pl-2 pt-1">
          <div className="flex items-center gap-2">
            <span className="flex items-center gap-1 text-[10px] text-slate-400">
              <Calendar size={10} />
              {card.due_date ? formatThaiDate(card.due_date) : "-"}
            </span>

            <span className="flex items-center gap-1 text-[10px] text-slate-400">
              <Clock size={10} />
              {card.estimated_hours ? `${card.estimated_hours} h` : "-"}
            </span>
          </div>

          {card.assignee_name && card.assignee_id ? (
            <div
              className={`w-6 h-6 rounded-full flex items-center justify-center text-white text-[10px] font-bold shrink-0 ${getAvatarColor(card.assignee_id)}`}
              title={card.assignee_name}
            >
              {card.assignee_name.charAt(0).toUpperCase()}
            </div>
          ) : (
            <div
              className="w-6 h-6 rounded-full border-2 border-dashed border-slate-300 flex items-center justify-center shrink-0"
              title="Unassigned"
            >
              <UserRound size={12} className="text-slate-300" />
            </div>
          )}
        </div>
      </div>

      {isDetailOpen && (
        <CardDetailModal
          key={card.id}
          card={card}
          boardId={boardId}
          isOpen={isDetailOpen}
          onClose={() => setIsDetailOpen(false)}
          onUpdated={(cardId, form) => onSaveCard(cardId, form)}
          onDelete={(cardId) => {
            onDeleteCard(cardId);
            setIsDetailOpen(false);
          }}
          onAddSubtask={handleAddSubtask}
          canEdit={canEdit}
        />
      )}
    </>
  );
});
