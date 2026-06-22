"use client";

import { useMemo, useState, useRef, useEffect } from "react";
import { ChevronDown, X } from "lucide-react";
import { useBoardStore } from "@/store/useBoardStore";
import type { PillState } from "./pillState";

export type StatusFilter = PillState | null;
type StatusMulti = PillState[];

interface Props {
  statusFilter: StatusMulti;
  onToggleStatus: (s: PillState) => void;
  onClearStatus: () => void;
  myTasksOnly: boolean;
  onToggleMyTasks: () => void;
}

const PRIORITY_OPTIONS = ["high", "medium", "low"] as const;

const STATUS_OPTIONS: { key: PillState; label: string }[] = [
  { key: "todo", label: "To do" },
  { key: "inProgress", label: "In progress" },
  { key: "done", label: "Done" },
  { key: "overdue", label: "Overdue" },
];

/**
 * Filter chip row above the calendar grid.
 *
 * Status + "My tasks" are local to this view (state lives in ProjectCalendar).
 * Priority / Tag / Assignee piggy-back on the existing board store filters so
 * the chips stay in sync if a user switches between Board and Calendar.
 */
export function CalendarFilters({
  statusFilter,
  onToggleStatus,
  onClearStatus,
  myTasksOnly,
  onToggleMyTasks,
}: Props) {
  const filterPriorities = useBoardStore((s) => s.filterPriorities);
  const toggleFilterPriority = useBoardStore((s) => s.toggleFilterPriority);
  const clearFilterPriorities = useBoardStore((s) => s.clearFilterPriorities);
  const filterTagIds = useBoardStore((s) => s.filterTagIds);
  const toggleFilterTag = useBoardStore((s) => s.toggleFilterTag);
  const clearFilterTags = useBoardStore((s) => s.clearFilterTags);
  const filterAssigneeId = useBoardStore((s) => s.filterAssigneeId);
  const setFilterAssigneeId = useBoardStore((s) => s.setFilterAssigneeId);
  const columns = useBoardStore((s) => s.columns);
  const boardMembers = useBoardStore((s) => s.boardMembers);

  const allTags = useMemo(() => {
    const seen = new Map<string, { id: string; name: string; color: string }>();
    columns.forEach((col) =>
      col.cards.forEach((c) =>
        c.tags?.forEach((t) => {
          if (!seen.has(t.id)) seen.set(t.id, t);
        }),
      ),
    );
    return Array.from(seen.values());
  }, [columns]);

  return (
    <div className="mb-3 flex flex-wrap items-center gap-2">
      <ChipToggle
        label="My tasks"
        active={myTasksOnly}
        onClick={onToggleMyTasks}
      />

      <ChipDropdown
        label="Priority"
        activeCount={filterPriorities.length}
        onClear={clearFilterPriorities}
      >
        {PRIORITY_OPTIONS.map((p) => (
          <DropdownItem
            key={p}
            label={p}
            active={filterPriorities.includes(p)}
            onClick={() => toggleFilterPriority(p)}
          />
        ))}
      </ChipDropdown>

      <ChipDropdown
        label="Status"
        activeCount={statusFilter.length}
        onClear={onClearStatus}
      >
        {STATUS_OPTIONS.map((s) => (
          <DropdownItem
            key={s.key}
            label={s.label}
            active={statusFilter.includes(s.key)}
            onClick={() => onToggleStatus(s.key)}
          />
        ))}
      </ChipDropdown>

      <ChipDropdown
        label="Tag"
        activeCount={filterTagIds.length}
        onClear={clearFilterTags}
      >
        {allTags.length === 0 ? (
          <div className="px-3 py-2 text-xs text-slate-400">No tags</div>
        ) : (
          allTags.map((t) => (
            <DropdownItem
              key={t.id}
              label={t.name}
              active={filterTagIds.includes(t.id)}
              onClick={() => toggleFilterTag(t.id)}
            />
          ))
        )}
      </ChipDropdown>

      <ChipDropdown
        label="Assignee"
        activeCount={filterAssigneeId ? 1 : 0}
        onClear={() => setFilterAssigneeId(null)}
      >
        {boardMembers.length === 0 ? (
          <div className="px-3 py-2 text-xs text-slate-400">No members</div>
        ) : (
          boardMembers.map((m) => (
            <DropdownItem
              key={m.user_id}
              label={m.full_name || m.email}
              active={filterAssigneeId === m.user_id}
              onClick={() =>
                setFilterAssigneeId(filterAssigneeId === m.user_id ? null : m.user_id)
              }
            />
          ))
        )}
      </ChipDropdown>
    </div>
  );
}

function ChipToggle({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
        active
          ? "border-indigo-300 bg-indigo-50 text-indigo-800"
          : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
      }`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${active ? "bg-primary" : "bg-slate-300"}`}
      />
      {label}
    </button>
  );
}

function ChipDropdown({
  label,
  activeCount,
  onClear,
  children,
}: {
  label: string;
  activeCount: number;
  onClear: () => void;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  const active = activeCount > 0;
  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors ${
          active
            ? "border-indigo-300 bg-indigo-50 text-indigo-800"
            : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
        }`}
      >
        {label}
        {active && (
          <span className="rounded-full bg-primary px-1.5 text-[10px] font-bold text-white">
            {activeCount}
          </span>
        )}
        {active ? (
          <X
            size={12}
            onClick={(e) => {
              e.stopPropagation();
              onClear();
            }}
            className="text-primary hover:text-primary-hover"
          />
        ) : (
          <ChevronDown size={12} className="text-slate-400" />
        )}
      </button>
      {open && (
        <div className="absolute left-0 top-full z-30 mt-1 min-w-[180px] rounded-md border border-slate-200 bg-white py-1 shadow-lg">
          {children}
        </div>
      )}
    </div>
  );
}

function DropdownItem({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-center justify-between px-3 py-1.5 text-left text-xs transition-colors hover:bg-slate-50 ${active ? "font-semibold text-indigo-800" : "text-slate-700"}`}
    >
      <span className="capitalize">{label}</span>
      {active && <span className="text-primary">✓</span>}
    </button>
  );
}
