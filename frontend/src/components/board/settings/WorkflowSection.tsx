// components/board/settings/WorkflowSection.tsx
"use client";

import { useState } from "react";
import { Columns3 } from "lucide-react";
import {
  SectionCard,
  SettingRow,
  Segmented,
  SelectField,
  Toggle,
} from "./SettingsParts";

/**
 * Entire section is a mockup — none of these behaviours have a backend yet.
 * The section header carries a single MockBadge to flag the whole card.
 */
export function WorkflowSection({ canManage }: { canManage: boolean }) {
  const [view, setView] = useState<"overview" | "board" | "planning" | "calendar">("board");
  const [autoArchive, setAutoArchive] = useState(false);
  const [archiveDays, setArchiveDays] = useState("หลัง 14 วัน");
  const [wipLimit, setWipLimit] = useState(true);
  const [requireDue, setRequireDue] = useState(false);

  return (
    <SectionCard
      id="sec-workflow"
      mock
      icon={<Columns3 size={15} />}
      title="เวิร์กโฟลว์"
      description="พฤติกรรมเริ่มต้นของบอร์ดและการ์ดงาน"
    >
      <SettingRow
        label="มุมมองเริ่มต้นเมื่อเปิดบอร์ด"
        help="หน้าที่จะแสดงเป็นอันดับแรกทุกครั้งที่สมาชิกเปิดบอร์ดนี้"
        control={
          <Segmented
            value={view}
            disabled={!canManage}
            onChange={setView}
            options={[
              { value: "overview", label: "Overview" },
              { value: "board", label: "Board" },
              { value: "planning", label: "Planning" },
              { value: "calendar", label: "Calendar" },
            ]}
          />
        }
      />

      <SettingRow
        label="เก็บการ์ดที่เสร็จแล้วอัตโนมัติ"
        help="ย้ายการ์ดในคอลัมน์ Done เข้าคลังหลังจากผ่านไประยะหนึ่ง เพื่อให้บอร์ดสะอาดอยู่เสมอ"
        control={
          <div className="flex items-center gap-3">
            {autoArchive && (
              <SelectField
                value={archiveDays}
                disabled={!canManage}
                onChange={setArchiveDays}
                options={["หลัง 7 วัน", "หลัง 14 วัน", "หลัง 30 วัน"]}
              />
            )}
            <Toggle checked={autoArchive} disabled={!canManage} onChange={setAutoArchive} />
          </div>
        }
      />

      <SettingRow
        label="จำกัดจำนวนงานต่อคอลัมน์ (WIP limit)"
        help="เตือนเมื่อคอลัมน์มีการ์ดมากเกินกำหนด ช่วยให้ทีมโฟกัสงานที่กำลังทำ"
        control={<Toggle checked={wipLimit} disabled={!canManage} onChange={setWipLimit} />}
      />

      <SettingRow
        label="บังคับกำหนดวันครบกำหนด"
        help="ผู้ใช้ต้องระบุวันครบกำหนดก่อนสร้างการ์ดใหม่ในบอร์ดนี้"
        control={<Toggle checked={requireDue} disabled={!canManage} onChange={setRequireDue} />}
      />
    </SectionCard>
  );
}
