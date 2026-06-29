"use client";

import { apiClient } from "@/lib/apiClient";
import { useRouter } from "next/navigation";
import { useState, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { X, FolderPlus, Loader2 } from "lucide-react";
import { BoardAppearancePicker } from "@/components/board/settings/BoardAppearancePicker";
import {
  DEFAULT_BOARD_COLOR,
  DEFAULT_BOARD_ICON,
  type BoardIconKey,
} from "@/lib/boardAppearance";

interface CreateProjectModalProps {
  onClose: () => void;
}

/**
 * Create-project modal — name (required) + description (optional) + appearance
 * (icon + colour, defaulting to the first set). Mirrors the create-task modal:
 * everything needed to spin up a board lives here so a board is born with its
 * chosen identity in a single POST /boards (no create-then-PATCH round trip).
 *
 * The icon/colour controls reuse BoardAppearancePicker — the same widget the
 * board settings page uses — so "what you pick when creating" matches "what you
 * edit later" pixel-for-pixel.
 */
export function CreateProjectModal({ onClose }: CreateProjectModalProps) {
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [color, setColor] = useState<string>(DEFAULT_BOARD_COLOR);
  const [icon, setIcon] = useState<BoardIconKey>(DEFAULT_BOARD_ICON);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleClose = () => {
    if (isCreating) return;
    onClose();
  };

  // Escape to close + focus the name field on open.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isCreating) onClose();
    };
    window.addEventListener("keydown", handler);
    const t = setTimeout(() => inputRef.current?.focus(), 50);
    return () => {
      window.removeEventListener("keydown", handler);
      clearTimeout(t);
    };
  }, [isCreating, onClose]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmedTitle = title.trim();
    if (!trimmedTitle) {
      setError("กรุณาใส่ชื่อโปรเจกต์");
      return;
    }

    setIsCreating(true);
    setError(null);

    try {
      const newBoard = await apiClient<{ id: string }>("/boards", {
        data: {
          title: trimmedTitle,
          description: description.trim(),
          color,
          icon,
        },
      });
      router.push(`/board/${newBoard.id}/tasks`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "สร้างโปรเจกต์ไม่สำเร็จ");
      setIsCreating(false);
    }
  };

  return createPortal(
    <>
      <div
        className="fixed inset-0 z-9998 bg-black/40 backdrop-blur-sm"
        onClick={handleClose}
      />

      <div className="fixed inset-0 z-9999 flex items-center justify-center pointer-events-none px-4">
        <div
          className="pointer-events-auto w-full max-w-md bg-white rounded-2xl shadow-2xl flex flex-col overflow-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-5 pt-5 pb-4 border-b border-slate-100">
            <div className="flex items-center gap-2.5">
              <div className="p-1.5 bg-blue-50 rounded-lg">
                <FolderPlus size={16} className="text-primary" />
              </div>
              <h2 className="text-sm font-bold text-slate-800">New Project</h2>
            </div>
            <button
              onClick={handleClose}
              disabled={isCreating}
              className="p-1 text-slate-400 hover:text-slate-600 rounded-md hover:bg-slate-100 transition-colors disabled:opacity-40"
            >
              <X size={16} />
            </button>
          </div>

          {/* Body */}
          <form onSubmit={handleSubmit} className="px-5 py-5 flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="project-name"
                className="text-xs font-semibold text-slate-500 uppercase tracking-wide"
              >
                Project Name
              </label>
              <input
                ref={inputRef}
                id="project-name"
                type="text"
                value={title}
                onChange={(e) => {
                  setTitle(e.target.value);
                  if (error) setError(null);
                }}
                disabled={isCreating}
                placeholder="ใส่ชื่อโปรเจกต์ของคุณ"
                className={`text-sm border rounded-lg px-3 py-2.5 focus:outline-none focus:ring-2 transition disabled:opacity-50 disabled:bg-slate-50 ${
                  error
                    ? "border-red-300 focus:ring-red-100 focus:border-red-400"
                    : "border-slate-200 focus:ring-blue-100 focus:border-blue-400"
                }`}
              />
              {error && (
                <p className="text-xs text-red-500 font-medium">{error}</p>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label
                htmlFor="project-desc"
                className="text-xs font-semibold text-slate-500 uppercase tracking-wide"
              >
                Description
              </label>
              <textarea
                id="project-desc"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={isCreating}
                rows={2}
                maxLength={160}
                placeholder="อธิบายโปรเจกต์สั้น ๆ (ไม่บังคับ)"
                className="text-sm border border-slate-200 rounded-lg px-3 py-2.5 resize-none focus:outline-none focus:ring-2 focus:ring-blue-100 focus:border-blue-400 transition disabled:opacity-50 disabled:bg-slate-50"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
                Appearance
              </span>
              <BoardAppearancePicker
                name={title}
                color={color}
                icon={icon}
                canManage={!isCreating}
                saving={null}
                onPickColor={setColor}
                onPickIcon={setIcon}
              />
            </div>

            {/* Footer */}
            <div className="flex items-center justify-end gap-2 pt-1">
              <button
                type="button"
                onClick={handleClose}
                disabled={isCreating}
                className="px-4 py-2 text-sm rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors disabled:opacity-40"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isCreating || !title.trim()}
                className="flex items-center gap-2 px-4 py-2 text-sm rounded-lg bg-primary text-white font-medium hover:bg-primary-hover disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              >
                {isCreating ? (
                  <>
                    <Loader2 size={14} className="animate-spin" />
                    Creating…
                  </>
                ) : (
                  "Create Project"
                )}
              </button>
            </div>
          </form>
        </div>
      </div>
    </>,
    document.body,
  );
}
