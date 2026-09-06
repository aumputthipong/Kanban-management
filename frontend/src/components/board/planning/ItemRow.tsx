"use client";

import {
  KeyboardEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  ArrowRight,
  CheckCircle2,
  CheckSquare,
  Circle,
  Clock,
  FileText,
  MessageSquare,
  Pencil,
  Square,
} from "lucide-react";
import { useBoardStore } from "@/store/useBoardStore";
import type {
  PlanningItem,
  PlanningItemStatus,
  PlanningItemType,
} from "@/types/planning";
import { CommentThread } from "./CommentThread";
import { ItemActionMenu } from "./ItemActionMenu";
import { ItemDetailsPanel } from "./ItemDetailsPanel";
import { ItemTypeChip } from "./ItemTypeChip";
import { usePlanningComments } from "@/hooks/usePlanningComments";

interface ItemRowProps {
  index: number;
  item: PlanningItem;
  focused: boolean;
  /** Rows show a leading checkbox and a click toggles selection instead of editing. */
  selectMode: boolean;
  isSelected: boolean;
  onToggleSelect: () => void;
  onFocus: () => void;
  onChangeType: (t: PlanningItemType) => void;
  onChangeTitle: (t: string) => void;
  onToggleStatus: (s: PlanningItemStatus) => void;
  onPromote: () => void;
  onDelete: () => void;
  onChangeNote: (value: string) => void;
  onUp: () => void;
  onDown: () => void;
}

// One row of the capture surface. The type chip opens a 3-option popover: the old
// cycle-on-click was discoverable only by accident and made REQ to DEC a two-click
// trip, so do not go back to it. Keyboard surface is limited to the edit input.
export function ItemRow({
  index,
  item,
  focused,
  selectMode,
  isSelected,
  onToggleSelect,
  onFocus,
  onChangeType,
  onChangeTitle,
  onToggleStatus,
  onPromote,
  onDelete,
  onChangeNote,
  onUp,
  onDown,
}: ItemRowProps) {
  const [editing, setEditing] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const [commentsExpanded, setCommentsExpanded] = useState(false);
  const currentUserId = useBoardStore((s) => s.currentUserId);
  // Lazy — nothing fetches until the user opens the thread.
  const comments = usePlanningComments(item.id, currentUserId || null);

  const [draft, setDraft] = useState(item.title);
  // Mirror a parent title change into the draft during render, not in an effect —
  // see AGENTS.md, "Data fetching & loading states".
  const [syncedTitle, setSyncedTitle] = useState(item.title);
  const inputRef = useRef<HTMLInputElement | null>(null);

  if (syncedTitle !== item.title) {
    setSyncedTitle(item.title);
    setDraft(item.title);
  }

  useEffect(() => {
    if (editing) inputRef.current?.focus();
  }, [editing]);


  const commit = () => {
    setEditing(false);
    const trimmed = draft.trim();
    if (!trimmed) {
      setDraft(item.title);
      return;
    }
    if (trimmed !== item.title) onChangeTitle(trimmed);
  };

  // Enter commits, Escape cancels, arrows navigate; every other action is click-only.
  const onKey = (e: KeyboardEvent<HTMLElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      commit();
      return;
    }
    if (e.key === "Escape") {
      e.preventDefault();
      setDraft(item.title);
      setEditing(false);
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      onUp();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      onDown();
      return;
    }
  };

  const promoted = item.status === "promoted";
  const dropped = item.status === "dropped";
  const selected = item.status === "selected";
  const hasNote = (item.implementation_note ?? "").trim().length > 0;
  const hasDetails = hasNote;

  // One-glance "where is this item now". No "waiting on <person>" — no such data.
  const statusMeta = promoted
    ? { Icon: CheckCircle2, label: "บน Board แล้ว", cls: "text-emerald-600" }
    : dropped
      ? { Icon: Circle, label: "พักไว้ก่อน", cls: "text-slate-400" }
      : selected
        ? { Icon: ArrowRight, label: "เลือกไว้ส่งเข้า Board", cls: "text-primary" }
        : item.type === "Q"
          ? { Icon: Clock, label: "รอคำตอบ", cls: "text-amber-600" }
          : { Icon: Circle, label: "ยังอยู่ในบันทึก", cls: "text-slate-400" };
  const StatusIcon = statusMeta.Icon;

  return (
    <div
      id={`item-${item.id}`}
      className="flex flex-col border-b border-slate-100 last:border-b-0"
    >
      <div
        onClick={selectMode ? (promoted ? undefined : onToggleSelect) : onFocus}
        className={`group flex items-start gap-3 rounded-md border-l-2 px-2 py-2.5 ${
          isSelected ? "border-indigo-400" : "border-transparent"
        } ${
          focused
            ? "bg-indigo-50"
            : isSelected
              ? "bg-indigo-50/50"
              : "hover:bg-slate-50"
        } ${selectMode && !promoted ? "cursor-pointer" : ""}`}
      >
        {selectMode ? (
          // Promoted rows cannot be re-sent, so their checkbox is disabled.
          <span className="flex w-6 shrink-0 justify-center pt-0.5" aria-hidden>
            {promoted ? (
              <Square size={16} className="text-slate-200" />
            ) : isSelected ? (
              <CheckSquare size={16} className="text-primary" />
            ) : (
              <Square size={16} className="text-slate-400" />
            )}
          </span>
        ) : (
          <span className="w-6 shrink-0 pt-0.5 text-right text-xs font-semibold tabular-nums text-slate-400">
            {String(index + 1).padStart(2, "0")}
          </span>
        )}

        <div className="min-w-0 flex-1">
          {/* Line 1 — type chip, title, detail badges */}
          <div className="flex items-center gap-2">
            <ItemTypeChip type={item.type} />
            {editing && !selectMode ? (
              <input
                ref={inputRef}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onBlur={commit}
                onKeyDown={onKey}
                className="min-w-0 flex-1 rounded-md border border-indigo-300 bg-white px-2 py-1 text-sm text-slate-800 shadow-sm outline-none focus:ring-2 focus:ring-indigo-400"
              />
            ) : (
              // Display-only — renaming lives in the row menu, not a click on the title.
              <span
                className={`min-w-0 flex-1 truncate text-sm font-medium ${
                  dropped
                    ? "text-slate-400 line-through"
                    : promoted
                      ? "text-slate-500"
                      : "text-slate-800"
                }`}
              >
                {item.title}
              </span>
            )}
            {/* Indicator badges — visible at full opacity so a glance reveals
                which rows have detail attached without having to expand them. */}
            {!expanded && hasNote && (
              <span
                className="inline-flex shrink-0 items-center gap-0.5 rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600"
                title="มีรายละเอียด"
              >
                <FileText size={10} /> มีรายละเอียด
              </span>
            )}
          </div>

          {/* Line 2 — status, or an explicit "you're editing" hint while the
              title is being renamed so the edit state is unmistakable. */}
          {editing ? (
            <div className="mt-1 flex items-center gap-1.5 text-[11px] font-medium text-indigo-500">
              <Pencil size={11} />
              <span>กำลังแก้ไขชื่อ — Enter เพื่อบันทึก · Esc ยกเลิก</span>
            </div>
          ) : (
            <div
              className={`mt-1 flex items-center gap-1.5 text-[11px] font-medium ${statusMeta.cls}`}
            >
              <StatusIcon size={12} />
              <span>{statusMeta.label}</span>
            </div>
          )}
        </div>

        {/* Right action cluster — hidden in select-mode, where the row's only
            job is to be ticked. */}
        {!selectMode && (
        <div className="flex shrink-0 items-center gap-1 pt-0.5">
          {/* Direct promote — primary action on a live row (the picture's
              "→ ส่งขึ้น Board"). Promoted/paused rows hide it. */}
          {!promoted && !dropped && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onPromote();
              }}
              title="ส่งรายการนี้ขึ้น Board"
              className="inline-flex items-center gap-1 rounded-md border border-indigo-200 bg-indigo-50 px-2.5 py-1 text-xs font-semibold text-indigo-700 transition-colors hover:bg-indigo-100"
            >
              <ArrowRight size={13} /> ส่งขึ้น Board
            </button>
          )}
          {/* Comment count badge — click toggles the thread expansion. The
              count comes from the hook's local list (includes optimistic
              additions, excludes soft-deleted) so it stays in sync with what
              the thread actually shows. Initial value is "—" until first
              load; clicking triggers the fetch so an unopened row never
              burns a request. */}
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              if (!commentsExpanded && !comments.loaded) void comments.load();
              setCommentsExpanded((v) => !v);
            }}
            title={commentsExpanded ? "ซ่อนความคิดเห็น" : "ดูความคิดเห็น"}
            aria-label="ความคิดเห็น"
            aria-expanded={commentsExpanded}
            className={`inline-flex shrink-0 items-center gap-0.5 rounded px-1.5 py-0.5 text-[10px] font-semibold transition-colors ${
              commentsExpanded
                ? "bg-indigo-50 text-indigo-700"
                : "text-slate-500 hover:bg-slate-100"
            }`}
          >
            <MessageSquare size={12} />
            {comments.loaded
              ? comments.comments.filter((c) => !c.deleted_at).length
              : ""}
          </button>
          <ItemActionMenu
            currentType={item.type}
            dropped={dropped}
            promoted={promoted}
            expanded={expanded}
            hasDetails={hasDetails}
            onRename={() => setEditing(true)}
            onChangeType={onChangeType}
            onToggleDropped={() => onToggleStatus("dropped")}
            onDelete={onDelete}
            onToggleExpanded={() => setExpanded((v) => !v)}
          />
        </div>
        )}
      </div>
    {expanded && (
      <ItemDetailsPanel
        note={item.implementation_note}
        onChangeNote={onChangeNote}
      />
    )}
    {commentsExpanded && (
      <CommentThread
        comments={comments.comments}
        isLoading={comments.isLoading}
        loaded={comments.loaded}
        currentUserId={currentUserId || null}
        onCreate={comments.create}
        onEdit={comments.edit}
        onDelete={comments.remove}
      />
    )}
    </div>
  );
}
