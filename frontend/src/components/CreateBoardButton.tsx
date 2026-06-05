"use client";

import { apiClient } from "@/lib/apiClient";
import { useRouter } from "next/navigation";
import { useState, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { Plus, X, FolderPlus, Loader2 } from "lucide-react";

export function CreateBoardButton() {
  const router = useRouter();
  const [isOpen, setIsOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleOpen = () => setIsOpen(true);

  const handleClose = () => {
    if (isCreating) return;
    setIsOpen(false);
    setTitle("");
    setError(null);
  };

  // Escape to close
  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") handleClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [isOpen, isCreating]);

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen) setTimeout(() => inputRef.current?.focus(), 50);
  }, [isOpen]);

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
        data: { title: trimmedTitle },
      });
      handleClose();
      router.push(`/board/${newBoard.id}/tasks`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "สร้างโปรเจกต์ไม่สำเร็จ");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <>
      <button
        onClick={handleOpen}
        className="flex items-center gap-1.5 bg-blue-600 hover:bg-blue-700 active:scale-95 text-white px-4 py-2 rounded-lg font-medium transition-all shadow-sm text-sm"
      >
        <Plus size={16} />
        โปรเจกต์ใหม่
      </button>

      {isOpen &&
        createPortal(
          <>
            {/* Backdrop */}
            <div
              className="fixed inset-0 z-9998 bg-black/40 backdrop-blur-sm"
              onClick={handleClose}
            />

            {/* Modal */}
            <div className="fixed inset-0 z-9999 flex items-center justify-center pointer-events-none px-4">
              <div
                className="pointer-events-auto w-full max-w-sm bg-white rounded-2xl shadow-2xl flex flex-col overflow-hidden"
                onClick={(e) => e.stopPropagation()}
              >
                {/* Header */}
                <div className="flex items-center justify-between px-5 pt-5 pb-4 border-b border-slate-100">
                  <div className="flex items-center gap-2.5">
                    <div className="p-1.5 bg-blue-50 rounded-lg">
                      <FolderPlus size={16} className="text-blue-600" />
                    </div>
                    <h2 className="text-sm font-bold text-slate-800">
                      สร้างโปรเจกต์ใหม่
                    </h2>
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
                      ชื่อโปรเจกต์
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

                  {/* Footer */}
                  <div className="flex items-center justify-end gap-2 pt-1">
                    <button
                      type="button"
                      onClick={handleClose}
                      disabled={isCreating}
                      className="px-4 py-2 text-sm rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors disabled:opacity-40"
                    >
                      ยกเลิก
                    </button>
                    <button
                      type="submit"
                      disabled={isCreating || !title.trim()}
                      className="flex items-center gap-2 px-4 py-2 text-sm rounded-lg bg-blue-600 text-white font-medium hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                    >
                      {isCreating ? (
                        <>
                          <Loader2 size={14} className="animate-spin" />
                          กำลังสร้าง...
                        </>
                      ) : (
                        "สร้างโปรเจกต์"
                      )}
                    </button>
                  </div>
                </form>
              </div>
            </div>
          </>,
          document.body,
        )}
    </>
  );
}
