"use client";

import { use, useEffect, useState } from "react";
import { KanbanBoard } from "@/components/board/task-board/KanbanBoard";
import { useCardHighlightStore } from "@/store/useCardHighlightStore";
import { useBoardActions } from "@/hooks/useBoardActions";
import { useBoardStore } from "@/store/useBoardStore";
import { Plus } from "lucide-react";
import { MemberFilterBar } from "@/components/board/task-board/MemberFilterBar";
import { PriorityFilterDropdown } from "@/components/board/task-board/PriorityFilterDropdown";
import { TagFilterDropdown } from "@/components/board/task-board/TagFilterDropdown";
import { CreateTaskModal } from "@/components/board/task-board/CreateTaskModal";
import { ColumnOptionsModal } from "@/components/board/task-board/ColumnOptionsModal";
import { useCanManageBoard } from "@/hooks/useBoardRole";

interface PageProps {
  params: Promise<{ boardId: string }>;
}


function AddColumnButton({
  onAdd,
}: {
  onAdd: (title: string, category: "TODO" | "DONE", color: string | null) => void;
}) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="cursor-pointer flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 hover:text-primary transition-colors shadow-sm"
      >
        <Plus size={14} /> Add column
      </button>
      {open && (
        <ColumnOptionsModal
          open
          mode="create"
          initialTitle=""
          initialCategory="TODO"
          initialColor={null}
          cardCount={0}
          onSave={onAdd}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  );
}

function BoardToolbar({ boardId }: { boardId: string }) {
  const { handleAddColumn, handleAddCard } = useBoardActions(boardId);
  const canManage = useCanManageBoard();
  const [creating, setCreating] = useState(false);

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
      <button
        onClick={() => setCreating(true)}
        className="ml-auto cursor-pointer inline-flex items-center gap-1.5 px-3.5 py-1.5 text-sm font-semibold text-white bg-primary rounded-lg hover:bg-primary-hover transition-colors shadow-sm"
      >
        <Plus size={15} /> New Task
      </button>
      {creating && (
        <CreateTaskModal
          onClose={() => setCreating(false)}
          onCreate={(columnId, title, opts) => handleAddCard(columnId, title, opts)}
        />
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
    <div className="animate-in fade-in duration-200 flex flex-col">
      <BoardToolbar boardId={boardId} />
      <KanbanBoard boardId={boardId} />
    </div>
  );
}
