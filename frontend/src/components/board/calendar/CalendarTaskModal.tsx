"use client";

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { ArrowRight, Calendar, Check, Clock, ListChecks, X } from "lucide-react";
import { apiClient } from "@/lib/apiClient";
import { Skeleton } from "@/components/ui/Skeleton";
import { useEscapeKey } from "@/hooks/useEscapeKey";
import { PriorityBadge } from "@/components/board/task-board/PriorityBadge";
import { TagChip } from "@/components/board/task-board/TagChip";
import { formatThaiDate } from "@/utils/date_helper";
import { getAvatarColor } from "@/utils/avatar";
import { classifyPillState, type PillState } from "./pillState";
import type { Card } from "@/types/board";

interface Props {
  card: Card;
  /** Resolved column title for the status line. */
  columnName?: string;
  onClose: () => void;
  /** Open the full (editable) card — "go to task". */
  onOpenTask: () => void;
}

const STATE_DOT: Record<PillState, string> = {
  todo: "bg-slate-400",
  inProgress: "bg-blue-700",
  done: "bg-emerald-700",
  overdue: "bg-red-600",
};

// Read-only detail view for a calendar task — same shape as the My Work modal.
// Editing stays on the board card ("open task"), the modal's single nav action.
// Basics render instantly from the board card; AC / dev note / subtask list are
// enriched via GET /cards/:id.
export function CalendarTaskModal({ card, columnName, onClose, onOpenTask }: Props) {
  const [detail, setDetail] = useState<Card | null>(null);
  const [loading, setLoading] = useState(true);

  useEscapeKey(true, onClose);

  useEffect(() => {
    let cancelled = false;
    apiClient<Card>(`/cards/${card.id}`)
      .then((c) => {
        if (!cancelled) {
          setDetail(c);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [card.id]);

  const state = classifyPillState(card);
  const total = card.total_subtasks ?? 0;
  const done = card.completed_subtasks ?? 0;

  return createPortal(
    <>
      <div className="fixed inset-0 z-9998 bg-slate-900/45" onClick={onClose} />
      <div className="fixed inset-0 z-9999 flex items-center justify-center pointer-events-none px-4 py-6">
        <div
          className="pointer-events-auto w-full max-w-lg bg-white rounded-2xl shadow-2xl flex flex-col overflow-hidden max-h-full"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="px-6 pt-5 pb-4 border-b border-slate-100">
            <div className="flex items-start justify-between gap-3">
              <h2 className="text-lg font-bold text-slate-900 leading-snug">
                {card.title}
              </h2>
              <button
                onClick={onClose}
                aria-label="ปิด"
                className="p-1 text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100 transition-colors shrink-0"
              >
                <X size={18} />
              </button>
            </div>
            {columnName && (
              <div className="mt-1.5 inline-flex items-center gap-1.5 text-xs font-medium text-slate-500">
                <span aria-hidden className={`w-1.5 h-1.5 rounded-full ${STATE_DOT[state]}`} />
                {columnName}
              </div>
            )}
          </div>

          {/* Body */}
          <div className="px-6 py-4 flex flex-col gap-4 overflow-y-auto">
            <div className="flex flex-wrap items-center gap-2">
              {card.due_date && (
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-100 text-xs font-medium text-slate-700">
                  <Calendar size={13} className="text-slate-400" />
                  {formatThaiDate(card.due_date)}
                </span>
              )}
              {card.priority && <PriorityBadge priority={card.priority} />}
              {card.estimated_hours != null && (
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-100 text-xs font-medium text-slate-700">
                  <Clock size={13} className="text-slate-400" />
                  {card.estimated_hours} ชม.
                </span>
              )}
            </div>

            {/* Assignee */}
            <div className="flex items-center gap-3">
              <span className="text-[11px] font-bold uppercase tracking-wider text-slate-400 w-20 shrink-0">
                Assignee
              </span>
              {card.assignee_name && card.assignee_id ? (
                <span className="inline-flex items-center gap-2 min-w-0">
                  <span
                    className={`w-6 h-6 rounded-full flex items-center justify-center text-white text-[10px] font-bold shrink-0 ${getAvatarColor(card.assignee_id)}`}
                  >
                    {card.assignee_name.charAt(0).toUpperCase()}
                  </span>
                  <span className="text-sm text-slate-700 truncate">
                    {card.assignee_name}
                  </span>
                </span>
              ) : (
                <span className="text-sm text-slate-400">ยังไม่มีผู้รับผิดชอบ</span>
              )}
            </div>

            {/* Tags */}
            {card.tags && card.tags.length > 0 && (
              <div className="flex items-start gap-3">
                <span className="mt-0.5 text-[11px] font-bold uppercase tracking-wider text-slate-400 w-20 shrink-0">
                  Tags
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {card.tags.map((t) => (
                    <TagChip key={t.id} tag={t} />
                  ))}
                </div>
              </div>
            )}

            <div>
              <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400 mb-1.5">
                Description
              </p>
              {card.description ? (
                <p className="text-sm text-slate-700 whitespace-pre-wrap leading-relaxed">
                  {card.description}
                </p>
              ) : (
                <p className="text-sm text-slate-400">ไม่มีคำอธิบาย</p>
              )}
            </div>

            {total > 0 && (
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400 mb-1.5 flex items-center gap-1.5">
                  <ListChecks size={12} /> Subtasks {done}/{total}
                </p>
                <div className="h-1.5 w-full rounded-full bg-slate-100 overflow-hidden">
                  <div
                    className={`h-full rounded-full ${done === total ? "bg-emerald-500" : "bg-blue-500"}`}
                    style={{ width: `${Math.round((done / total) * 100)}%` }}
                  />
                </div>
                {detail?.subtasks && detail.subtasks.length > 0 && (
                  <ul className="mt-2.5 flex flex-col gap-1.5">
                    {detail.subtasks.map((st) => (
                      <li key={st.id} className="flex items-center gap-2 text-sm">
                        <span
                          aria-hidden
                          className={`w-4 h-4 rounded flex items-center justify-center shrink-0 border ${
                            st.is_done
                              ? "bg-emerald-500 border-emerald-500 text-white"
                              : "border-slate-300 text-transparent"
                          }`}
                        >
                          <Check size={11} strokeWidth={3} />
                        </span>
                        <span className={st.is_done ? "text-slate-400" : "text-slate-700"}>
                          {st.title}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {detail?.acceptance_criteria && detail.acceptance_criteria.trim() && (
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400 mb-1.5">
                  Acceptance Criteria
                </p>
                <ul className="flex flex-col gap-1">
                  {detail.acceptance_criteria
                    .split("\n")
                    .filter((l) => l.trim())
                    .map((line, i) => (
                      <li key={i} className="flex items-start gap-2 text-sm text-slate-700">
                        <span className="mt-[7px] w-1 h-1 rounded-full bg-slate-400 shrink-0" />
                        <span className="whitespace-pre-wrap">{line}</span>
                      </li>
                    ))}
                </ul>
              </div>
            )}

            {detail?.implementation_note && detail.implementation_note.trim() && (
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400 mb-1.5">
                  Dev Note
                </p>
                <p className="text-sm text-slate-700 whitespace-pre-wrap leading-relaxed">
                  {detail.implementation_note}
                </p>
              </div>
            )}

            {loading && !detail && (
              <div className="space-y-1.5">
                <Skeleton className="h-3 w-2/3" />
              </div>
            )}
          </div>

          {/* Footer — view-only: nav to the full task */}
          <div className="px-6 py-4 border-t border-slate-100 flex items-center gap-2">
            <button
              type="button"
              onClick={onOpenTask}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-700 px-3.5 py-2 text-sm font-semibold text-white hover:bg-blue-800 transition-colors"
            >
              เปิด task
              <ArrowRight size={15} />
            </button>
          </div>
        </div>
      </div>
    </>,
    document.body,
  );
}
