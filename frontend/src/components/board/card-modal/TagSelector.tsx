"use client";

import { memo, useState, useRef, useEffect } from "react";
import { createPortal } from "react-dom";
import { Plus } from "lucide-react";
import type { Tag } from "@/types/board";
import { TagChip } from "@/components/board/task-board/TagChip";
import { COLUMN_COLOR_PALETTE } from "@/components/board/task-board/ColumnOptionsModal";
import { API_URL } from "@/lib/constants";
import { Skeleton } from "@/components/ui/Skeleton";

const TAG_COLORS = COLUMN_COLOR_PALETTE.filter((c) => c.key !== null) as {
  key: string;
  hex: string;
  label: string;
}[];

const MAX_TAGS = 5;

interface TagSelectorProps {
  boardId: string;
  selected: Tag[];
  onChange: (tags: Tag[]) => void;
  /** Persist the current tag set (per-field auto-save). Called after add/remove. */
  onCommit?: () => void;
  canEdit: boolean;
}

function TagSelectorImpl({ boardId, selected, onChange, onCommit, canEdit }: TagSelectorProps) {
  const [boardTags, setBoardTags] = useState<Tag[]>([]);
  const [loadingTags, setLoadingTags] = useState(true);
  const [loadedBoardId, setLoadedBoardId] = useState(boardId);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newColor, setNewColor] = useState(TAG_COLORS[0].key);
  const [dropdownStyle, setDropdownStyle] = useState<React.CSSProperties>({});
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Reset loading state on board switch using the setState-during-render pattern
  // (https://react.dev/reference/react/useState#storing-information-from-previous-renders).
  if (loadedBoardId !== boardId) {
    setLoadedBoardId(boardId);
    setBoardTags([]);
    setLoadingTags(true);
  }

  useEffect(() => {
    let cancelled = false;
    fetch(`${API_URL}/boards/${boardId}/tags`, { credentials: "include" })
      .then((r) => r.json())
      .then((data: Tag[]) => {
        if (!cancelled) setBoardTags(Array.isArray(data) ? data : []);
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoadingTags(false);
      });
    return () => {
      cancelled = true;
    };
  }, [boardId]);

  // Position dropdown relative to input using viewport coords
  useEffect(() => {
    if (!open || !inputRef.current) return;
    const rect = inputRef.current.getBoundingClientRect();
    setDropdownStyle({
      position: "fixed",
      top: rect.bottom + 4,
      left: rect.left,
      width: Math.max(rect.width, 192),
      zIndex: 99999,
    });
  }, [open, creating]);

  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (
        dropdownRef.current?.contains(e.target as Node) ||
        inputRef.current?.contains(e.target as Node)
      ) return;
      setOpen(false);
      setCreating(false);
      setQuery("");
    };
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  const selectedIds = new Set(selected.map((t) => t.id));
  const filtered = boardTags.filter(
    (t) =>
      !selectedIds.has(t.id) &&
      t.name.toLowerCase().includes(query.toLowerCase()),
  );
  const exactMatch = boardTags.some(
    (t) => t.name.toLowerCase() === query.trim().toLowerCase(),
  );

  const handleSelect = (tag: Tag) => {
    if (selected.length >= MAX_TAGS) return;
    onChange([...selected, tag]);
    onCommit?.();
    setQuery("");
    setOpen(false);
    inputRef.current?.focus();
  };

  const handleRemove = (tagId: string) => {
    onChange(selected.filter((t) => t.id !== tagId));
    onCommit?.();
  };

  const handleCreate = async () => {
    const name = query.trim();
    if (!name) return;
    try {
      const res = await fetch(`${API_URL}/boards/${boardId}/tags`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, color: newColor }),
      });
      if (!res.ok) return;
      const tag: Tag = await res.json();
      setBoardTags((prev) => [...prev, tag]);
      handleSelect(tag);
      setCreating(false);
      setNewColor(TAG_COLORS[0].key);
    } catch {}
  };

  if (!canEdit) {
    return (
      <div className="flex flex-wrap gap-1">
        {selected.length === 0 ? (
          <span className="text-xs text-slate-400">—</span>
        ) : (
          selected.map((tag) => <TagChip key={tag.id} tag={tag} />)
        )}
      </div>
    );
  }

  const dropdown = open ? (
    <div
      ref={dropdownRef}
      style={dropdownStyle}
      className="bg-white border border-slate-200 rounded-xl shadow-xl overflow-hidden"
    >
      {/* Loading tags */}
      {loadingTags && boardTags.length === 0 && (
        <div className="px-3 py-2 flex flex-col gap-1.5">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-5 w-24 rounded-full" />
          ))}
        </div>
      )}

      {/* Existing tags */}
      {filtered.map((tag) => (
        <button
          key={tag.id}
          type="button"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => handleSelect(tag)}
          className="w-full text-left px-3 py-1.5 hover:bg-slate-50 flex items-center gap-2"
        >
          <TagChip tag={tag} />
        </button>
      ))}

      {/* Create new */}
      {query.trim() && !exactMatch && !creating && (
        <button
          type="button"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => setCreating(true)}
          className="w-full text-left px-3 py-1.5 hover:bg-slate-50 flex items-center gap-1.5 text-xs text-blue-600 font-semibold"
        >
          <Plus size={12} />
          Create &ldquo;{query.trim()}&rdquo;
        </button>
      )}

      {/* Color picker when creating */}
      {creating && (
        <div className="px-3 py-2 flex flex-col gap-2 border-t border-slate-100">
          <p className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">Pick color</p>
          <div className="flex flex-wrap gap-1.5">
            {TAG_COLORS.map(({ key, hex }) => (
              <button
                key={key}
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => setNewColor(key)}
                className="w-5 h-5 rounded-full border-2 transition-transform hover:scale-110"
                style={{
                  backgroundColor: hex,
                  borderColor: newColor === key ? "#1e293b" : "transparent",
                  outline: newColor === key ? "2px solid #1e293b" : "none",
                  outlineOffset: "2px",
                }}
              />
            ))}
          </div>
          <button
            type="button"
            onMouseDown={(e) => e.preventDefault()}
            onClick={handleCreate}
            className="text-xs bg-blue-500 hover:bg-blue-600 text-white font-semibold py-1 rounded-md transition-colors cursor-pointer"
          >
            Create
          </button>
        </div>
      )}

      {filtered.length === 0 && !query.trim() && !creating && (
        <p className="px-3 py-2 text-xs text-slate-400">No tags yet</p>
      )}
    </div>
  ) : null;

  return (
    <div className="flex flex-col gap-2">
      {/* Selected chips */}
      {selected.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {selected.map((tag) => (
            <TagChip key={tag.id} tag={tag} onRemove={handleRemove} />
          ))}
        </div>
      )}

      {/* Input */}
      {selected.length < MAX_TAGS && (
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
            setCreating(false);
          }}
          onFocus={() => setOpen(true)}
          onPointerDown={(e) => e.stopPropagation()}
          placeholder="Add tag…"
          className="w-full text-xs px-2 py-1 rounded-md border border-slate-200 bg-white text-slate-700 placeholder:text-slate-400 focus:outline-none focus:border-blue-400 focus:ring-1 focus:ring-blue-300"
        />
      )}

      {typeof document !== "undefined" && createPortal(dropdown, document.body)}
    </div>
  );
}

export const TagSelector = memo(TagSelectorImpl);
