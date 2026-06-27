"use client";

import { Inbox } from "lucide-react";
import type { MyWorkFilter } from "@/types/myWork";

const COPY: Record<MyWorkFilter, { title: string; body: string }> = {
  all: {
    title: "ไม่มีงานค้างอยู่",
    body: "งานที่ถูก assign ให้คุณจากทุก board จะมาโผล่ที่นี่",
  },
  overdue: {
    title: "ไม่มีงานเลยกำหนด",
    body: "ทำงานทันเวลาดีมาก",
  },
  today: {
    title: "ไม่มีงานวันนี้",
    body: "ลองดูงานสัปดาห์นี้หรือเพิ่ม due date ให้งานในรายการ",
  },
  this_week: {
    title: "ไม่มีงานในสัปดาห์นี้",
    body: "ลองเช็คงานเลยกำหนดหรือ filter ทั้งหมด",
  },
  no_date: {
    title: "ทุกงานมีกำหนดส่งแล้ว",
    body: "ไม่มีงานที่ค้างไว้โดยไม่มีวันที่",
  },
};

export function MyWorkEmptyState({ filter }: { filter: MyWorkFilter }) {
  const copy = COPY[filter];
  return (
    <div className="text-center py-16 px-8 border border-dashed border-slate-200 rounded-xl bg-white">
      <div className="inline-flex items-center justify-center w-10 h-10 rounded-full bg-slate-100 text-slate-400 mb-3">
        <Inbox size={18} />
      </div>
      <p className="text-sm font-semibold text-slate-700">{copy.title}</p>
      <p className="text-xs text-slate-400 mt-1">{copy.body}</p>
    </div>
  );
}
