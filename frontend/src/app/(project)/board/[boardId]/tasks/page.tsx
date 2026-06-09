"use client";

import { use, useEffect, useState, useRef } from "react";
import { KanbanBoard } from "@/components/board/task-board/KanbanBoard";
import { useCardHighlightStore } from "@/store/useCardHighlightStore";
import { useBoardActions } from "@/hooks/useBoardActions";
import { useBoardStore } from "@/store/useBoardStore";
import { Plus } from "lucide-react";
import { MemberFilterBar } from "@/components/board/task-board/MemberFilterBar";
import { PriorityFilterDropdown } from "@/components/board/task-board/PriorityFilterDropdown";
import { TagFilterDropdown } from "@/components/board/task-board/TagFilterDropdown";
import { useCanManageBoard } from "@/hooks/useBoardRole";

interface PageProps {
  params: Promise<{ boardId: string }>;
}


function AddColumnButton({ onAdd }: { onAdd: (title: string) => void }) {
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const submit = () => {
    if (value.trim()) onAdd(value.trim());
    setValue("");
    setEditing(false);
  };

  if (!editing) {
    return (
      <button
        onClick={() => {
          setEditing(true);
          setTimeout(() => inputRef.current?.focus(), 0);
        }}
        className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 hover:text-blue-600 transition-colors shadow-sm"
      >
        <Plus size={14} /> Add column
      </button>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <input
        ref={inputRef}
        autoFocus
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") submit();
          if (e.key === "Escape") {
            setValue("");
            setEditing(false);
          }
        }}
        placeholder="Column name..."
        className="text-sm border border-blue-400 rounded-lg px-3 py-1.5 outline-none ring-2 ring-blue-100 w-40"
      />
      <button
        onClick={submit}
        className="text-xs bg-blue-500 hover:bg-blue-600 text-white rounded-lg px-3 py-1.5 font-medium transition-colors"
      >
        Add
      </button>
      <button
        onClick={() => {
          setValue("");
          setEditing(false);
        }}
        className="text-xs border border-slate-200 hover:bg-slate-50 rounded-lg px-3 py-1.5 text-slate-600 transition-colors"
      >
        Cancel
      </button>
    </div>
  );
}

function BoardToolbar({ boardId }: { boardId: string }) {
  const { handleAddColumn } = useBoardActions(boardId);
  const canManage = useCanManageBoard();

  return (
    <div className="-mx-8 flex items-center gap-3 px-8 h-14 bg-slate-50 border-b border-slate-200 mb-6">
      <MemberFilterBar />
      <div className="w-px h-5 bg-slate-300 mx-1" />
      <PriorityFilterDropdown />
      <TagFilterDropdown boardId={boardId} />
      {canManage && (
        <>
          <div className="w-px h-5 bg-slate-300 mx-1" />
          <AddColumnButton onAdd={handleAddColumn} />
        </>
      )}
    </div>
  );
}

export default function KanbanPage({ params }: PageProps) {
  const { boardId } = use(params);
  const setTarget = useCardHighlightStore((s) => s.setTarget);

  // Deep link from My Work ("เปิดในบอร์ด" → ?card=<id>): flag the card for the
  // highlight, then strip the param so a reload/back doesn't re-trigger it.
  // Reads window.location directly to avoid a useSearchParams Suspense boundary.
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const cardId = params.get("card");
    if (!cardId) return;
    setTarget(cardId);
    params.delete("card");
    const qs = params.toString();
    window.history.replaceState(
      null,
      "",
      `${window.location.pathname}${qs ? `?${qs}` : ""}`,
    );
  }, [setTarget]);

  return (
    <div className="animate-in fade-in slide-in-from-bottom-2 duration-500 flex flex-col">
      <BoardToolbar boardId={boardId} />
      <KanbanBoard boardId={boardId} />
    </div>
  );
}
