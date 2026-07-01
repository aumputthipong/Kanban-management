"use client";

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import Link from "next/link";
import { ArrowRight, Calendar, Check, Clock, ListChecks, X } from "lucide-react";
import { apiClient } from "@/lib/apiClient";
import { useEscapeKey } from "@/hooks/useEscapeKey";
import { BoardGlyph, boardColor } from "@/lib/boardAppearance";
import { PriorityBadge } from "@/components/board/task-board/PriorityBadge";
import { TagChip } from "@/components/board/task-board/TagChip";
import { formatThaiDate } from "@/utils/date_helper";
import { getAvatarColor } from "@/utils/avatar";
import { relativeDueDate } from "@/lib/myWorkApi";
import type { Card } from "@/types/board";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard, MyWorkStatus } from "@/types/myWork";

interface Props {
  card: MyWorkCard;
  boardMeta: Map<string, BoardMeta>;
  onClose: () => void;
  onComplete: (cardId: string) => void;
  onSnooze: (cardId: string, dueDate: string, label: string) => void;
}

function statusDot(status: MyWorkStatus): string {
  if (status === "in_progress") return "bg-blue-700";
  if (status === "done") return "bg-emerald-700";
  return "bg-slate-400";
}

// Read-focused detail view for a My Work task — opens in place of navigating to
// the board. Basics render instantly from the row's data; description /
// subtasks / tags are enriched via GET /cards/:id. Full editing stays on the
// board (open-in-board) so this modal needs none of the board's WS/store wiring.
export function MyWorkTaskModal({ card, boardMeta, onClose, onComplete, onSnooze }: Props) {
  const [detail, setDetail] = useState<Card | null>(null);
  const [loading, setLoading] = useState(true);
  const meta = boardMeta.get(card.board_id);

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

  const snooze = (offset: number, label: string) => {
    onSnooze(card.id, relativeDueDate(offset), label);
    onClose();
  };

  const subtasks = detail?.subtasks ?? [];
  const total = subtasks.length > 0 ? subtasks.length : (detail?.total_subtasks ?? 0);
  const done =
    subtasks.length > 0
      ? subtasks.filter((s) => s.is_done).length
      : (detail?.completed_subtasks ?? 0);

  // Toggle a subtask done/undone via REST — self-contained (no board store/WS),
  // same pattern as snooze. Optimistic; reverts on failure.
  const toggleSubtask = (subtaskId: string, current: boolean) => {
    const flip = (value: boolean) =>
      setDetail((prev) =>
        prev?.subtasks
          ? {
              ...prev,
              subtasks: prev.subtasks.map((s) =>
                s.id === subtaskId ? { ...s, is_done: value } : s,
              ),
            }
          : prev,
      );
    flip(!current);
    apiClient(`/cards/${card.id}/subtasks/${subtaskId}`, {
      method: "PATCH",
      data: { is_done: !current },
    }).catch(() => flip(current));
  };

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
            <div className="flex items-center justify-between gap-3">
              <span className="inline-flex items-center gap-1.5 min-w-0">
                <span
                  aria-hidden
                  className="w-5 h-5 rounded flex items-center justify-center text-white shrink-0"
                  style={{ background: boardColor(meta?.color) }}
                >
                  <BoardGlyph icon={meta?.icon} size={12} />
                </span>
                <span className="text-xs font-semibold text-slate-500 truncate">
                  {card.board_name}
                </span>
              </span>
              <button
                onClick={onClose}
                aria-label="ปิด"
                className="p-1 text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100 transition-colors shrink-0"
              >
                <X size={18} />
              </button>
            </div>
            <h2 className="mt-2 text-lg font-bold text-slate-900 leading-snug">
              {card.title}
            </h2>
            {card.column_name && (
              <div className="mt-1.5 inline-flex items-center gap-1.5 text-xs font-medium text-slate-500">
                <span aria-hidden className={`w-1.5 h-1.5 rounded-full ${statusDot(card.status)}`} />
                {card.column_name}
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
              {loading ? (
                <span className="h-6 w-28 rounded bg-slate-100 animate-pulse" />
              ) : detail?.assignee_name && detail.assignee_id ? (
                <span className="inline-flex items-center gap-2 min-w-0">
                  <span
                    className={`w-6 h-6 rounded-full flex items-center justify-center text-white text-[10px] font-bold shrink-0 ${getAvatarColor(detail.assignee_id)}`}
                  >
                    {detail.assignee_name.charAt(0).toUpperCase()}
                  </span>
                  <span className="text-sm text-slate-700 truncate">
                    {detail.assignee_name}
                  </span>
                </span>
              ) : (
                <span className="text-sm text-slate-400">ยังไม่มีผู้รับผิดชอบ</span>
              )}
            </div>

            {/* Tags */}
            {detail?.tags && detail.tags.length > 0 && (
              <div className="flex items-start gap-3">
                <span className="mt-0.5 text-[11px] font-bold uppercase tracking-wider text-slate-400 w-20 shrink-0">
                  Tags
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {detail.tags.map((t) => (
                    <TagChip key={t.id} tag={t} />
                  ))}
                </div>
              </div>
            )}

            <div>
              <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400 mb-1.5">
                Description
              </p>
              {loading ? (
                <div className="space-y-1.5">
                  <div className="h-3 w-full rounded bg-slate-100 animate-pulse" />
                  <div className="h-3 w-2/3 rounded bg-slate-100 animate-pulse" />
                </div>
              ) : detail?.description ? (
                <p className="text-sm text-slate-700 whitespace-pre-wrap leading-relaxed">
                  {detail.description}
                </p>
              ) : (
                <p className="text-sm text-slate-400">ไม่มีคำอธิบาย</p>
              )}
            </div>

            {/* Acceptance criteria (dev note) — one bullet per non-empty line */}
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

            {/* Implementation note (dev note) */}
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
                {subtasks.length > 0 && (
                  <ul className="mt-2.5 flex flex-col gap-1.5">
                    {subtasks.map((st) => (
                      <li key={st.id} className="flex items-center gap-2 text-sm">
                        <button
                          type="button"
                          onClick={() => toggleSubtask(st.id, st.is_done)}
                          aria-pressed={st.is_done}
                          aria-label={`${st.is_done ? "ยกเลิก" : "ทำ"}เครื่องหมายเสร็จ: ${st.title}`}
                          className={`w-4 h-4 rounded flex items-center justify-center shrink-0 border transition-colors ${
                            st.is_done
                              ? "bg-emerald-500 border-emerald-500 text-white hover:bg-emerald-600"
                              : "border-slate-300 text-transparent hover:border-emerald-400 hover:text-emerald-400"
                          }`}
                        >
                          <Check size={11} strokeWidth={3} />
                        </button>
                        <button
                          type="button"
                          onClick={() => toggleSubtask(st.id, st.is_done)}
                          className={`text-left transition-colors ${
                            st.is_done
                              ? "text-slate-400"
                              : "text-slate-700 hover:text-slate-900"
                          }`}
                        >
                          {st.title}
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </div>

          {/* Footer actions */}
          <div className="px-6 py-4 border-t border-slate-100 flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => {
                onComplete(card.id);
                onClose();
              }}
              className="inline-flex items-center gap-1.5 rounded-lg bg-emerald-600 px-3.5 py-2 text-sm font-semibold text-white hover:bg-emerald-700 transition-colors"
            >
              <Check size={15} strokeWidth={2.5} /> ทำเสร็จ
            </button>
            <button
              type="button"
              onClick={() => snooze(1, "พรุ่งนี้")}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50 transition-colors"
            >
              <Clock size={14} /> พรุ่งนี้
            </button>
            <button
              type="button"
              onClick={() => snooze(7, "สัปดาห์หน้า")}
              className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-50 transition-colors"
            >
              สัปดาห์หน้า
            </button>
            <Link
              href={`/board/${card.board_id}/tasks?card=${card.id}`}
              className="ml-auto inline-flex items-center gap-1.5 text-sm font-semibold text-blue-700 hover:gap-2 transition-all"
            >
              เปิดใน Board
              <ArrowRight size={15} />
            </Link>
          </div>
        </div>
      </div>
    </>,
    document.body,
  );
}
