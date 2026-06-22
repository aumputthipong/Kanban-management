"use client";

import { useState } from "react";
import { Trash2, RotateCcw, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { API_URL } from "@/lib/constants";
import { boardColor, BoardGlyph } from "@/lib/boardAppearance";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { TypeToConfirmDialog } from "@/components/ui/TypeToConfirmDialog";
import type { StashedBoard } from "@/app/(app)/stash/page"; // ดึง Type มาจากหน้า page

interface StashTableProps {
  boards: StashedBoard[];
}

export function StashTable({ boards: initialBoards }: StashTableProps) {
  const router = useRouter();
  const [boards, setBoards] = useState(initialBoards);
  const [loadingId, setLoadingId] = useState<string | null>(null);
  // Only one dialog is open at a time; the target board drives its content.
  const [restoreTarget, setRestoreTarget] = useState<StashedBoard | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<StashedBoard | null>(null);

  const handleRestore = async (board: StashedBoard) => {
    setLoadingId(board.id);
    try {
      const res = await fetch(`${API_URL}/stash/${board.id}/restore`, {
        method: "PATCH",
        credentials: "include",
      });
      if (!res.ok) throw new Error("Failed to restore");

      setBoards((prev) => prev.filter((b) => b.id !== board.id));
      setRestoreTarget(null);
      router.refresh();
    } catch (err) {
      console.error(err);
    } finally {
      setLoadingId(null);
    }
  };

  const handleDelete = async (board: StashedBoard) => {
    setLoadingId(board.id);
    try {
      const res = await fetch(`${API_URL}/stash/${board.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) throw new Error("Failed to delete");

      setBoards((prev) => prev.filter((b) => b.id !== board.id));
      setDeleteTarget(null);
    } catch (err) {
      console.error(err);
    } finally {
      setLoadingId(null);
    }
  };

  return (
    <>
      <div className="divide-y divide-slate-100">
        {boards.map((board) => {
          const isLoading = loadingId === board.id;
          const accent = boardColor(board.color);
          const hasDesc = !!board.description && board.description.trim() !== "";

          return (
            <div
              key={board.id}
              className="flex items-center gap-4 px-5 py-4 hover:bg-slate-50 transition-colors"
            >
              {/* identity — mirrors the project-list card name block */}
              <div
                className="h-[38px] w-[38px] rounded-lg flex items-center justify-center text-white shrink-0"
                style={{ background: accent }}
              >
                <BoardGlyph icon={board.icon} size={20} />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-sm font-bold text-slate-800 truncate">
                  {board.title}
                </div>
                <div
                  className={`text-xs mt-0.5 truncate ${hasDesc ? "text-slate-400" : "text-slate-300 italic"}`}
                >
                  {hasDesc ? board.description : "ยังไม่มีคำอธิบาย"}
                </div>
              </div>

              {/* stashed-at */}
              <div className="hidden sm:block text-xs text-slate-500 shrink-0 text-right">
                <div className="text-slate-400">เก็บเข้าคลังเมื่อ</div>
                <div className="font-semibold text-slate-600 mt-0.5">
                  {new Date(board.stashed_at).toLocaleDateString("th-TH", {
                    day: "numeric",
                    month: "short",
                    year: "numeric",
                  })}
                </div>
              </div>

              {/* actions */}
              <div className="flex items-center gap-2 shrink-0">
                <button
                  onClick={() => setRestoreTarget(board)}
                  disabled={isLoading}
                  className="inline-flex items-center gap-1.5 h-9 px-3 rounded-lg border border-slate-200 text-[13px] font-semibold text-slate-700 hover:bg-slate-50 hover:border-slate-300 disabled:opacity-50 transition-colors"
                  title="กู้คืน"
                >
                  {isLoading ? (
                    <Loader2 size={15} className="animate-spin" />
                  ) : (
                    <RotateCcw size={15} />
                  )}
                  กู้คืน
                </button>
                <button
                  onClick={() => setDeleteTarget(board)}
                  disabled={isLoading}
                  className="inline-flex items-center gap-1.5 h-9 px-3 rounded-lg border border-red-200 text-[13px] font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50 transition-colors"
                  title="ลบถาวร"
                >
                  <Trash2 size={15} />
                  ลบถาวร
                </button>
              </div>
            </div>
          );
        })}
      </div>

      {/* Restore — simple confirm */}
      <ConfirmDialog
        open={restoreTarget !== null}
        title="กู้คืนบอร์ดนี้?"
        description={
          restoreTarget
            ? `“${restoreTarget.title}” จะกลับไปแสดงในรายการโปรเจกต์ที่ใช้งาน`
            : undefined
        }
        confirmLabel="กู้คืน"
        cancelLabel="ยกเลิก"
        onConfirm={() => restoreTarget && handleRestore(restoreTarget)}
        onCancel={() => setRestoreTarget(null)}
      />

      {/* Permanent delete — GitHub-style type-to-confirm */}
      <TypeToConfirmDialog
        open={deleteTarget !== null}
        title="ลบบอร์ดนี้ถาวร?"
        description={
          <>
            การลบนี้ <b className="text-red-600">ไม่สามารถกู้คืนได้</b> — บอร์ด
            การ์ด และข้อมูลทั้งหมดจะถูกลบอย่างถาวร
          </>
        }
        confirmPhrase={deleteTarget?.title ?? ""}
        inputLabel={
          <>
            พิมพ์ <b className="text-slate-900">{deleteTarget?.title}</b>{" "}
            เพื่อยืนยัน
          </>
        }
        confirmLabel="ลบถาวร"
        cancelLabel="ยกเลิก"
        loading={deleteTarget !== null && loadingId === deleteTarget.id}
        onConfirm={() => deleteTarget && handleDelete(deleteTarget)}
        onCancel={() => setDeleteTarget(null)}
      />
    </>
  );
}
