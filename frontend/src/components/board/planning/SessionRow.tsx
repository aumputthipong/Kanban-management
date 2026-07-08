"use client";

import { useRef, useState } from "react";
import Link from "next/link";
import {
  CheckCircle2,
  ChevronRight,
  FileText,
  HelpCircle,
  Pencil,
} from "lucide-react";
import type { PlanningSessionSummary } from "@/types/planning";
import { formatRelativeFromNow } from "./planningFormat";

// Each row answers, at a glance: what state is this session in (open
// questions / all sent / fresh / in-play), what's inside (type tally), and
// how much already went to the Board (progress). All derived from the
// summary counts — no extra fetch.
type SessionState = "followup" | "done" | "fresh" | "active";

const STATE_THEME: Record<
  SessionState,
  { accent: string; tile: string; Icon: typeof FileText }
> = {
  followup: {
    accent: "bg-amber-400",
    tile: "bg-amber-50 text-amber-600",
    Icon: HelpCircle,
  },
  active: {
    accent: "bg-indigo-400",
    tile: "bg-indigo-50 text-primary",
    Icon: FileText,
  },
  done: {
    accent: "bg-emerald-400",
    tile: "bg-emerald-50 text-emerald-600",
    Icon: CheckCircle2,
  },
  fresh: {
    accent: "bg-slate-300",
    tile: "bg-slate-100 text-slate-400",
    Icon: FileText,
  },
};

function sessionStat(s: PlanningSessionSummary) {
  const live = s.req_count + s.dec_count + s.q_count;
  // "to send" universe excludes the paused (dropped) pile — those have their
  // own bucket and aren't pending promotion.
  const total = live + s.promoted_count;
  const pct = total > 0 ? Math.round((s.promoted_count / total) * 100) : 0;
  let state: SessionState;
  if (s.q_count > 0) state = "followup";
  else if (s.promoted_count > 0 && live === 0) state = "done";
  else if (total <= 1 && s.promoted_count === 0) state = "fresh";
  else state = "active";
  return { live, total, pct, state };
}

// A session is "not yet named" while it still carries the date-based title
// minted by defaultSessionTitle ("Note <date>") — the row uses this to nudge
// the user to give it a real name instead of just showing a quiet pencil.
// The legacy Thai note-title prefix is still matched so notes created
// before the rename keep behaving as auto-titled.
function isAutoTitle(title: string) {
  return /^(Note|บันทึก)\s+\d/.test(title.trim());
}

export function SessionRow({
  boardId,
  session,
  onRename,
}: {
  boardId: string;
  session: PlanningSessionSummary;
  onRename: (id: string, title: string) => void;
}) {
  const { total, pct, state } = sessionStat(session);
  const theme = STATE_THEME[state];
  const Icon = theme.Icon;
  const isAuto = isAutoTitle(session.title);

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(session.title);
  // Escape sets this so the unmount-triggered onBlur doesn't also commit.
  const escapingRef = useRef(false);

  const startEdit = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDraft(session.title);
    setEditing(true);
  };
  const commit = () => {
    if (escapingRef.current) {
      escapingRef.current = false;
      return;
    }
    setEditing(false);
    const t = draft.trim();
    if (t && t !== session.title) onRename(session.id, t);
  };

  return (
    <Link
      href={`/board/${boardId}/planning/${session.id}`}
      className="group relative flex items-stretch gap-3.5 rounded-xl border border-slate-200 bg-white py-4 pl-5 pr-3 transition-colors hover:border-indigo-300 hover:bg-indigo-50/30"
    >
      <span
        className={`absolute left-0 top-3 bottom-3 w-[3px] rounded-full ${theme.accent}`}
      />
      <span
        className={`mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${theme.tile}`}
      >
        <Icon size={18} />
      </span>

      <div className="min-w-0 flex-1">
        {editing ? (
          <input
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
            }}
            onKeyDown={(e) => {
              e.stopPropagation();
              if (e.key === "Enter") {
                e.preventDefault();
                commit();
              } else if (e.key === "Escape") {
                e.preventDefault();
                escapingRef.current = true;
                setDraft(session.title);
                setEditing(false);
              }
            }}
            onBlur={commit}
            placeholder="Name this note…"
            className="w-full max-w-md rounded-md border border-indigo-300 bg-white px-2 py-1 text-sm font-semibold text-slate-800 shadow-sm outline-none focus:ring-2 focus:ring-indigo-400"
          />
        ) : (
          <div className="flex items-center gap-1.5">
            <p
              className={`truncate text-sm ${
                isAuto
                  ? "font-medium italic text-slate-500"
                  : "font-semibold text-slate-800 group-hover:text-primary-hover"
              }`}
            >
              {session.title}
            </p>
            {/* Rename affordance — always shown as "ตั้งชื่อ" while the session
                still carries its auto date-title; a quiet hover pencil once it
                has a real name. */}
            <button
              type="button"
              onClick={startEdit}
              title="Name this note"
              className={`inline-flex shrink-0 items-center gap-1 rounded px-1 py-0.5 text-[11px] font-semibold text-indigo-500 transition-opacity hover:text-primary-hover ${
                isAuto ? "" : "opacity-0 group-hover:opacity-100"
              }`}
            >
              <Pencil size={12} />
              {isAuto && <span>Name</span>}
            </button>
          </div>
        )}
        <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-400">
          <span>
            {session.label ? `${session.label} · ` : ""}
            {formatRelativeFromNow(session.updated_at)}
          </span>
          <Tally session={session} />
        </div>
      </div>

      {/* Fixed-height status + progress slots so every row's right column is the
          same height — rows without a progress bar no longer come up short and
          the list reads as one even stack. */}
      <div className="flex w-40 shrink-0 flex-col items-end justify-center gap-2">
        <div className="flex h-6 items-center">
          {state === "followup" ? (
            <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-[11px] font-bold text-amber-700">
              <HelpCircle size={13} />
              {session.q_count} คำถามค้าง
            </span>
          ) : state === "done" ? (
            <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-[11px] font-bold text-emerald-700">
              <CheckCircle2 size={13} />
              ส่งครบแล้ว
            </span>
          ) : state === "fresh" ? (
            <span className="text-[11px] font-semibold text-slate-400">
              ยังไม่ได้คัดส่ง
            </span>
          ) : null}
        </div>

        <div className="h-7 w-full">
          {total > 0 && state !== "fresh" && (
            <>
              <div className="mb-1 text-right text-[11px] font-semibold text-slate-500">
                <strong className="font-bold text-slate-800">
                  {session.promoted_count}
                </strong>
                /{total} ส่งเข้า Board
              </div>
              <div className="h-1 w-full overflow-hidden rounded-full bg-slate-200">
                <div
                  className={`h-full rounded-full ${
                    pct === 100 ? "bg-emerald-500" : "bg-indigo-500"
                  }`}
                  style={{ width: `${pct}%` }}
                />
              </div>
            </>
          )}
        </div>
      </div>

      <ChevronRight
        size={16}
        className="self-center text-slate-300 group-hover:text-indigo-400"
      />
    </Link>
  );
}

// Type tally — coloured dots + counts for the live REQ/DEC/Q items, matching
// the chip palette used inside a session.
function Tally({ session }: { session: PlanningSessionSummary }) {
  const parts: [string, number, string][] = [
    ["REQ", session.req_count, "bg-red-500"],
    ["DEC", session.dec_count, "bg-blue-500"],
    ["Q", session.q_count, "bg-amber-500"],
  ];
  const shown = parts.filter(([, c]) => c > 0);
  if (shown.length === 0) return null;
  return (
    <span className="flex items-center gap-2.5">
      <span className="text-slate-300">·</span>
      {shown.map(([label, count, dot]) => (
        <span
          key={label}
          className="inline-flex items-center gap-1 font-semibold text-slate-500 tabular-nums"
        >
          <span className={`h-1.5 w-1.5 rounded-sm ${dot}`} />
          {count} {label}
        </span>
      ))}
    </span>
  );
}
