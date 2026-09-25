"use client";

import { useCardDetail } from "@/hooks/useCardDetail";
import { useBoardStore } from "@/store/useBoardStore";
import { TaskQuickModal } from "@/components/board/card-modal/TaskQuickModal";
import { TaskQuickDetails } from "@/components/board/card-modal/TaskQuickDetails";
import { classifyPillState, type PillState } from "./pillState";
import type { Card } from "@/types/board";

interface Props {
  card: Card;
  columnName?: string;
  onClose: () => void;
  onOpenTask: () => void;
}

const STATE_DOT: Record<PillState, string> = {
  todo: "bg-slate-400",
  inProgress: "bg-blue-700",
  done: "bg-emerald-700",
  overdue: "bg-red-600",
};

export function CalendarTaskModal({ card, columnName, onClose, onOpenTask }: Props) {
  const { detail, isLoading } = useCardDetail(card.id);
  const boardTitle = useBoardStore((s) => s.boardMeta?.title);

  return (
    <TaskQuickModal
      title={card.title}
      onClose={onClose}
      context={
        <>
          <span aria-hidden className={`w-1.25 h-1.25 rounded-full shrink-0 ${STATE_DOT[classifyPillState(card)]}`} />
          <span className="truncate">
            {boardTitle}
            {columnName && (
              <span className={boardTitle ? "text-slate-400" : undefined}>
                {boardTitle ? " · " : ""}
                {columnName}
              </span>
            )}
          </span>
        </>
      }
      footer={
        <button
          type="button"
          onClick={onOpenTask}
          className="ml-auto inline-flex items-center rounded-md bg-primary px-3.5 py-2 text-sm font-semibold text-white hover:bg-primary-hover transition-colors"
        >
          เปิด task
        </button>
      }
    >
      <TaskQuickDetails summary={card} detail={detail} loading={isLoading} />
    </TaskQuickModal>
  );
}
