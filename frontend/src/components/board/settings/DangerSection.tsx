"use client";

import { useState } from "react";
import { AlertTriangle, Archive, Loader2 } from "lucide-react";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { DangerRow } from "./DangerRow";
import { SectionCard } from "./SettingsParts";
import { TransferOwnershipRow } from "./TransferOwnershipRow";

interface DangerSectionProps {
  boardId: string;
  onArchive: () => void;
  isArchiving: boolean;
}

export function DangerSection({ boardId, onArchive, isArchiving }: DangerSectionProps) {
  const [confirmArchive, setConfirmArchive] = useState(false);

  return (
    <>
      <SectionCard
        id="sec-danger"
        danger
        icon={<AlertTriangle size={15} />}
        title="Danger Zone"
        description="การกระทำต่อไปนี้ส่งผลกับทั้ง Project โปรดดำเนินการอย่างระมัดระวัง"
      >
        {/* Stash */}
        <DangerRow
          title="เก็บ Project เข้า Stash"
          help="ซ่อน Project จากรายการที่ใช้งาน — กู้คืนได้ทุกเมื่อจาก Stash ข้อมูลทั้งหมดยังอยู่ครบ"
        >
          <button
            type="button"
            disabled={isArchiving}
            onClick={() => setConfirmArchive(true)}
            className="inline-flex items-center gap-2 h-10 px-4 rounded-lg border border-slate-200 text-[13.5px] font-bold text-slate-700 hover:bg-slate-50 hover:border-slate-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isArchiving ? <Loader2 size={16} className="animate-spin" /> : <Archive size={16} />}
            Stash
          </button>
        </DangerRow>

        {/* Transfer ownership */}
        <TransferOwnershipRow boardId={boardId} />
      </SectionCard>

      <ConfirmDialog
        open={confirmArchive}
        title="เก็บ Project นี้เข้า Stash?"
        description="Project จะถูกเก็บเข้า Stash และซ่อนจากรายการที่ใช้งาน — คุณกู้คืนได้ทุกเมื่อ"
        confirmLabel="Stash"
        cancelLabel="Cancel"
        destructive
        onConfirm={() => {
          setConfirmArchive(false);
          onArchive();
        }}
        onCancel={() => setConfirmArchive(false)}
      />
    </>
  );
}
