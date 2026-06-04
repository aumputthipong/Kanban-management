// components/board/settings/SettingsRail.tsx
"use client";

import { Clock, Lock, Columns3, Bell, AlertTriangle } from "lucide-react";

export interface RailEntry {
  id: string;
  label: string;
  icon: React.ReactNode;
  danger?: boolean;
}

export const RAIL_ITEMS: RailEntry[] = [
  { id: "sec-general", label: "ทั่วไป", icon: <Clock size={18} /> },
  { id: "sec-access", label: "การเข้าถึง", icon: <Lock size={18} /> },
  { id: "sec-workflow", label: "เวิร์กโฟลว์", icon: <Columns3 size={18} /> },
  { id: "sec-notify", label: "การแจ้งเตือน", icon: <Bell size={18} /> },
  { id: "sec-danger", label: "พื้นที่อันตราย", icon: <AlertTriangle size={18} />, danger: true },
];

interface SettingsRailProps {
  active: string;
  items: RailEntry[];
  onJump: (id: string) => void;
}

/** Sticky section navigator. `active` is driven by scroll-spy in the parent. */
export function SettingsRail({ active, items, onJump }: SettingsRailProps) {
  return (
    <nav className="sticky top-0 flex flex-col gap-0.5">
      {items.map((item) => {
        const isActive = active === item.id;
        const dangerActive = item.danger && isActive;
        return (
          <div key={item.id}>
            {item.danger && <div className="h-px bg-slate-100 my-2.5 mx-2" />}
            <button
              type="button"
              onClick={() => onJump(item.id)}
              className={`w-full flex items-center gap-2.5 h-10 px-3.5 rounded-lg text-sm font-semibold text-left transition-colors ${
                dangerActive
                  ? "bg-red-50 text-red-600"
                  : isActive
                    ? "bg-indigo-50 text-blue-800"
                    : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              <span
                className={`shrink-0 ${
                  dangerActive
                    ? "text-red-600"
                    : isActive
                      ? "text-blue-800"
                      : "text-slate-400"
                }`}
              >
                {item.icon}
              </span>
              {item.label}
            </button>
          </div>
        );
      })}
    </nav>
  );
}
