"use client";

import { useState } from "react";
import { createPortal } from "react-dom";
import { Calendar, Check, User, UserX, X } from "lucide-react";
import { useBoardStore } from "@/store/useBoardStore";
import { useEscapeKey } from "@/hooks/useEscapeKey";
import { getAvatarColor } from "@/utils/avatar";
import { QUICK_DATE_OPTIONS } from "@/utils/quickSelect";

interface Props {
  onClose: () => void;
  onCreate: (
    columnId: string,
    title: string,
    opts: { assigneeId: string | null; priority: string | null; dueDate: string | null },
  ) => void;
  /** Preselect a column (e.g. opened from a column's "+" button). */
  defaultColumnId?: string;
}

// Local YYYY-MM-DD for a day offset (matches CardFormFields' quick-date math).
function offsetDate(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().split("T")[0];
}

const PRIORITIES = [
  { key: "low", label: "Low", text: "text-emerald-600", bar: "bg-emerald-500" },
  { key: "medium", label: "Medium", text: "text-amber-600", bar: "bg-amber-500" },
  { key: "high", label: "High", text: "text-red-600", bar: "bg-red-600" },
] as const;

// Avatar + name pill for one assignee option. Names stay visible (no hover) so
// a small team can pick at a glance. Active = filled blue.
function AssigneeChip({
  active,
  colorId,
  initial,
  label,
  onClick,
}: {
  active: boolean;
  colorId: string;
  initial: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={label}
      className={`inline-flex items-center gap-1.5 h-8 pl-1 pr-2.5 rounded-full border text-xs font-semibold transition-colors max-w-[150px] ${
        active
          ? "bg-blue-600 text-white border-blue-600"
          : "bg-white text-slate-700 border-slate-200 hover:border-blue-400"
      }`}
    >
      <span
        className={`flex items-center justify-center w-6 h-6 rounded-full text-white text-[11px] font-bold shrink-0 ${getAvatarColor(colorId)}`}
      >
        {initial}
      </span>
      <span className="truncate">{label}</span>
    </button>
  );
}

// Lightweight "create with essential context" modal — distinct from the
// auto-save edit modal (CardDetailModal). Captures title + the 3-state assignee
// + due + priority + column, fires one CARD_CREATED, then closes. Heavier fields
// (tags/subtasks/AC/dev-note) are added later by opening the card. Kept as one
// coherent unit (> the usual split threshold is fine here — it's a single form).
export function CreateTaskModal({ onClose, onCreate, defaultColumnId }: Props) {
  const columns = useBoardStore((s) => s.columns);
  const boardMembers = useBoardStore((s) => s.boardMembers);
  const currentUserId = useBoardStore((s) => s.currentUserId);

  const todoColumns = columns.filter((c) => c.category !== "DONE");
  const others = boardMembers.filter((m) => m?.user_id && m.user_id !== currentUserId);

  const [title, setTitle] = useState("");
  const [columnId, setColumnId] = useState(
    () => defaultColumnId ?? todoColumns[0]?.id ?? "",
  );
  // Default the assignee to the creator — the common case. "ยังไม่ระบุ" (null) is
  // a deliberate click, never the silent default.
  const [assigneeId, setAssigneeId] = useState<string | null>(currentUserId);
  const [priority, setPriority] = useState<string | null>(null);
  const [dueDate, setDueDate] = useState("");

  const myInitial =
    boardMembers.find((m) => m.user_id === currentUserId)?.full_name?.charAt(0).toUpperCase() ?? "?";

  useEscapeKey(true, onClose);

  const canSubmit = title.trim().length > 0 && columnId !== "";
  const submit = () => {
    if (!canSubmit) return;
    onCreate(columnId, title.trim(), { assigneeId, priority, dueDate: dueDate || null });
    onClose();
  };

  const fieldLabel = "text-[11px] font-bold uppercase tracking-wider text-slate-400";

  return createPortal(
    <>
      <div className="fixed inset-0 z-9998 bg-slate-900/45" onClick={onClose} />
      <div className="fixed inset-0 z-9999 flex items-center justify-center pointer-events-none px-4 py-6">
        <div
          className="pointer-events-auto w-full max-w-lg bg-white rounded-2xl shadow-2xl flex flex-col overflow-hidden max-h-full"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-6 pt-5 pb-4 border-b border-slate-100">
            <h2 className="text-lg font-bold text-slate-900">New Task</h2>
            <button
              onClick={onClose}
              aria-label="ปิด"
              className="p-1 text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100 transition-colors"
            >
              <X size={18} />
            </button>
          </div>

          {/* Body */}
          <div className="px-6 py-4 flex flex-col gap-4 overflow-y-auto">
            <input
              autoFocus
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") submit();
              }}
              placeholder="Task title..."
              className="w-full text-base font-medium text-slate-900 placeholder-slate-400 border border-slate-200 rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 focus:ring-blue-400"
            />

            {/* Assignee — avatar + name for every option (small-team persona:
                read who's who at a glance, no hover). 3 states: me (default) /
                a member / explicitly unassigned. */}
            <div>
              <label className={`${fieldLabel} flex items-center gap-1 mb-2`}>
                <User size={11} /> Assignee
              </label>
              <div className="flex flex-wrap items-center gap-1.5">
                {currentUserId && (
                  <AssigneeChip
                    active={assigneeId === currentUserId}
                    colorId={currentUserId}
                    initial={myInitial}
                    label="ฉัน"
                    onClick={() => setAssigneeId(currentUserId)}
                  />
                )}
                {others.map((m) => (
                  <AssigneeChip
                    key={m.user_id}
                    active={assigneeId === m.user_id}
                    colorId={m.user_id}
                    initial={m.full_name?.charAt(0).toUpperCase() ?? "?"}
                    label={m.full_name}
                    onClick={() => setAssigneeId(m.user_id)}
                  />
                ))}
                {/* Unassigned = deliberate "needs an owner" — dashed outline sets
                    it apart from real people, not a silent default. */}
                <button
                  type="button"
                  onClick={() => setAssigneeId(null)}
                  className={`inline-flex items-center gap-1.5 h-8 pl-1.5 pr-3 rounded-full border text-xs font-semibold transition-colors ${
                    assigneeId === null
                      ? "bg-slate-800 text-white border-slate-800"
                      : "bg-white text-slate-500 border-dashed border-slate-300 hover:border-slate-400"
                  }`}
                >
                  <span
                    aria-hidden
                    className={`flex items-center justify-center w-6 h-6 rounded-full border border-dashed ${
                      assigneeId === null ? "border-white/60" : "border-slate-300"
                    }`}
                  >
                    <UserX size={12} />
                  </span>
                  ยังไม่ระบุ
                </button>
              </div>
            </div>

            {/* Due date */}
            <div>
              <label className={`${fieldLabel} flex items-center gap-1 mb-2`}>
                <Calendar size={11} /> Due Date
              </label>
              <div className="flex flex-wrap items-center gap-1.5">
                {QUICK_DATE_OPTIONS.map((c) => {
                  const value = offsetDate(c.days);
                  const active = dueDate === value;
                  return (
                    <button
                      key={c.label}
                      type="button"
                      onClick={() => setDueDate(active ? "" : value)}
                      className={`px-2.5 py-1 text-[11px] font-semibold rounded-md border transition-colors ${
                        active
                          ? "bg-blue-50 border-blue-200 text-blue-700"
                          : "bg-white border-slate-200 text-slate-500 hover:bg-slate-50"
                      }`}
                    >
                      {c.label}
                    </button>
                  );
                })}
                <input
                  type="date"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                  className="text-sm border border-slate-200 rounded-md px-2 py-1 text-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-400"
                />
              </div>
            </div>

            {/* Priority */}
            <div>
              <label className={`${fieldLabel} block mb-2`}>Priority</label>
              <div className="flex gap-0.5 p-0.5 bg-slate-50 border border-slate-200 rounded-md">
                {PRIORITIES.map((p) => {
                  const active = priority === p.key;
                  return (
                    <button
                      key={p.key}
                      type="button"
                      onClick={() => setPriority(active ? null : p.key)}
                      className={`flex-1 inline-flex items-center justify-center gap-1.5 py-1 rounded text-xs font-medium transition-colors ${
                        active ? `bg-white shadow-sm ${p.text}` : "text-slate-500 hover:text-slate-700"
                      }`}
                    >
                      <span aria-hidden className={`w-[3px] h-3 rounded-sm ${active ? p.bar : "bg-slate-300"}`} />
                      {p.label}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Column */}
            <div>
              <label className={`${fieldLabel} block mb-2`}>Column</label>
              <select
                value={columnId}
                onChange={(e) => setColumnId(e.target.value)}
                className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-400"
              >
                {todoColumns.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.title}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              className="px-3.5 py-2 text-sm font-semibold text-slate-600 rounded-lg hover:bg-slate-100 transition-colors"
            >
              ยกเลิก
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={!canSubmit}
              className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Check size={15} strokeWidth={2.5} /> สร้าง Task
            </button>
          </div>
        </div>
      </div>
    </>,
    document.body,
  );
}
