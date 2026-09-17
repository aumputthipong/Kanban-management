import { memo, useEffect, useRef, useState } from "react";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import type { Subtask } from "@/types/board";
import { SubtaskCheckbox, subtaskRowClass } from "../../card-modal/modalParts";

interface SubtaskItemProps {
  cardId: string;
  subtask: Subtask;
  onToggle: (cardId: string, subtaskId: string, currentStatus: boolean) => void;
  onUpdateTitle: (cardId: string, subtaskId: string, newTitle: string) => void;
  onDelete: (cardId: string, subtaskId: string) => void;
  /** Mirrors the backend rule: only people who can edit the card can change its subtasks. */
  canEdit: boolean;
}

export const SubtaskItem = memo(function SubtaskItem({
  cardId,
  subtask,
  onToggle,
  onUpdateTitle,
  onDelete,
  canEdit,
}: SubtaskItemProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState(subtask.title);
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const onClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, [menuOpen]);

  const handleSave = () => {
    const trimmed = editTitle.trim();
    if (trimmed && trimmed !== subtask.title) {
      onUpdateTitle(cardId, subtask.id, trimmed);
    } else {
      setEditTitle(subtask.title);
    }
    setIsEditing(false);
  };

  const startEdit = () => {
    setEditTitle(subtask.title);
    setIsEditing(true);
    setMenuOpen(false);
  };

  return (
    <div className={`relative ${subtaskRowClass(subtask.is_done)}`}>
      <SubtaskCheckbox
        checked={subtask.is_done}
        disabled={!canEdit}
        label={`${subtask.is_done ? "ยกเลิก" : "ทำ"}เครื่องหมายเสร็จ: ${subtask.title}`}
        onToggle={() => onToggle(cardId, subtask.id, subtask.is_done)}
      />

      {isEditing ? (
        <input
          autoFocus
          value={editTitle}
          onChange={(e) => setEditTitle(e.target.value)}
          className="flex-1 min-w-0 text-sm px-1.5 py-0.5 bg-white border border-primary rounded-md outline-none focus:ring-3 focus:ring-surface-tint"
          onBlur={handleSave}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleSave();
            if (e.key === "Escape") {
              setEditTitle(subtask.title);
              setIsEditing(false);
            }
          }}
        />
      ) : (
        <span className="flex-1 min-w-0 truncate">
          {subtask.title}
        </span>
      )}

      {canEdit && (
        <div className="relative shrink-0" ref={menuRef}>
          <button
            type="button"
            onClick={() => setMenuOpen((v) => !v)}
            className={`p-1 rounded transition-colors ${
              menuOpen
                ? "bg-slate-200 text-slate-700"
                : "text-slate-400 opacity-0 group-hover:opacity-100 hover:bg-white hover:text-slate-900"
            }`}
            aria-label="Subtask options"
          >
            <MoreHorizontal size={14} />
          </button>

          {menuOpen && (
            <div className="absolute top-7 right-0 z-50 w-36 bg-white border border-slate-200 rounded-lg shadow-lg py-1 overflow-hidden">
              <button
                type="button"
                onClick={startEdit}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-slate-700 hover:bg-slate-50 cursor-pointer"
              >
                <Pencil size={12} />
                Rename
              </button>
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false);
                  onDelete(cardId, subtask.id);
                }}
                className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-danger hover:bg-red-100 cursor-pointer"
              >
                <Trash2 size={12} />
                Delete
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
});
