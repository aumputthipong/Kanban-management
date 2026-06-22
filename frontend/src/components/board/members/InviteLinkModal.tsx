"use client";

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { Link2, Copy, Check, RefreshCw, X, Loader2 } from "lucide-react";
import { inviteApi, type InviteLink } from "@/lib/inviteApi";
import { useToastStore } from "@/store/useToastStore";
import { useEscapeKey } from "@/hooks/useEscapeKey";

// Discord-style invite modal. Opening it guarantees a ready-to-share link — it
// reuses the board's active link, or quietly mints one if there's none/it has
// expired — so the manager never has to think about the link's lifecycle: open
// → copy. The link stays hidden behind the button until intentionally opened
// (less leak on screen-share). "สร้างลิงก์ใหม่" is a quiet escape hatch to
// invalidate the old link if it ever gets out.
export function InviteLinkModal({
  boardId,
  onClose,
}: {
  boardId: string;
  onClose: () => void;
}) {
  const showToast = useToastStore((s) => s.show);
  const [link, setLink] = useState<InviteLink | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  useEscapeKey(true, onClose);

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

  return createPortal(
    <>
      <div className="fixed inset-0 z-9998 bg-slate-900/45" onClick={onClose} />
      <div className="fixed inset-0 z-9999 flex items-center justify-center px-4 pointer-events-none">
        <div
          className="pointer-events-auto w-full max-w-md rounded-2xl bg-white shadow-2xl"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between border-b border-slate-100 px-5 pt-5 pb-4">
            <div className="flex items-center gap-2.5">
              <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-50 text-primary">
                <Link2 size={16} />
              </span>
              <h2 className="text-sm font-bold text-slate-800">Invite Link</h2>
            </div>
            <button
              onClick={onClose}
              aria-label="ปิด"
              className="rounded-md p-1 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
            >
              <X size={18} />
            </button>
          </div>

          {/* Body */}
          <div className="px-5 py-5">
            <p className="mb-3 text-[13px] text-slate-500">
              ใครมีลิงก์นี้และล็อกอินอยู่ จะเข้าร่วมบอร์ดเป็นสมาชิกได้
            </p>

            {loading ? (
              <div className="h-11 animate-pulse rounded-md bg-slate-100" />
            ) : failed ? (
              <button
                type="button"
                onClick={regenerate}
                disabled={busy}
                className="inline-flex h-11 items-center gap-2 text-sm font-semibold text-slate-500 hover:text-primary disabled:opacity-50"
              >
                {busy ? <Loader2 size={16} className="animate-spin" /> : <RefreshCw size={15} />}
                โหลดลิงก์ไม่สำเร็จ · ลองใหม่
              </button>
            ) : (
              <>
                <div className="flex items-center gap-2">
                  <input
                    readOnly
                    value={fullUrl}
                    onFocus={(e) => e.currentTarget.select()}
                    className="h-11 min-w-0 flex-1 rounded-md border border-slate-200 bg-slate-50/70 px-3 text-sm text-slate-600 outline-none"
                  />
                  <button
                    type="button"
                    onClick={copy}
                    className="inline-flex h-11 shrink-0 items-center gap-1.5 rounded-md bg-blue-800 px-4 text-sm font-bold text-white transition hover:bg-blue-900"
                  >
                    {copied ? <Check size={16} /> : <Copy size={16} />}
                    {copied ? "คัดลอกแล้ว" : "คัดลอก"}
                  </button>
                </div>
                <p className="mt-3 flex flex-wrap items-center gap-x-2 text-[11.5px] text-slate-400">
                  <span>ลิงก์หมดอายุ {expiryLabel}</span>
                  <span aria-hidden>·</span>
                  <button
                    type="button"
                    onClick={regenerate}
                    disabled={busy}
                    className="inline-flex items-center gap-1 font-semibold text-slate-500 hover:text-primary disabled:opacity-50"
                  >
                    {busy ? <Loader2 size={11} className="animate-spin" /> : <RefreshCw size={11} />}
                    สร้างลิงก์ใหม่
                  </button>
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    </>,
    document.body,
  );
}
