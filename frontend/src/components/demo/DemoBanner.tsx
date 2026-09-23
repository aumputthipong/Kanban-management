import Link from "next/link";
import { Sparkles } from "lucide-react";

/**
 * Shown only inside a demo session. A visitor dropped on a board with no context
 * does not know which parts are worth touching, so the bar names them — and keeps
 * the way out of the sandbox one click away.
 */
export function DemoBanner() {
  return (
    <div className="shrink-0 bg-surface-tint border-b border-slate-200 px-4 py-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-600">
      <span className="flex items-center gap-1.5 font-semibold text-slate-700">
        <Sparkles size={14} />
        Demo mode
      </span>
      <span>
        บอร์ดนี้เป็นของคุณคนเดียว ลากการ์ดข้ามคอลัมน์ เปิดการ์ดเพื่อดูรายละเอียด
        หรือเปิดแท็บที่สองเพื่อดูการ sync แบบเรียลไทม์ได้เลย
      </span>
      <Link
        href="/register"
        className="ml-auto shrink-0 font-semibold text-primary hover:underline"
      >
        Sign up
      </Link>
    </div>
  );
}
