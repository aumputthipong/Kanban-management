// components/board/settings/AccessSection.tsx
"use client";

import { useState } from "react";
import Link from "next/link";
import { Lock, Globe, ChevronRight } from "lucide-react";
import {
  SectionCard,
  SettingRow,
  Segmented,
  SelectField,
  MockBadge,
} from "./SettingsParts";
import { avatarColor, initials } from "@/components/board/overview/activityFormat";

interface AccessSectionProps {
  boardId: string;
  members: { user_id: string; full_name: string }[];
  canManage: boolean;
}

/**
 * Only the members shortcut is real: member count + avatar stack come from the
 * loaded board, and the button links to the existing /members route. Visibility,
 * invite permission, and default role have no backend — all mockups.
 */
export function AccessSection({ boardId, members, canManage }: AccessSectionProps) {
  const [visibility, setVisibility] = useState<"private" | "workspace">("private");
  const [invite, setInvite] = useState<"managers" | "everyone">("managers");
  const [defaultRole, setDefaultRole] = useState("Member");

  const shown = members.slice(0, 4);
  const extra = members.length - shown.length;

  return (
    <SectionCard
      id="sec-access"
      icon={<Lock size={15} />}
      title="การเข้าถึง"
      description="ใครเห็นบอร์ดนี้ได้ และใครมีสิทธิ์เชิญสมาชิกใหม่"
    >
      {/* Visibility — MOCKUP */}
      <SettingRow
        label={
          <>
            การมองเห็น <MockBadge />
          </>
        }
        help="บอร์ดส่วนตัวเห็นเฉพาะสมาชิกที่ถูกเชิญ — บอร์ดเวิร์กสเปซเปิดให้ทุกคนในเวิร์กสเปซเข้าร่วมได้เอง"
        control={
          <Segmented
            value={visibility}
            disabled={!canManage}
            onChange={setVisibility}
            options={[
              { value: "private", label: <><Lock size={14} /> ส่วนตัว</> },
              { value: "workspace", label: <><Globe size={14} /> ทั้งเวิร์กสเปซ</> },
            ]}
          />
        }
      />

      {/* Who can invite — MOCKUP */}
      <SettingRow
        label={
          <>
            ใครเชิญสมาชิกได้ <MockBadge />
          </>
        }
        help="กำหนดว่าใครส่งคำเชิญและจัดการสมาชิกของบอร์ดได้"
        control={
          <Segmented
            value={invite}
            disabled={!canManage}
            onChange={setInvite}
            options={[
              { value: "managers", label: "Owner & Manager" },
              { value: "everyone", label: "ทุกคน" },
            ]}
          />
        }
      />

      {/* Default role — MOCKUP */}
      <SettingRow
        label={
          <>
            สิทธิ์เริ่มต้นของสมาชิกใหม่ <MockBadge />
          </>
        }
        help="บทบาทที่กำหนดให้อัตโนมัติเมื่อมีคนเข้าร่วมบอร์ด"
        control={
          <SelectField
            value={defaultRole}
            disabled={!canManage}
            onChange={setDefaultRole}
            options={["Member", "Manager"]}
          />
        }
      />

      {/* Members shortcut — REAL */}
      <SettingRow
        label="สมาชิกในบอร์ด"
        help={`ขณะนี้มีสมาชิก ${members.length} คน`}
        control={
          <div className="flex items-center gap-3.5">
            <div className="flex items-center">
              {shown.map((m, i) => (
                <div
                  key={m.user_id}
                  className={`w-[34px] h-[34px] rounded-full flex items-center justify-center text-[13px] font-bold border-[2.5px] border-white ${avatarColor(
                    m.full_name
                  )} ${i > 0 ? "-ml-2.5" : ""}`}
                  title={m.full_name}
                >
                  {initials(m.full_name)}
                </div>
              ))}
              {extra > 0 && (
                <div className="w-[34px] h-[34px] -ml-2.5 rounded-full bg-slate-100 text-slate-600 text-xs font-bold flex items-center justify-center border-[2.5px] border-white">
                  +{extra}
                </div>
              )}
            </div>
            <Link
              href={`/board/${boardId}/members`}
              className="inline-flex items-center gap-2 h-[42px] px-4 rounded-lg border border-slate-200 text-sm font-semibold text-slate-600 hover:bg-slate-50 hover:border-slate-300 transition-colors"
            >
              จัดการสมาชิก
              <ChevronRight size={15} />
            </Link>
          </div>
        }
      />
    </SectionCard>
  );
}
