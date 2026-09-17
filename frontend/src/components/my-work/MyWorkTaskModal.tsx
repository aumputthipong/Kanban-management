"use client";

import Link from "next/link";
import { Check } from "lucide-react";
import { apiClient } from "@/lib/apiClient";
import { useCardDetail } from "@/hooks/useCardDetail";
import { boardColor } from "@/lib/boardAppearance";
import { relativeDueDate } from "@/lib/myWorkApi";
import { TaskQuickModal } from "@/components/board/card-modal/TaskQuickModal";
import { TaskQuickDetails } from "@/components/board/card-modal/TaskQuickDetails";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard } from "@/types/myWork";

interface Props {
  card: MyWorkCard;
  boardMeta: Map<string, BoardMeta>;
  onClose: () => void;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string, label: string) => void;
}

// Quick view for a My Work task. Full editing stays on the board (open-in-board),
// so this modal needs none of the board's WS/store wiring.
export function MyWorkTaskModal({ card, boardMeta, onClose, onComplete, onSnooze }: Props) {
  const { detail, setDetail, isLoading } = useCardDetail(card.id);
  const meta = boardMeta.get(card.board_id);

  const snooze = (offset: number, label: string) => {
    onSnooze(card.id, relativeDueDate(offset), label);
    onClose();
  };

  // Self-contained REST toggle (no board store/WS). Optimistic; reverts on failure.
  const toggleSubtask = (subtaskId: string, current: boolean) => {
    const flip = (value: boolean) =>
      setDetail((prev) =>
        prev?.subtasks
          ? {
              ...prev,
              subtasks: prev.subtasks.map((s) => (s.id === subtaskId ? { ...s, is_done: value } : s)),
            }
          : prev,
      );
    flip(!current);
    apiClient(`/cards/${card.id}/subtasks/${subtaskId}`, {
      method: "PATCH",
      data: { is_done: !current },
    }).catch(() => flip(current));
  };

  return (
    <TaskQuickModal
      title={card.title}
      onClose={onClose}
      context={
        <>
          <span
            aria-hidden
            className="w-1.25 h-1.25 rounded-full shrink-0"
            style={{ background: boardColor(meta?.color) }}
          />
          <span className="truncate">
            {card.board_name}
            {card.column_name && <span className="text-slate-400"> · {card.column_name}</span>}
          </span>
        </>
      }
      footer={
        <>
          <button
            type="button"
            onClick={() => {
              onComplete(card.id);
              onClose();
            }}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3.5 py-2 text-sm font-semibold text-white hover:bg-primary-hover transition-colors"
          >
            <Check size={14} strokeWidth={2.6} /> ทำเสร็จ
          </button>
          <button
            type="button"
            onClick={() => snooze(1, "พรุ่งนี้")}
            className="rounded-md px-3.5 py-2 text-sm font-medium text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition-colors"
          >
            พรุ่งนี้
          </button>
          <button
            type="button"
            onClick={() => snooze(7, "สัปดาห์หน้า")}
            className="rounded-md px-3.5 py-2 text-sm font-medium text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition-colors"
          >
            สัปดาห์หน้า
          </button>
          <Link
            href={`/board/${card.board_id}/tasks?card=${card.id}`}
            className="ml-auto text-sm font-medium text-slate-600 hover:text-slate-900 transition-colors"
          >
            เปิดใน Board →
          </Link>
        </>
      }
    >
      <TaskQuickDetails summary={card} detail={detail} loading={isLoading} onToggleSubtask={toggleSubtask} />
    </TaskQuickModal>
  );
}
