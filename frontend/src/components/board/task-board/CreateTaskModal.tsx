"use client";

import { useState } from "react";
import { createPortal } from "react-dom";
import { Calendar, Check, Flag, ListChecks, Plus, User, UserX, X } from "lucide-react";
import { useBoardStore } from "@/store/useBoardStore";
import { useEscapeKey } from "@/hooks/useEscapeKey";
import { getAvatarColor } from "@/utils/avatar";
import { QUICK_DATE_OPTIONS } from "@/utils/quickSelect";

interface Props {
  onClose: () => void;
  onCreate: (
    columnId: string,
    title: string,
    opts: {
      assigneeId: string | null;
      priority: string | null;
      dueDate: string | null;
      description: string | null;
      subtasks: string[];
    },
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
  { key: "low", label: "Low", text: "text-emerald-600" },
  { key: "medium", label: "Medium", text: "text-amber-600" },
  { key: "high", label: "High", text: "text-red-600" },
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
          ? "bg-primary text-white border-primary"
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

// "Create with essential context" modal — distinct from the auto-save edit modal
// (CardDetailModal). The fast path is title + Enter; description and subtasks are
// optional content (subtasks stay collapsed so quick capture never sees them),
// and the assignee / due / priority / column meta sits in one compact zone so it
// can't crowd out the content. Fires one CARD_CREATED, then closes. Kept as one
// coherent unit (> the usual split threshold is fine here — it's a single form).
export function CreateTaskModal({ onClose, onCreate, defaultColumnId }: Props) {
  const columns = useBoardStore((s) => s.columns);
  const boardMembers = useBoardStore((s) => s.boardMembers);
  const currentUserId = useBoardStore((s) => s.currentUserId);

  const todoColumns = columns.filter((c) => c.category !== "DONE");
  const others = boardMembers.filter((m) => m?.user_id && m.user_id !== currentUserId);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [subtasks, setSubtasks] = useState<string[]>([]);
  const [columnId, setColumnId] = useState(
    () => defaultColumnId ?? todoColumns[0]?.id ?? "",
  );
  // Default the assignee to the creator — the common case. "Unassigned" (null) is
  // a deliberate click, never the silent default.
  const [assigneeId, setAssigneeId] = useState<string | null>(currentUserId);
  const [priority, setPriority] = useState<string | null>(null);
  const [dueDate, setDueDate] = useState("");

  const myInitial =
    boardMembers.find((m) => m.user_id === currentUserId)?.full_name?.charAt(0).toUpperCase() ?? "?";

  useEscapeKey(true, onClose);

  const addSubtask = () => setSubtasks((s) => [...s, ""]);
  const updateSubtask = (i: number, v: string) =>
    setSubtasks((s) => s.map((t, idx) => (idx === i ? v : t)));
  const removeSubtask = (i: number) =>
    setSubtasks((s) => s.filter((_, idx) => idx !== i));

  const canSubmit = title.trim().length > 0 && columnId !== "";
  const submit = () => {
    if (!canSubmit) return;
    onCreate(columnId, title.trim(), {
      assigneeId,
      priority,
      dueDate: dueDate || null,
      description: description.trim() || null,
      subtasks: subtasks.map((t) => t.trim()).filter(Boolean),
    });
    onClose();
  };

  const fieldLabel = "text-[11px] font-bold uppercase tracking-wider text-slate-400";
  // One compact control geometry for every text input/select/textarea so the
  // modal reads as one consistent set (radius + padding + border) — sized to the
  // small, tidy Due Date controls rather than chunkier inputs.
  const inputClass =
    "text-sm text-slate-700 placeholder-slate-400 bg-white border border-slate-200 rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-400";
  // Quick-select / toggle chips — the original compact Due Date chip size.
  const chipClass =
    "inline-flex items-center px-2.5 py-1 text-[11px] font-semibold rounded-md border transition-colors";

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
            {/* Content zone — what the task IS. */}
            <input
              autoFocus
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") submit();
              }}
              placeholder="Task title..."
              className={`w-full text-[15px] font-medium text-slate-900 placeholder-slate-400 bg-white border border-slate-200 rounded-md px-2.5 py-1.5 focus:outline-none focus:ring-2 focus:ring-blue-400`}
            />

            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="เพิ่มรายละเอียด... (ไม่บังคับ)"
              rows={2}
              className={`w-full ${inputClass} resize-none`}
            />

            {/* Subtasks — collapsed by default so quick capture never sees them. */}
            <div>
              {subtasks.length === 0 ? (
                <button
                  type="button"
                  onClick={addSubtask}
                  className="w-full inline-flex items-center justify-center gap-1.5 rounded-md border border-dashed border-slate-300 px-2.5 py-1.5 text-xs font-semibold text-slate-500 hover:border-blue-400 hover:text-primary hover:bg-blue-50/40 transition-colors"
                >
                  <Plus size={14} /> Add subtask
                </button>
              ) : (
                <div className="flex flex-col gap-1.5">
                  <span className={`${fieldLabel} flex items-center gap-1`}>
                    <ListChecks size={11} /> Subtasks
                  </span>
                  {subtasks.map((t, i) => (
                    <div key={i} className="flex items-center gap-2">
                      <span
                        aria-hidden
                        className="w-4 h-4 rounded border border-slate-300 shrink-0"
                      />
                      <input
                        autoFocus
                        value={t}
                        onChange={(e) => updateSubtask(i, e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            addSubtask();
                          }
                        }}
                        placeholder="รายการย่อย..."
                        className={`flex-1 ${inputClass}`}
                      />
                      <button
                        type="button"
                        onClick={() => removeSubtask(i)}
                        aria-label="ลบรายการ"
                        className="p-1 text-slate-400 hover:text-red-500 rounded transition-colors"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  ))}
                  <button
                    type="button"
                    onClick={addSubtask}
                    className="inline-flex items-center gap-1 text-xs font-semibold text-slate-500 hover:text-primary transition-colors w-fit"
                  >
                    <Plus size={13} /> Add item
                  </button>
                </div>
              )}
            </div>

            {/* Meta zone — assignee / due / priority / column, kept compact so it
                never crowds the content above. */}
            <div className="flex flex-col gap-3 rounded-xl border border-slate-100 bg-slate-50/60 px-3 py-3">
              {/* Assignee — avatar + name for every option (small-team persona:
                  read who's who at a glance). 3 states: me / a member / unassigned. */}
              <div>
                <label className={`${fieldLabel} flex items-center gap-1 mb-1.5`}>
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
                <label className={`${fieldLabel} flex items-center gap-1 mb-1.5`}>
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
                        className={`${chipClass} ${
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
                    className="text-[11px] text-slate-600 bg-white border border-slate-200 rounded-md px-2.5 py-1 focus:outline-none focus:ring-2 focus:ring-blue-400"
                  />
                </div>
              </div>

              {/* Priority (compact flag chips, not a full-width bar) + Column,
                  sharing one row to reclaim vertical space. */}
              <div className="flex flex-wrap items-end gap-x-5 gap-y-3">
                <div>
                  <label className={`${fieldLabel} block mb-1.5`}>Priority</label>
                  <div className="flex items-center gap-1.5">
                    {PRIORITIES.map((p) => {
                      const active = priority === p.key;
                      return (
                        <button
                          key={p.key}
                          type="button"
                          onClick={() => setPriority(active ? null : p.key)}
                          className={`${chipClass} gap-1.5 ${
                            active
                              ? `bg-white border-current ${p.text}`
                              : "bg-white border-slate-200 text-slate-500 hover:border-slate-300"
                          }`}
                        >
                          <Flag size={12} className={active ? p.text : "text-slate-300"} />
                          {p.label}
                        </button>
                      );
                    })}
                  </div>
                </div>

                <div className="min-w-[140px] flex-1">
                  <label className={`${fieldLabel} block mb-1.5`}>Column</label>
                  <select
                    value={columnId}
                    onChange={(e) => setColumnId(e.target.value)}
                    className={`w-full ${inputClass}`}
                  >
                    {todoColumns.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.title}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              className="px-3.5 py-2 text-sm font-semibold text-slate-600 rounded-lg hover:bg-slate-100 transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={submit}
              disabled={!canSubmit}
              className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Check size={15} strokeWidth={2.5} /> Create Task
            </button>
          </div>
        </div>
      </div>
    </>,
    document.body,
  );
}
