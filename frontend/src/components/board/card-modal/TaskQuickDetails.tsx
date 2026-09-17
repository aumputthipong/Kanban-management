"use client";

import { Calendar, Clock } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
import { getColumnColorHex } from "@/components/board/task-board/ColumnOptionsModal";
import { formatThaiDate } from "@/utils/date_helper";
import type { Card, Tag } from "@/types/board";
import {
  Dot,
  FieldLabel,
  MetaChip,
  PRIORITY_DOT,
  PRIORITY_LABEL,
  SubtaskCheckbox,
  SubtaskProgress,
  subtaskRowClass,
} from "./modalParts";

/** Fields known before GET /cards/:id resolves. Undefined = not in the summary, wait for detail. */
export interface TaskSummary {
  due_date: string | null;
  priority: "low" | "medium" | "high" | null;
  estimated_hours: number | null;
  total_subtasks: number;
  completed_subtasks: number;
  assignee_name?: string | null;
  description?: string | null;
  tags?: Tag[];
}

interface Props {
  summary: TaskSummary;
  detail: Card | null;
  loading: boolean;
  /** Omit for a view-only modal: checkboxes render disabled. */
  onToggleSubtask?: (subtaskId: string, current: boolean) => void;
}

function Row({ label, top = false, children }: { label: string; top?: boolean; children: React.ReactNode }) {
  return (
    <div className={`flex gap-2.5 px-5 py-3 border-t border-slate-200 ${top ? "items-start" : "items-center"}`}>
      <FieldLabel className={`w-21 shrink-0 ${top ? "pt-px" : ""}`}>{label}</FieldLabel>
      <div className="min-w-0 flex-1 text-sm">{children}</div>
    </div>
  );
}

/** Body of the read-only task quick view: meta chips, labelled rows, then the subtask checklist. */
export function TaskQuickDetails({ summary, detail, loading, onToggleSubtask }: Props) {
  const assignee = detail ? detail.assignee_name : summary.assignee_name;
  const description = detail ? detail.description : summary.description;
  const tags = detail?.tags ?? summary.tags;
  const subtasks = detail?.subtasks ?? [];
  const total = subtasks.length > 0 ? subtasks.length : (detail?.total_subtasks ?? summary.total_subtasks);
  const done =
    subtasks.length > 0
      ? subtasks.filter((s) => s.is_done).length
      : (detail?.completed_subtasks ?? summary.completed_subtasks);
  const acceptance = detail?.acceptance_criteria?.split("\n").filter((l) => l.trim()) ?? [];
  const note = detail?.implementation_note?.trim();

  return (
    <>
      <div className="flex flex-wrap gap-1.5 px-5 pb-4">
        {summary.due_date && (
          <MetaChip strong>
            <Calendar size={13} className="text-slate-600" />
            {formatThaiDate(summary.due_date)}
          </MetaChip>
        )}
        {summary.priority && (
          <MetaChip strong>
            <Dot className={PRIORITY_DOT[summary.priority]} />
            {PRIORITY_LABEL[summary.priority]}
          </MetaChip>
        )}
        {summary.estimated_hours != null && summary.estimated_hours > 0 && (
          <MetaChip>
            <Clock size={13} />
            {summary.estimated_hours} ชม.
          </MetaChip>
        )}
        {tags?.map((t) => (
          <MetaChip key={t.id}>
            <span
              aria-hidden
              className="inline-block w-1.25 h-1.25 rounded-full shrink-0"
              style={{ backgroundColor: getColumnColorHex(t.color) ?? "#94a3b8" }}
            />
            {t.name}
          </MetaChip>
        ))}
      </div>

      <Row label="Assignee">
        {loading && assignee === undefined ? (
          <Skeleton className="h-4.5 w-28" />
        ) : assignee ? (
          <span className="inline-flex items-center gap-2 min-w-0">
            <span className="w-4.5 h-4.5 rounded-full grid place-items-center bg-surface-tint text-primary text-[10px] font-semibold shrink-0">
              {(Array.from(assignee.trim())[0] ?? "?").toUpperCase()}
            </span>
            <span className="truncate text-slate-900">{assignee}</span>
          </span>
        ) : (
          <span className="text-slate-600">ยังไม่มีผู้รับผิดชอบ</span>
        )}
      </Row>

      <Row label="Description" top>
        {loading && description === undefined ? (
          <div className="space-y-1.5 pt-0.5">
            <Skeleton className="h-3 w-full" />
            <Skeleton className="h-3 w-2/3" />
          </div>
        ) : description ? (
          <p className="text-slate-600 whitespace-pre-wrap leading-relaxed">{description}</p>
        ) : (
          <span className="text-slate-600">ไม่มีคำอธิบาย</span>
        )}
      </Row>

      {acceptance.length > 0 && (
        <Row label="Acceptance" top>
          <ul className="flex flex-col gap-1 text-slate-900">
            {acceptance.map((line, i) => (
              <li key={i} className="flex items-start gap-2">
                <Dot className="mt-2 bg-slate-400" />
                <span className="whitespace-pre-wrap">{line}</span>
              </li>
            ))}
          </ul>
        </Row>
      )}

      {note && (
        <Row label="Dev Note" top>
          <p className="text-slate-600 whitespace-pre-wrap leading-relaxed">{note}</p>
        </Row>
      )}

      {total > 0 && (
        <div className="px-5 pt-3.5 pb-4 border-t border-slate-200">
          <SubtaskProgress done={done} total={total} />
          {loading ? (
            <div className="flex flex-col gap-1">
              {Array.from({ length: Math.min(total, 4) }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full rounded-md" />
              ))}
            </div>
          ) : (
            <ul className="flex flex-col gap-1">
              {subtasks.map((st) => (
                <li key={st.id} className={subtaskRowClass(st.is_done)}>
                  <SubtaskCheckbox
                    checked={st.is_done}
                    disabled={!onToggleSubtask}
                    label={`${st.is_done ? "ยกเลิก" : "ทำ"}เครื่องหมายเสร็จ: ${st.title}`}
                    onToggle={() => onToggleSubtask?.(st.id, st.is_done)}
                  />
                  {onToggleSubtask ? (
                    <button
                      type="button"
                      onClick={() => onToggleSubtask(st.id, st.is_done)}
                      className="flex-1 min-w-0 text-left"
                    >
                      {st.title}
                    </button>
                  ) : (
                    <span className="flex-1 min-w-0">{st.title}</span>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </>
  );
}
