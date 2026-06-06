// components/kanban/card-modal/CardFormFields.tsx
"use client";

import { useState } from "react";
import { Calendar, Clock, User, Tag } from "lucide-react";
import type { BoardMember, Tag as TagType } from "@/types/board";
import { FormState } from "./CardDetailModal";
import { TagSelector } from "./TagSelector";
import { formatThaiDate } from "@/utils/date_helper";
import { QUICK_DATE_OPTIONS, QUICK_HOURS_OPTIONS } from "@/utils/quickSelect";
import { getAvatarColor } from "@/utils/avatar";

interface CardFormFieldsProps {
  form: FormState;
  members: BoardMember[];
  assigneeName?: string;
  boardId: string;
  onChange: (field: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => void;
  onTagsChange: (tags: TagType[]) => void;
  onCommit: (field: keyof FormState) => void;
  error: string | null;
  canEdit: boolean;
}

export function CardFormFields({ form, members, assigneeName, boardId, onChange, onTagsChange, onCommit, error, canEdit }: CardFormFieldsProps) {
  const [showQuickDates, setShowQuickDates] = useState(false);
  const [showQuickHours, setShowQuickHours] = useState(false);

  return (
    // PropRow rhythm: each group separated by a hairline divider (design A right rail)
    <div className="flex flex-col [&>div]:pt-4 [&>div]:mt-4 [&>div]:border-t [&>div]:border-slate-100 [&>div:first-child]:pt-0 [&>div:first-child]:mt-0 [&>div:first-child]:border-t-0">
      {/* Assignee */}
      <div>
        <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1 mb-2">
          <User size={10} /> Assignee
        </label>
        {canEdit ? (
          <select
            value={form.assignee_id}
            onChange={(e) => {
              onChange("assignee_id")(e);
              onCommit("assignee_id");
            }}
            className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 bg-white"
          >
            <option value="">Unassigned</option>
            {members.map((m) => (
              <option key={m.user_id} value={m.user_id}>
                {m.full_name} ({m.email})
              </option>
            ))}
          </select>
        ) : (
          <div className="flex items-center gap-2 px-1">
            {assigneeName ? (
              <>
                <div className={`w-6 h-6 rounded-full flex items-center justify-center text-white text-[10px] font-bold shrink-0 ${getAvatarColor(form.assignee_id)}`}>
                  {assigneeName.charAt(0).toUpperCase()}
                </div>
                <span className="text-sm text-slate-700">{assigneeName}</span>
              </>
            ) : (
              <span className="text-sm text-slate-400 italic">Unassigned</span>
            )}
          </div>
        )}
      </div>

      {/* Priority */}
      <div>
        <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider block mb-2">
          Priority
        </label>
        {canEdit ? (
          // Segmented control (design A): pill track, active = white chip with a
          // colored 3px priority bar + colored label. Priority colour stays on
          // the bar/label of the selected segment only.
          <div className="flex gap-0.5 p-0.5 bg-slate-50 border border-slate-200 rounded-md">
            {(["low", "medium", "high"] as const).map((p) => {
              const active = form.priority === p;
              const text =
                p === "high"
                  ? "text-red-600"
                  : p === "medium"
                    ? "text-amber-600"
                    : "text-emerald-600";
              const bar =
                p === "high"
                  ? "bg-red-600"
                  : p === "medium"
                    ? "bg-amber-500"
                    : "bg-emerald-500";
              return (
                <button
                  key={p}
                  type="button"
                  onClick={() => {
                    onChange("priority")({
                      target: { value: p },
                    } as React.ChangeEvent<HTMLInputElement>);
                    onCommit("priority");
                  }}
                  className={`flex-1 inline-flex items-center justify-center gap-1.5 py-1 rounded text-xs font-medium capitalize transition-colors ${
                    active ? `bg-white shadow-sm ${text}` : "text-slate-500 hover:text-slate-700"
                  }`}
                >
                  <span
                    aria-hidden
                    className={`w-[3px] h-3 rounded-sm ${active ? bar : "bg-slate-300"}`}
                  />
                  {p}
                </button>
              );
            })}
          </div>
        ) : (
          <p className="text-sm text-slate-600 px-1">
            {form.priority ? form.priority.charAt(0).toUpperCase() + form.priority.slice(1) : "—"}
          </p>
        )}
      </div>

      {/* Due Date */}
      <div>
        <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1 mb-2">
          <Calendar size={10} /> Due Date
        </label>
        {canEdit ? (
          <>
            <div className="flex items-center gap-1.5">
              <input
                type="date"
                value={form.due_date}
                onChange={(e) => {
                  onChange("due_date")(e);
                  onCommit("due_date");
                }}
                className="flex-1 min-w-0 text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 text-slate-600"
              />
              <button
                type="button"
                onClick={() => setShowQuickDates((v) => !v)}
                className="text-[10px] text-slate-400 hover:text-blue-600 px-1.5 py-1 rounded border border-slate-200 hover:border-blue-300 transition-colors shrink-0"
              >
                Quick
              </button>
            </div>
            {showQuickDates && (
              <div className="flex flex-wrap gap-1.5 mt-2">
                {QUICK_DATE_OPTIONS.map((choice) => {
                  const targetDate = new Date();
                  targetDate.setDate(targetDate.getDate() + choice.days);
                  const formattedDate = new Date(
                    targetDate.getTime() - targetDate.getTimezoneOffset() * 60000,
                  )
                    .toISOString()
                    .split("T")[0];
                  const isActive = form.due_date === formattedDate;
                  return (
                    <button
                      key={choice.label}
                      type="button"
                      onClick={() => {
                        onChange("due_date")({
                          target: { value: formattedDate },
                        } as React.ChangeEvent<HTMLInputElement>);
                        onCommit("due_date");
                      }}
                      className={`px-2.5 py-1 text-[11px] font-semibold rounded-md transition-colors border ${
                        isActive
                          ? "bg-blue-50 border-blue-200 text-blue-700"
                          : "bg-white border-slate-200 text-slate-500 hover:bg-slate-50 hover:text-slate-800"
                      }`}
                    >
                      {choice.label}
                    </button>
                  );
                })}
              </div>
            )}
          </>
        ) : (
          <p className="text-sm text-slate-600 px-1">{formatThaiDate(form.due_date) || "—"}</p>
        )}
      </div>

      {/* Estimated Hours */}
      <div>
        <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1 mb-2">
          <Clock size={10} /> Est. Hours
        </label>
        {canEdit ? (
          <>
            <div className="flex items-center gap-1.5">
              <input
                type="number"
                min="0"
                step="0.5"
                value={form.estimated_hours}
                onChange={onChange("estimated_hours")}
                onBlur={() => onCommit("estimated_hours")}
                placeholder="0"
                className="flex-1 min-w-0 text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <button
                type="button"
                onClick={() => setShowQuickHours((v) => !v)}
                className="text-[10px] text-slate-400 hover:text-blue-600 px-1.5 py-1 rounded border border-slate-200 hover:border-blue-300 transition-colors shrink-0"
              >
                Quick
              </button>
            </div>
            {showQuickHours && (
              <div className="flex flex-wrap gap-1.5 mt-2">
                {QUICK_HOURS_OPTIONS.map((choice) => {
                  const isActive = form.estimated_hours === choice.value;
                  return (
                    <button
                      key={choice.label}
                      type="button"
                      onClick={() => {
                        onChange("estimated_hours")({
                          target: { value: choice.value },
                        } as React.ChangeEvent<HTMLInputElement>);
                        onCommit("estimated_hours");
                      }}
                      className={`px-2.5 py-1 text-[11px] font-semibold rounded-md transition-colors border ${
                        isActive
                          ? "bg-blue-50 border-blue-200 text-blue-700"
                          : "bg-white border-slate-200 text-slate-500 hover:bg-slate-50 hover:text-slate-800"
                      }`}
                    >
                      {choice.label}
                    </button>
                  );
                })}
              </div>
            )}
          </>
        ) : (
          <p className="text-sm text-slate-600 px-1">
            {form.estimated_hours ? `${form.estimated_hours}h` : "—"}
          </p>
        )}
      </div>

      {/* Tags */}
      <div>
        <label className="text-[10px] font-bold text-slate-400 uppercase tracking-wider flex items-center gap-1 mb-2">
          <Tag size={10} /> Tags
        </label>
        <TagSelector
          boardId={boardId}
          selected={form.tags}
          onChange={onTagsChange}
          onCommit={() => onCommit("tags")}
          canEdit={canEdit}
        />
      </div>

      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  );
}
