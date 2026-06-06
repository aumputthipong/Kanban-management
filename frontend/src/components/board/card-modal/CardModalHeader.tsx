"use client";

import { memo } from "react";
import { Folder, Pencil, X } from "lucide-react";
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
  const { handleChangeColumn } = useBoardActions(boardId);

  return (
    <div className="flex items-center gap-3.5 px-6 pt-5 pb-4 group shrink-0">
      <div className="w-9 h-9 rounded-lg bg-indigo-50 grid place-items-center text-blue-800 shrink-0">
        <Folder size={17} />
      </div>
      <div className="flex-1 relative min-w-0">
        {canEdit ? (
          <>
            <input
              type="text"
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
              className="w-full text-lg font-semibold tracking-tight text-slate-900 bg-transparent border border-transparent rounded-lg px-3 py-0.5 focus:outline-none focus:bg-white focus:border-blue-400 focus:ring-4 focus:ring-blue-50 hover:bg-slate-100 hover:border-slate-200 transition-colors cursor-text placeholder:text-slate-300 pr-10"
            />
            <div className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-300 opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity duration-200">
              <Pencil size={16} />
            </div>
          </>
        ) : (
          <p className="text-lg font-semibold tracking-tight text-slate-900 px-3 py-0.5">
            {title}
          </p>
        )}
      </div>
      <StatusDropdown
        columns={columns}
        currentColumnId={columnId}
        onChange={(newColId) => handleChangeColumn(cardId, newColId)}
        disabled={!canEdit}
      />
      <button
        onClick={onClose}
        className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition-colors shrink-0"
      >
        <X size={18} />
      </button>
    </div>
  );
}

export const CardModalHeader = memo(CardModalHeaderImpl);
