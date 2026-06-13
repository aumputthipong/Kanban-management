"use client";

import { useEffect, useState } from "react";
import { Link2, Copy, Check, RefreshCw, X, Loader2 } from "lucide-react";
import { inviteApi, type InviteLink } from "@/lib/inviteApi";
import { useToastStore } from "@/store/useToastStore";

// Manager-facing invite-link panel: shows the board's active shareable link (or
// a "create" prompt), with copy / regenerate / turn-off. Anyone who opens the
// link and is logged in joins as a member; the link expires and can be revoked.
export function InviteLinkSection({ boardId }: { boardId: string }) {
  const showToast = useToastStore((s) => s.show);
  const [link, setLink] = useState<InviteLink | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    inviteApi
      .getActive(boardId)
      .then((l) => !cancelled && setLink(l))
      .catch(() => !cancelled && setLink(null))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [boardId]);

  const fullUrl =
    link && typeof window !== "undefined"
      ? `${window.location.origin}/invite/${link.token}`
      : "";

  const generate = async () => {
    setBusy(true);
    try {
      setLink(await inviteApi.create(boardId));
    } catch {
      showToast({ message: "สร้างลิงก์เชิญไม่สำเร็จ", duration: 4000 });
    } finally {
      setBusy(false);
    }
  };

  const revoke = async () => {
    setBusy(true);
    try {
      await inviteApi.revoke(boardId);
      setLink(null);
    } catch {
      showToast({ message: "ปิดลิงก์ไม่สำเร็จ", duration: 4000 });
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
            ใครมีลิงก์นี้และล็อกอินอยู่ เข้าร่วมเป็นสมาชิกได้
          </p>
        </div>
      </div>

      {loading ? (
        <div className="h-10 animate-pulse rounded-md bg-slate-100" />
      ) : link ? (
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
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11.5px] text-slate-400">
            <span>หมดอายุ {expiryLabel}</span>
            <button
              type="button"
              onClick={generate}
              disabled={busy}
              className="inline-flex items-center gap-1 font-semibold text-slate-500 hover:text-indigo-600 disabled:opacity-50"
            >
              <RefreshCw size={12} /> สร้างลิงก์ใหม่
            </button>
            <button
              type="button"
              onClick={revoke}
              disabled={busy}
              className="inline-flex items-center gap-1 font-semibold text-slate-500 hover:text-red-600 disabled:opacity-50"
            >
              <X size={12} /> ปิดลิงก์
            </button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={generate}
          disabled={busy}
          className="inline-flex h-10 items-center gap-2 rounded-md border border-dashed border-slate-300 px-4 text-sm font-semibold text-slate-600 transition hover:border-indigo-400 hover:text-indigo-600 disabled:opacity-50"
        >
          {busy ? <Loader2 size={16} className="animate-spin" /> : <Link2 size={16} />}
          สร้างลิงก์เชิญ
        </button>
      )}
    </div>
  );
}
