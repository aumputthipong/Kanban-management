"use client";

import { useEffect, useState } from "react";
import { Link2, Copy, Check, RefreshCw, Loader2 } from "lucide-react";
import { inviteApi, type InviteLink } from "@/lib/inviteApi";
import { useToastStore } from "@/store/useToastStore";

// Discord-style invite link: when a manager opens this, the link is already
// there (auto-generated if the board has none) — no separate "create" step. The
// only prominent action is Copy; "สร้างลิงก์ใหม่" quietly regenerates (which
// invalidates the old link server-side), covering the "a link leaked" case
// without a dedicated revoke button.
export function InviteLinkSection({ boardId }: { boardId: string }) {
  const showToast = useToastStore((s) => s.show);
  const [link, setLink] = useState<InviteLink | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const existing = await inviteApi.getActive(boardId);
        const l = existing ?? (await inviteApi.create(boardId));
        if (!cancelled) setLink(l);
      } catch {
        if (!cancelled) setFailed(true);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [boardId]);

  const fullUrl =
    link && typeof window !== "undefined"
      ? `${window.location.origin}/invite/${link.token}`
      : "";

  const regenerate = async () => {
    setBusy(true);
    setFailed(false);
    try {
      setLink(await inviteApi.create(boardId));
    } catch {
      showToast({ message: "สร้างลิงก์ใหม่ไม่สำเร็จ", duration: 4000 });
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(fullUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      showToast({ message: "คัดลอกไม่สำเร็จ", duration: 3000 });
    }
  };

  const expiryLabel = link
    ? new Date(link.expires_at).toLocaleDateString("th-TH", {
        day: "numeric",
        month: "short",
        year: "numeric",
      })
    : "";

  return (
    <div className="p-3">
      <div className="mb-2 flex items-center gap-2">
        <Link2 size={16} className="shrink-0 text-slate-400" />
        <div>
          <p className="text-[13px] font-bold text-slate-800">ลิงก์เชิญ</p>
          <p className="text-[11.5px] text-slate-400">
            ส่งลิงก์นี้ให้เพื่อน — เปิดแล้วล็อกอินจะเข้าร่วมเป็นสมาชิกได้เลย
          </p>
        </div>
      </div>

      {loading ? (
        <div className="h-10 animate-pulse rounded-md bg-slate-100" />
      ) : failed ? (
        <button
          type="button"
          onClick={regenerate}
          disabled={busy}
          className="inline-flex h-10 items-center gap-2 text-sm font-semibold text-slate-500 hover:text-indigo-600 disabled:opacity-50"
        >
          {busy ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={15} />}
          โหลดลิงก์ไม่สำเร็จ · ลองใหม่
        </button>
      ) : (
        <div className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <input
              readOnly
              value={fullUrl}
              onFocus={(e) => e.currentTarget.select()}
              className="h-10 min-w-0 flex-1 rounded-md border border-slate-200 bg-slate-50/70 px-3 text-sm text-slate-600 outline-none"
            />
            <button
              type="button"
              onClick={copy}
              className="inline-flex h-10 shrink-0 items-center gap-1.5 rounded-md bg-blue-800 px-3.5 text-[13.5px] font-bold text-white transition hover:bg-blue-900"
            >
              {copied ? <Check size={16} /> : <Copy size={16} />}
              {copied ? "คัดลอกแล้ว" : "คัดลอก"}
            </button>
          </div>
          <p className="flex flex-wrap items-center gap-x-2 text-[11.5px] text-slate-400">
            <span>ลิงก์หมดอายุ {expiryLabel}</span>
            <span aria-hidden>·</span>
            <button
              type="button"
              onClick={regenerate}
              disabled={busy}
              className="inline-flex items-center gap-1 font-semibold text-slate-500 hover:text-indigo-600 disabled:opacity-50"
            >
              {busy ? <Loader2 size={11} className="animate-spin" /> : <RefreshCw size={11} />}
              สร้างลิงก์ใหม่
            </button>
          </p>
        </div>
      )}
    </div>
  );
}
