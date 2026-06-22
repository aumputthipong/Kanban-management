"use client";

import { useState } from "react";
import { Link2, ChevronRight } from "lucide-react";
import { InviteLinkModal } from "./InviteLinkModal";

// A quiet row in the members panel that opens the invite-link modal on click —
// the link itself isn't shown inline, so it can't leak from a glance at the
// screen. The modal ensures a ready link, so this is purely "open → copy".
export function InviteLinkButton({ boardId }: { boardId: string }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="flex w-full items-center gap-2.5 p-3 text-left transition-colors hover:bg-slate-50"
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-indigo-50 text-primary">
          <Link2 size={16} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-bold text-slate-800">เชิญด้วยลิงก์</p>
          <p className="text-[11.5px] text-slate-400">
            แชร์ลิงก์ให้คนเข้าร่วมบอร์ดเอง
          </p>
        </div>
        <ChevronRight size={16} className="shrink-0 text-slate-300" />
      </button>

      {open && <InviteLinkModal boardId={boardId} onClose={() => setOpen(false)} />}
    </>
  );
}
