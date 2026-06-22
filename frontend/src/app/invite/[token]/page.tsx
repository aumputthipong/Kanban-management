"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { Loader2, Link2, AlertCircle } from "lucide-react";
import { inviteApi } from "@/lib/inviteApi";
import { ApiError } from "@/lib/apiClient";
import { useToastStore } from "@/store/useToastStore";

// Invite accept page. Opening /invite/<token> tries to join the board the token
// points at. If the visitor isn't logged in, apiClient's 401 bounce sends them
// to /login?redirect=/invite/<token>, so after auth they land back here and the
// join completes. On an invalid / expired link we show a friendly dead-end.
export default function InviteAcceptPage() {
  const params = useParams();
  const router = useRouter();
  const showToast = useToastStore((s) => s.show);
  const token = Array.isArray(params.token) ? params.token[0] : params.token;

  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) return;
    let cancelled = false;

    inviteApi
      .accept(token)
      .then((res) => {
        if (cancelled) return;
        showToast({ message: "เข้าร่วมบอร์ดแล้ว", duration: 3000 });
        router.replace(`/board/${res.board_id}/tasks`);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        // 401 → apiClient already redirected to login; let that win.
        if (err instanceof ApiError && err.status === 401) return;
        const message =
          err instanceof ApiError
            ? err.status === 410
              ? "ลิงก์เชิญนี้หมดอายุแล้ว"
              : err.message
            : "เข้าร่วมบอร์ดไม่สำเร็จ";
        setError(message);
      });

    return () => {
      cancelled = true;
    };
  }, [token, router, showToast]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-50 px-4">
      <div className="w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm">
        {error ? (
          <>
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-50 text-red-500">
              <AlertCircle size={24} />
            </div>
            <h1 className="text-base font-bold text-slate-800">เข้าร่วมบอร์ดไม่ได้</h1>
            <p className="mt-1.5 text-sm text-slate-500">{error}</p>
            <Link
              href="/dashboard"
              className="mt-5 inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary-hover"
            >
              ไปหน้าโปรเจกต์ของฉัน
            </Link>
          </>
        ) : (
          <>
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-indigo-50 text-primary">
              <Link2 size={22} />
            </div>
            <h1 className="text-base font-bold text-slate-800">กำลังเข้าร่วมบอร์ด…</h1>
            <p className="mt-1.5 inline-flex items-center gap-1.5 text-sm text-slate-500">
              <Loader2 size={14} className="animate-spin" /> โปรดรอสักครู่
            </p>
          </>
        )}
      </div>
    </div>
  );
}
