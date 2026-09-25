"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2, Play } from "lucide-react";
import { apiClient } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";
import type { DemoSession } from "@/types/demo";

type Variant = "hero" | "quiet";

const STYLES: Record<Variant, string> = {
  hero: "border border-slate-300 bg-white text-slate-700 hover:bg-slate-50 px-7 py-3.5 rounded-full",
  quiet:
    "w-full justify-center border border-slate-200 bg-white text-slate-700 hover:bg-slate-50 px-4 py-2.5 rounded-lg",
};

// Secondary style on purpose: both hosts already have their one button-primary.
export function TryDemoButton({
  variant = "hero",
  label = "Try demo",
}: {
  variant?: Variant;
  label?: string;
}) {
  const router = useRouter();
  const showToast = useToastStore((s) => s.show);
  const [isStarting, setIsStarting] = useState(false);

  const start = async () => {
    setIsStarting(true);
    try {
      const session = await apiClient<DemoSession>("/auth/demo", { method: "POST" });
      // /board/[boardId] has no page; /tasks is the board.
      router.push(`/board/${session.board_id}/tasks`);
      router.refresh();
    } catch {
      // apiClient toasts 403/5xx itself; this covers the rate limit.
      showToast({ message: "เปิดโหมดทดลองไม่สำเร็จ ลองใหม่อีกครั้งในอีกสักครู่" });
      setIsStarting(false);
    }
  };

  return (
    <button
      type="button"
      onClick={start}
      disabled={isStarting}
      className={`inline-flex items-center gap-2 font-semibold text-sm transition-colors disabled:opacity-50 ${STYLES[variant]}`}
    >
      {isStarting ? (
        <>
          <Loader2 size={16} className="animate-spin" />
          กำลังเตรียมบอร์ด...
        </>
      ) : (
        <>
          <Play size={16} />
          {label}
        </>
      )}
    </button>
  );
}
