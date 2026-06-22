"use client";

import { useEffect, useRef, useState } from "react";
import { Tag as TagIcon } from "lucide-react";
import { useBoardStore } from "@/store/useBoardStore";
import { API_URL } from "@/lib/constants";
import type { Tag } from "@/types/board";
import { getColumnColorHex } from "./ColumnOptionsModal";

interface TagFilterDropdownProps {
  boardId: string;
}

export function TagFilterDropdown({ boardId }: TagFilterDropdownProps) {
  const { filterTagIds, toggleFilterTag, clearFilterTags } = useBoardStore();
  const [tags, setTags] = useState<Tag[]>([]);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!boardId) return;
    fetch(`${API_URL}/boards/${boardId}/tags`, { credentials: "include" })
      .then((r) => (r.ok ? r.json() : []))
      .then((data: Tag[]) => setTags(Array.isArray(data) ? data : []))
      .catch(() => {
        // Non-critical: the tag filter is an affordance, not core data. On a
        // failed load the dropdown simply shows no tags.
      });
  }, [boardId]);

  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  const activeCount = filterTagIds.length;
  const isActive = activeCount > 0;

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        className={`cursor-pointer flex items-center gap-1.5 h-7 px-2.5 rounded-lg border text-xs font-medium transition-colors ${
          isActive
            ? "bg-blue-50 border-blue-300 text-blue-700"
            : "bg-white border-slate-200 text-slate-600 hover:border-slate-400 hover:text-slate-800"
        }`}
      >
        <TagIcon size={13} />
        <span>Tags</span>
        {isActive && (
          <span className="flex items-center justify-center w-4 h-4 rounded-full bg-blue-500 text-white text-[9px] font-bold leading-none">
            {activeCount}
          </span>
        )}
        <svg
          className={`w-3 h-3 text-slate-400 transition-transform ${open ? "rotate-180" : ""}`}
          viewBox="0 0 10 10"
          fill="none"
        >
          <path
            d="M2 3.5L5 6.5L8 3.5"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
        </svg>
      </button>

      {open && (
        <div className="absolute top-9 left-0 z-50 w-56 bg-white border border-slate-200 rounded-xl shadow-lg py-1.5 overflow-hidden">
          {tags.length === 0 ? (
            <p className="px-3 py-2 text-xs text-slate-400 italic">
              No tags on this board
            </p>
          ) : (
            <div className="max-h-64 overflow-y-auto">
              {tags.map((tag) => {
                const checked = filterTagIds.includes(tag.id);
                const hex = getColumnColorHex(tag.color) ?? "#94a3b8";
                return (
                  <button
                    key={tag.id}
                    onClick={() => toggleFilterTag(tag.id)}
                    className={`cursor-pointer flex items-center gap-2 w-full px-3 py-1.5 text-xs text-left transition-colors ${
                      checked
                        ? "bg-blue-50 text-blue-700"
                        : "text-slate-700 hover:bg-slate-50"
                    }`}
                  >
                    <span
                      className="w-2 h-2 rounded-full shrink-0"
                      style={{ backgroundColor: hex }}
                    />
                    <span className="flex-1 font-medium truncate">{tag.name}</span>
                    {checked && (
                      <svg
                        className="w-3.5 h-3.5 text-blue-500 shrink-0"
                        viewBox="0 0 14 14"
                        fill="none"
                      >
                        <path
                          d="M2.5 7L5.5 10L11.5 4"
                          stroke="currentColor"
                          strokeWidth="1.8"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        />
                      </svg>
                    )}
                  </button>
                );
              })}
            </div>
          )}

          {isActive && (
            <>
              <div className="my-1 border-t border-slate-100" />
              <button
                onClick={() => {
                  clearFilterTags();
                  setOpen(false);
                }}
                className="cursor-pointer flex items-center gap-2 w-full px-3 py-1.5 text-xs text-slate-400 hover:text-slate-600 hover:bg-slate-50 transition-colors"
              >
                Clear filter
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}
