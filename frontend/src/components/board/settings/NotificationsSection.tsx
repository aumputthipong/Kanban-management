// components/board/settings/NotificationsSection.tsx
"use client";

import { useState } from "react";
import { Bell } from "lucide-react";
import { SectionCard, SettingRow, Toggle } from "./SettingsParts";

const PREFS = [
  {
    key: "assigned",
    label: "มีคนมอบหมายงานให้ฉัน",
    help: "แจ้งเตือนทันทีเมื่อมีการ์ดถูกมอบหมายมาที่คุณ",
    on: true,
  },
  {
    key: "mention",
    label: "มีคนพูดถึงฉัน (@mention)",
    help: "แจ้งเตือนเมื่อมีคนแท็กคุณในความคิดเห็น",
    on: true,
  },
  {
    key: "status",
    label: "การ์ดของฉันเปลี่ยนสถานะ",
    help: "แจ้งเตือนเมื่อมีคนย้ายหรืออัปเดตการ์ดที่คุณดูแล",
    on: false,
  },
  {
    key: "digest",
    label: "สรุปความคืบหน้ารายวันทางอีเมล",
    help: "อีเมลสรุปงานค้างและกิจกรรมในบอร์ดทุกเช้าเวลา 9:00 น.",
    on: false,
  },
] as const;

/** Entire section is a mockup — per-user notification prefs have no backend. */
export function NotificationsSection() {
  const [state, setState] = useState<Record<string, boolean>>(
    () => Object.fromEntries(PREFS.map((p) => [p.key, p.on]))
  );

  return (
    <SectionCard
      id="sec-notify"
      mock
      icon={<Bell size={15} />}
      title="การแจ้งเตือน"
      description="เลือกเหตุการณ์ในบอร์ดนี้ที่คุณอยากได้รับแจ้ง — ตั้งค่าเฉพาะของคุณเอง"
    >
      {PREFS.map((p) => (
        <SettingRow
          key={p.key}
          label={p.label}
          help={p.help}
          control={
            <Toggle
              checked={state[p.key]}
              onChange={(next) => setState((s) => ({ ...s, [p.key]: next }))}
            />
          }
        />
      ))}
    </SectionCard>
  );
}
