// components/board/settings/DangerSection.tsx
"use client";

import { useState } from "react";
import { AlertTriangle, Archive, Loader2 } from "lucide-react";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { SectionCard } from "./SettingsParts";

interface DangerSectionProps {
  /** Real, recoverable removal — DELETE /boards/:id (soft delete → คลังบอร์ด/stash). */
  onArchive: () => void;
  isArchiving: boolean;
}

function DangerRow({
  title,
  help,
  children,
}: {
  title: React.ReactNode;
  help: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-5 px-5 py-4 border-t border-slate-100 first:border-t-0">
      <div className="flex-1 min-w-0">
        <div className="text-sm font-bold text-slate-900 flex items-center gap-2 flex-wrap">
          {title}
        </div>
        <p className="text-[12.5px] text-slate-500 mt-1 leading-relaxed">{help}</p>
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

/**
 * "เก็บเข้าคลัง" maps to the real stash (recoverable from คลังบอร์ด). It's the
 * only board-level destructive action we expose — transfer-ownership and
 * permanent delete were dropped as out of scope for now.
 */
export function DangerSection({ onArchive, isArchiving }: DangerSectionProps) {
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
        {/* Stash — REAL (recoverable soft-delete → คลังบอร์ด) */}
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
