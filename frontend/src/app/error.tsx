"use client";

import { useEffect } from "react";
import Link from "next/link";
import { AlertTriangle, RotateCcw } from "lucide-react";
import { logger } from "@/lib/logger";
import { captureException } from "@/lib/sentry";

export default function RouteError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    logger.error("[route-error]", error);
    captureException(error, { boundary: "route", digest: error.digest });
  }, [error]);

  return (
    <div className="min-h-[60vh] flex items-center justify-center p-6">
      <div className="max-w-md w-full bg-white border border-danger/20 rounded-2xl shadow-sm p-6 text-center">
        <div className="mx-auto w-12 h-12 rounded-full bg-danger/10 text-danger flex items-center justify-center mb-4">
          <AlertTriangle size={22} />
        </div>
        <h1 className="text-lg font-bold text-slate-900 mb-1">
          เกิดข้อผิดพลาดบางอย่าง
        </h1>
        <p className="text-sm text-slate-500 mb-5">
          ลองโหลดส่วนนี้ใหม่ — ถ้ายังเจอปัญหาอยู่ กลับไปหน้าแรกได้
        </p>
        {error.digest && (
          <p className="text-[11px] font-mono text-slate-400 mb-4">
            ref: {error.digest}
          </p>
        )}
        <div className="flex items-center justify-center gap-2">
          <button
            type="button"
            onClick={reset}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary hover:bg-primary-hover text-white text-sm font-semibold rounded-lg transition-colors"
          >
            <RotateCcw size={14} />
            ลองอีกครั้ง
          </button>
          <Link
            href="/"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 text-sm font-semibold rounded-lg transition-colors"
          >
            กลับหน้าแรก
          </Link>
        </div>
      </div>
    </div>
  );
}
