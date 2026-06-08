import { memo, useEffect, useRef, useState } from "react";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";
import type { Subtask } from "@/types/board";

interface SubtaskItemProps {
  cardId: string;
  subtask: Subtask;
  onToggle: (cardId: string, subtaskId: string, currentStatus: boolean) => void;
  onUpdateTitle: (cardId: string, subtaskId: string, newTitle: string) => void;
  onDelete: (cardId: string, subtaskId: string) => void;
}

export const SubtaskItem = memo(function SubtaskItem({
  cardId,
  subtask,
  onToggle,
  onUpdateTitle,
  onDelete,
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
    <div className="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-slate-50 transition-colors group relative">
      <input
        type="checkbox"
        checked={subtask.is_done}
        onChange={() => onToggle(cardId, subtask.id, subtask.is_done)}
        className="shrink-0 rounded border-slate-300 text-blue-600 focus:ring-blue-500 cursor-pointer"
      />

      {isEditing ? (
        <input
          autoFocus
          value={editTitle}
          onChange={(e) => setEditTitle(e.target.value)}
          className="flex-1 text-sm px-2 py-0.5 border border-blue-400 rounded outline-none focus:ring-2 focus:ring-blue-100"
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
        <span
          className={`flex-1 text-sm px-2 py-0.5 truncate ${
            subtask.is_done ? "line-through text-slate-400" : "text-slate-700"
          }`}
        >
          {subtask.title}
        </span>
      )}

      <div className="relative shrink-0" ref={menuRef}>
        <button
          type="button"
          onClick={() => setMenuOpen((v) => !v)}
          className={`p-1 rounded transition-colors ${
            menuOpen
              ? "bg-slate-200 text-slate-700"
              : "text-slate-400 opacity-0 group-hover:opacity-100 hover:bg-slate-200 hover:text-slate-700"
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
              className="flex items-center gap-2 w-full px-3 py-1.5 text-xs text-red-600 hover:bg-red-50 cursor-pointer"
            >
              <Trash2 size={12} />
              Delete
            </button>
          </div>
        )}
      </div>
    </div>
  );
});
