"use client";

import { memo } from "react";
import { Folder, X } from "lucide-react";
import { StatusDropdown } from "./StatusDropdown";
import { useBoardStore } from "@/store/useBoardStore";
import { useBoardActions } from "@/hooks/useBoardActions";
import type { FormState } from "./CardDetailModal";

interface CardModalHeaderProps {
  cardId: string;
  columnId: string;
  boardId: string;
  title: string;
  onTitleChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onCommit: (field: keyof FormState) => void;
  canEdit: boolean;
  onClose: () => void;
}

function CardModalHeaderImpl({
  cardId,
  columnId,
  boardId,
  title,
  onTitleChange,
  onCommit,
  canEdit,
  onClose,
}: CardModalHeaderProps) {
  const columns = useBoardStore((s) => s.columns);
  const boardTitle = useBoardStore((s) => s.boardMeta?.title);
  const { handleChangeColumn } = useBoardActions(boardId);

  return (
    <div className="flex items-start gap-3 px-6 pt-5 pb-4 shrink-0">
      <div className="w-8 h-8 rounded-lg bg-surface-tint grid place-items-center text-primary shrink-0">
        <Folder size={16} />
      </div>
      <div className="flex-1 min-w-0">
        {boardTitle && (
          <p className="mb-0.5 text-xs font-medium text-slate-600 truncate">{boardTitle}</p>
        )}
        {canEdit ? (
          <input
            type="text"
            aria-label="Title"
            value={title}
            onChange={onTitleChange}
            onBlur={() => onCommit("title")}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                e.currentTarget.blur();
              }
            }}
            placeholder="Enter card title..."
            className="-ml-2 w-full px-2 py-0.5 text-xl font-semibold tracking-tight text-slate-900 bg-transparent border border-transparent rounded-md hover:border-slate-200 focus:outline-none focus:border-primary focus:ring-3 focus:ring-surface-tint transition-colors placeholder:text-slate-400"
          />
        ) : (
          <h2 className="py-0.5 text-xl font-semibold tracking-tight text-slate-900">{title}</h2>
        )}
      </div>
      <div className="flex items-center gap-2 shrink-0 pt-1">
        <StatusDropdown
          columns={columns}
          currentColumnId={columnId}
          onChange={(newColId) => handleChangeColumn(cardId, newColId)}
          disabled={!canEdit}
        />
        <button
          type="button"
          onClick={onClose}
          aria-label="ปิด"
          className="w-7 h-7 grid place-items-center rounded-md text-slate-600 hover:bg-slate-50 transition-colors"
        >
          <X size={14} strokeWidth={2.2} />
        </button>
      </div>
    </div>
  );
}

export const CardModalHeader = memo(CardModalHeaderImpl);
