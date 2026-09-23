import Link from "next/link";
import { Clock, Sparkles } from "lucide-react";

/** "อีกไม่ถึงชั่วโมง" once under an hour — a "0 ชั่วโมง" countdown reads like it already expired. */
function remainingLabel(expiresAt: string): string | null {
  const msLeft = new Date(expiresAt).getTime() - Date.now();
  if (Number.isNaN(msLeft) || msLeft <= 0) return null;
  const hours = Math.floor(msLeft / 3_600_000);
  return hours < 1 ? "อีกไม่ถึงชั่วโมง" : `อีก ${hours} ชั่วโมง`;
}

/**
 * Shown only inside a demo session. A visitor dropped on a board with no context does
 * not know which parts are worth touching, so the bar names them — and states the two
 * limits that surprise people later: the sandbox expires, and signing up starts empty.
 */
export function DemoBanner({ expiresAt }: { expiresAt?: string | null }) {
  const remaining = expiresAt ? remainingLabel(expiresAt) : null;

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
      <span className="flex items-center gap-1.5 text-slate-500">
        <Clock size={13} />
        {remaining
          ? `บอร์ดทดลองจะถูกลบใน${remaining} และงานที่ทำไว้จะไม่ย้ายไปบัญชีใหม่`
          : "บอร์ดทดลองจะถูกลบอัตโนมัติ และงานที่ทำไว้จะไม่ย้ายไปบัญชีใหม่"}
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
