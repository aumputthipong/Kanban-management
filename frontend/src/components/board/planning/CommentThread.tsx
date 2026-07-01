"use client";

// CommentThread — comment list + compose row for one planning item. Lives
// inline below the row when expanded; capped width so multi-paragraph
// posts wrap naturally next to the parent item without taking over the
// session view.
//
// Deleted comments render as an italic "deleted" placeholder with the original author
// + timestamp so the thread position stays stable (no scroll jumps when
// someone removes a row). Edit / delete icons are own-comment-only —
// permission is enforced on the server too (404 if non-author tries).
import { KeyboardEvent, useState } from "react";
import { Pencil, Trash2 } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
import type { PlanningComment } from "@/types/planning";
import { formatRelativeFromNow } from "./planningFormat";

interface Props {
  comments: PlanningComment[];
  isLoading: boolean;
  loaded: boolean;
  currentUserId: string | null;
  onCreate: (body: string) => void;
  onEdit: (commentId: string, body: string) => void;
  onDelete: (commentId: string) => void;
}

export function CommentThread({
  comments,
  isLoading,
  loaded,
  currentUserId,
  onCreate,
  onEdit,
  onDelete,
}: Props) {
  const [draft, setDraft] = useState("");

  const handleSubmit = () => {
    if (!draft.trim()) return;
    onCreate(draft);
    setDraft("");
  };

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Cmd/Ctrl+Enter sends, mirroring Slack/Linear conventions. Plain
    // Enter inserts a newline so multi-line comments are still natural.
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="ml-8 mt-1 flex flex-col gap-3 rounded border border-slate-200 bg-white p-3">
      {isLoading && !loaded ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-3 w-32" />
          <Skeleton className="h-4 w-64" />
        </div>
      ) : comments.length === 0 ? (
        <p className="text-xs text-slate-400">ยังไม่มีความคิดเห็น · เริ่มได้เลย</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {comments.map((c) => (
            <CommentRow
              key={c.id}
              comment={c}
              isOwn={!!currentUserId && c.author_id === currentUserId}
              onEdit={onEdit}
              onDelete={onDelete}
            />
          ))}
        </ul>
      )}

      <div className="flex flex-col gap-1.5">
        <textarea
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={onKeyDown}
          rows={2}
          placeholder="เพิ่มความคิดเห็น... (Cmd/Ctrl+Enter เพื่อส่ง)"
          className="resize-y rounded border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-800 placeholder:text-slate-300 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-200"
        />
        <div className="flex justify-end">
          <button
            type="button"
            onClick={handleSubmit}
            disabled={draft.trim().length === 0}
            className="rounded-md bg-primary px-3 py-1 text-xs font-semibold text-white hover:bg-primary-hover disabled:opacity-40"
          >
            ส่ง
          </button>
        </div>
      </div>
    </div>
  );
}

interface RowProps {
  comment: PlanningComment;
  isOwn: boolean;
  onEdit: (commentId: string, body: string) => void;
  onDelete: (commentId: string) => void;
}

function CommentRow({ comment, isOwn, onEdit, onDelete }: RowProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(comment.body ?? "");

  // Soft-deleted comments keep their slot in the thread for context, but
  // body + edit/delete are hidden. The italic label tells the user this
  // is a placeholder, not a missing render.
  if (comment.deleted_at) {
    return (
      <li className="flex flex-col gap-0.5 text-xs">
        <span className="font-semibold text-slate-700">{comment.author_name || "—"}</span>
        <span className="italic text-slate-400">
          ความคิดเห็นถูกลบแล้ว · {formatRelativeFromNow(comment.deleted_at)}
        </span>
      </li>
    );
  }

  const edited = comment.updated_at !== comment.created_at;

  return (
    <li className="flex flex-col gap-1">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-xs">
          <span className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-[10px] font-bold text-indigo-700">
            {(comment.author_name || "?").slice(0, 1).toUpperCase()}
          </span>
          <span className="font-semibold text-slate-700">{comment.author_name || "—"}</span>
          <span className="text-slate-400">· {formatRelativeFromNow(comment.created_at)}</span>
          {edited && <span className="text-slate-400">· แก้ไขแล้ว</span>}
        </div>
        {isOwn && !editing && (
          <div className="flex items-center gap-0.5 opacity-60 hover:opacity-100">
            <button
              type="button"
              onClick={() => {
                setDraft(comment.body ?? "");
                setEditing(true);
              }}
              title="Edit"
              aria-label="Edit"
              className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700"
            >
              <Pencil size={12} />
            </button>
            <button
              type="button"
              onClick={() => onDelete(comment.id)}
              title="Delete"
              aria-label="Delete"
              className="rounded p-1 text-slate-400 hover:bg-red-50 hover:text-red-600"
            >
              <Trash2 size={12} />
            </button>
          </div>
        )}
      </div>

      {editing ? (
        <div className="flex flex-col gap-1.5">
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            rows={2}
            className="resize-y rounded border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-800 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-200"
          />
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => {
                setDraft(comment.body ?? "");
                setEditing(false);
              }}
              className="rounded px-2 py-0.5 text-xs text-slate-600 hover:bg-slate-100"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={() => {
                onEdit(comment.id, draft);
                setEditing(false);
              }}
              disabled={draft.trim().length === 0 || draft === comment.body}
              className="rounded bg-primary px-2 py-0.5 text-xs font-semibold text-white hover:bg-primary-hover disabled:opacity-40"
            >
              Save
            </button>
          </div>
        </div>
      ) : (
        <p className="whitespace-pre-wrap text-sm text-slate-800">{comment.body}</p>
      )}
    </li>
  );
}
