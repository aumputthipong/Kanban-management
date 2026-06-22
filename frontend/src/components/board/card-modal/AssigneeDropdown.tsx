"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Check, ChevronDown } from "lucide-react";
import type { BoardMember } from "@/types/board";
import { getAvatarColor } from "@/utils/avatar";
import { useBoardStore } from "@/store/useBoardStore";

interface AssigneeDropdownProps {
  members: BoardMember[];
  /** Selected user id; "" = Unassigned. */
  value: string;
  /** Fires with the chosen user id ("" to unassign). Caller commits. */
  onSelect: (userId: string) => void;
}

/** First grapheme of a name, emoji-safe (charAt would split surrogate pairs —
 *  names can lead with an emoji per the team's naming habits). */
function initial(name: string): string {
  return (Array.from(name.trim())[0] ?? "?").toUpperCase();
}

function Avatar({ userId, name, size }: { userId: string; name: string; size: 5 | 6 }) {
  const dim = size === 6 ? "w-6 h-6" : "w-5 h-5";
  return (
    <div
      className={`${dim} rounded-full flex items-center justify-center text-white text-[10px] font-bold shrink-0 ${getAvatarColor(userId)}`}
    >
      {initial(name)}
    </div>
  );
}

// The right rail is a fixed 224px column, so a native <select> truncates long
// Thai names + email. This dropdown keeps a rail-width trigger but opens a
// wider portal panel (right-aligned to the trigger, growing leftward) where
// each member shows avatar + full name + email without cramping.
const PANEL_W = 288;

export function AssigneeDropdown({ members, value, onSelect }: AssigneeDropdownProps) {
  const [open, setOpen] = useState(false);
  const [menuStyle, setMenuStyle] = useState<React.CSSProperties>({});
  const btnRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // "Me" gets pinned to the top + a badge so users find themselves first when
  // self-assigning. currentUserId is hydrated on board load (useBoardData).
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const me = members.find((m) => m.user_id === currentUserId);
  const others = members.filter((m) => m.user_id !== currentUserId);

  const current = members.find((m) => m.user_id === value);

  useEffect(() => {
    if (!open || !btnRef.current) return;
    const r = btnRef.current.getBoundingClientRect();
    setMenuStyle({
      position: "fixed",
      top: r.bottom + 4,
      left: Math.max(8, r.right - PANEL_W),
      width: PANEL_W,
      maxWidth: "calc(100vw - 16px)",
      zIndex: 99999,
    });
  }, [open]);

  useEffect(() => {
    const onClickOutside = (e: MouseEvent) => {
      if (
        menuRef.current?.contains(e.target as Node) ||
        btnRef.current?.contains(e.target as Node)
      )
        return;
      setOpen(false);
    };
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
  }, []);

  const choose = (id: string) => {
    onSelect(id);
    setOpen(false);
  };

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center gap-2 text-sm border border-slate-200 rounded-lg px-3 py-2 bg-white hover:border-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-400 transition-colors"
      >
        {current ? (
          <>
            <Avatar userId={current.user_id} name={current.full_name} size={6} />
            <span className="truncate text-slate-700">{current.full_name}</span>
            {current.user_id === currentUserId && <MeBadge />}
          </>
        ) : (
          <>
            <span className="w-6 h-6 rounded-full border border-dashed border-slate-300 shrink-0" />
            <span className="truncate text-slate-400">Unassigned</span>
          </>
        )}
        <ChevronDown size={14} className="ml-auto shrink-0 text-slate-400" />
      </button>

      {open &&
        createPortal(
          <div
            ref={menuRef}
            style={menuStyle}
            className="bg-white border border-slate-200 rounded-lg shadow-xl overflow-y-auto max-h-72 py-1"
          >
            {/* Me first, then the clear option, then everyone else. */}
            {me && (
              <MemberRow m={me} isMe selected={value === me.user_id} onSelect={choose} />
            )}

            <Row active={value === ""} onClick={() => choose("")}>
              <span className="w-6 h-6 rounded-full border border-dashed border-slate-300 shrink-0" />
              <span className="text-slate-500">Unassigned</span>
            </Row>

            {others.map((m) => (
              <MemberRow
                key={m.user_id}
                m={m}
                isMe={false}
                selected={value === m.user_id}
                onSelect={choose}
              />
            ))}
          </div>,
          document.body,
        )}
    </>
  );
}

function Row({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 transition-colors ${
        active ? "bg-blue-50" : "hover:bg-slate-50"
      }`}
    >
      {children}
    </button>
  );
}

function MemberRow({
  m,
  isMe,
  selected,
  onSelect,
}: {
  m: BoardMember;
  isMe: boolean;
  selected: boolean;
  onSelect: (userId: string) => void;
}) {
  return (
    <Row active={selected} onClick={() => onSelect(m.user_id)}>
      <Avatar userId={m.user_id} name={m.full_name} size={6} />
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5">
          <span className="truncate text-slate-700 font-medium">{m.full_name}</span>
          {isMe && <MeBadge />}
        </span>
        <span className="block truncate text-[11px] text-slate-400">{m.email}</span>
      </span>
      {selected && <Check size={14} className="shrink-0 text-primary" />}
    </Row>
  );
}

/** "ฉัน" marker for the current user — a chip, not prose, so the accent read
 *  is allowed; kept distinct from the blue selected-row wash by the stronger
 *  blue-100 fill. */
function MeBadge() {
  return (
    <span className="shrink-0 text-[10px] font-semibold leading-none px-1.5 py-0.5 rounded bg-blue-100 text-blue-700">
      ฉัน
    </span>
  );
}
