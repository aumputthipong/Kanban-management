"use client";

import { useState } from "react";
import { Crown, Loader2 } from "lucide-react";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { ApiError, apiClient } from "@/lib/apiClient";
import { useBoardStore } from "@/store/useBoardStore";
import { useToastStore } from "@/store/useToastStore";
import type { BoardMember } from "@/types/board";
import { DangerRow } from "./DangerRow";
import { SelectField } from "./SettingsParts";

export function TransferOwnershipRow({ boardId }: { boardId: string }) {
  const boardMembers = useBoardStore((s) => s.boardMembers);
  const setBoardMembers = useBoardStore((s) => s.setBoardMembers);
  const showToast = useToastStore((s) => s.show);
  const candidates = boardMembers.filter((m) => m.role !== "owner");
  const [targetId, setTargetId] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [transferring, setTransferring] = useState(false);

  const target = candidates.find((m) => m.user_id === targetId);

  const transfer = async () => {
    if (!target) return;
    setTransferring(true);
    try {
      await apiClient(`/boards/${boardId}/members/transfer-ownership`, {
        data: { user_id: target.user_id },
      });
      const fresh: BoardMember[] = await apiClient(`/boards/${boardId}/members`);
      setBoardMembers(Array.isArray(fresh) ? fresh.filter(Boolean) : []);
      showToast({ message: `โอนสิทธิ์ Owner ให้ ${target.full_name} แล้ว — ตอนนี้คุณเป็น Manager`, duration: 5000 });
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) return;
      showToast({ message: err instanceof Error ? err.message : "โอนสิทธิ์ไม่สำเร็จ", duration: 4000 });
    } finally {
      setTransferring(false);
    }
  };

  return (
    <>
      <DangerRow
        title="โอนสิทธิ์ Owner"
        help={
          candidates.length === 0
            ? "เชิญสมาชิกเข้า Project ก่อน จึงจะโอนสิทธิ์ได้"
            : "คนที่เลือกจะเป็น Owner คนใหม่ ส่วนคุณจะกลายเป็น Manager — ทำแล้วย้อนกลับเองไม่ได้"
        }
      >
        {candidates.length > 0 && (
          <div className="flex items-center gap-2">
            <SelectField
              value={targetId}
              options={[
                { value: "", label: "เลือกสมาชิก" },
                ...candidates.map((m) => ({ value: m.user_id, label: m.full_name })),
              ]}
              onChange={setTargetId}
              disabled={transferring}
              ariaLabel="new owner"
            />
            <button
              type="button"
              disabled={!target || transferring}
              onClick={() => setConfirmOpen(true)}
              className="inline-flex items-center gap-2 h-10 px-4 rounded-lg border border-slate-200 text-[13.5px] font-bold text-slate-700 hover:bg-slate-50 hover:border-slate-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {transferring ? <Loader2 size={16} className="animate-spin" /> : <Crown size={16} />}
              Transfer
            </button>
          </div>
        )}
      </DangerRow>

      <ConfirmDialog
        open={confirmOpen}
        title={`โอนสิทธิ์ Owner ให้ ${target?.full_name ?? ""}?`}
        description="คุณจะกลายเป็น Manager และจะลบหรือ Stash Project นี้ไม่ได้อีก จนกว่า Owner คนใหม่จะโอนสิทธิ์กลับมา"
        confirmLabel="Transfer"
        cancelLabel="Cancel"
        destructive
        onConfirm={() => {
          setConfirmOpen(false);
          transfer();
        }}
        onCancel={() => setConfirmOpen(false)}
      />
    </>
  );
}
