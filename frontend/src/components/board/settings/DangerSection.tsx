// components/board/settings/DangerSection.tsx
"use client";

import { useState } from "react";
import { AlertTriangle, Archive, Repeat2, Trash2, Loader2 } from "lucide-react";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { SectionCard, MockBadge } from "./SettingsParts";

interface DangerSectionProps {
  /** Real, recoverable removal — DELETE /boards/:id (soft delete → Trash). */
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
 * "เก็บเข้าคลัง" maps to the real move-to-trash (recoverable from Trash), so it's
 * the one wired action. Transfer-ownership and permanent delete have no backend
 * — they're mockups and disabled, each flagged with a MockBadge.
 */
export function DangerSection({ onArchive, isArchiving }: DangerSectionProps) {
  const [confirmArchive, setConfirmArchive] = useState(false);

  return (
    <>
      <SectionCard
        id="sec-danger"
        danger
        icon={<AlertTriangle size={15} />}
        title="พื้นที่อันตราย"
        description="การกระทำต่อไปนี้ส่งผลถาวรกับทั้งบอร์ด โปรดดำเนินการอย่างระมัดระวัง"
      >
        {/* Archive — REAL (move to trash, recoverable) */}
        <DangerRow
          title="เก็บบอร์ดเข้าคลัง"
          help="ซ่อนบอร์ดจากรายการที่ใช้งาน — กู้คืนได้ทุกเมื่อจากถังขยะ ข้อมูลทั้งหมดยังอยู่ครบ"
        >
          <button
            type="button"
            disabled={isArchiving}
            onClick={() => setConfirmArchive(true)}
            className="inline-flex items-center gap-2 h-10 px-4 rounded-lg border border-slate-200 text-[13.5px] font-bold text-slate-700 hover:bg-slate-50 hover:border-slate-300 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isArchiving ? <Loader2 size={16} className="animate-spin" /> : <Archive size={16} />}
            เก็บเข้าคลัง
          </button>
        </DangerRow>

        {/* Transfer ownership — MOCKUP */}
        <DangerRow
          title={<>โอนความเป็นเจ้าของ <MockBadge /></>}
          help="ส่งต่อสิทธิ์ Owner ให้สมาชิกคนอื่น — คุณจะกลายเป็น Manager หลังโอนสำเร็จ"
        >
          <button
            type="button"
            disabled
            title="ยังไม่ได้ implement"
            className="inline-flex items-center gap-2 h-10 px-4 rounded-lg border border-red-200 text-[13.5px] font-bold text-red-600 opacity-50 cursor-not-allowed"
          >
            <Repeat2 size={16} />
            โอนความเป็นเจ้าของ
          </button>
        </DangerRow>

        {/* Permanent delete — MOCKUP */}
        <DangerRow
          title={<>ลบบอร์ดนี้ถาวร <MockBadge /></>}
          help={
            <>
              ลบบอร์ดและข้อมูลทั้งหมดอย่างถาวร รวมถึงการ์ด ความคิดเห็น และไฟล์แนบ —{" "}
              <b>ไม่สามารถกู้คืนได้</b>
            </>
          }
        >
          <button
            type="button"
            disabled
            title="ยังไม่ได้ implement"
            className="inline-flex items-center gap-2 h-10 px-4 rounded-lg bg-red-600 text-white text-[13.5px] font-bold opacity-50 cursor-not-allowed"
          >
            <Trash2 size={16} />
            ลบบอร์ด
          </button>
        </DangerRow>
      </SectionCard>

      <ConfirmDialog
        open={confirmArchive}
        title="เก็บบอร์ดนี้เข้าคลัง?"
        description="บอร์ดจะถูกย้ายไปถังขยะและซ่อนจากรายการที่ใช้งาน — คุณกู้คืนได้ทุกเมื่อ"
        confirmLabel="เก็บเข้าคลัง"
        cancelLabel="ยกเลิก"
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
