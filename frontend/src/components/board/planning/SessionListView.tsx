"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ArrowRight,
  CheckCircle2,
  ChevronRight,
  FileText,
  HelpCircle,
  Plus,
} from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
import { useToastStore } from "@/store/useToastStore";
import { planningApi } from "@/lib/planningApi";
import type { PlanningSessionSummary } from "@/types/planning";
import { formatRelativeFromNow } from "./planningFormat";

interface Props {
  boardId: string;
}

// Sessions list — chronological with a "this week / this month / older"
// rough grouping. Counts read live from each row so they stay accurate
// after the user moves items in/out of dropped or promoted.
export function SessionListView({ boardId }: Props) {
  const router = useRouter();
  const [sessions, setSessions] = useState<PlanningSessionSummary[] | null>(null);
  const [creating, setCreating] = useState(false);
  const showToast = useToastStore((s) => s.show);

  useEffect(() => {
    let cancelled = false;
    planningApi
      .listSessions(boardId)
      .then((rows) => {
        if (!cancelled) setSessions(rows);
      })
      .catch(() => {
        if (!cancelled) setSessions([]);
      });
    return () => {
      cancelled = true;
    };
  }, [boardId]);

  const createSession = useCallback(async () => {
    if (creating) return;
    setCreating(true);
    try {
      const sess = await planningApi.createSession(boardId, {
        title: defaultSessionTitle(),
      });
      router.push(`/board/${boardId}/planning/${sess.id}`);
    } catch {
      showToast({ message: "สร้างบันทึกใหม่ไม่ได้", duration: 4000 });
      setCreating(false);
    }
  }, [boardId, creating, router, showToast]);

  if (sessions === null) {
    return <ListSkeleton />;
  }

  // Aggregate stats across all sessions for the header line.
  const openQuestions = sessions.reduce((sum, s) => sum + s.q_count, 0);
  const promoted = sessions.reduce((sum, s) => sum + s.promoted_count, 0);

  const grouped = groupSessions(sessions);

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <FileText size={20} className="text-indigo-600" />
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              Planning
            </h1>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            บันทึกไอเดียที่คุย แล้วเลือกไปทำต่อบนบอร์ดได้ทันที
          </p>
        </div>
        <button
          type="button"
          onClick={createSession}
          disabled={creating}
          className="inline-flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          <Plus size={14} />
          บันทึกใหม่
        </button>
      </div>

      {sessions.length === 0 ? (
        <EmptyState onCreate={createSession} creating={creating} />
      ) : (
        <>
          <div className="mb-5 flex flex-wrap items-center gap-2">
            <StatPill icon={<FileText size={14} />}>
              <strong className="font-bold text-slate-800">
                {sessions.length}
              </strong>{" "}
              บันทึก
            </StatPill>
            <StatPill
              icon={<HelpCircle size={14} />}
              variant={openQuestions > 0 ? "alert" : "neutral"}
            >
              <strong className="font-bold">{openQuestions}</strong> คำถามค้าง ·
              รอตอบ
            </StatPill>
            <StatPill icon={<ArrowRight size={14} />} variant="ok">
              <strong className="font-bold">{promoted}</strong> ส่งเข้า Board แล้ว
            </StatPill>
          </div>

          {grouped.map((group) => (
            <div key={group.label} className="mb-5">
              <p className="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-400">
                {group.label}
              </p>
              <div className="flex flex-col gap-2">
                {group.sessions.map((s) => (
                  <SessionRow key={s.id} boardId={boardId} session={s} />
                ))}
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  );
}

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
    tile: "bg-indigo-50 text-indigo-600",
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

function SessionRow({
  boardId,
  session,
}: {
  boardId: string;
  session: PlanningSessionSummary;
}) {
  const { total, pct, state } = sessionStat(session);
  const theme = STATE_THEME[state];
  const Icon = theme.Icon;

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
        <p className="truncate text-sm font-semibold text-slate-800 group-hover:text-indigo-700">
          {session.title}
        </p>
        <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-400">
          <span>
            {session.label ? `${session.label} · ` : ""}
            {formatRelativeFromNow(session.updated_at)}
          </span>
          <Tally session={session} />
        </div>
      </div>

      <div className="flex w-40 shrink-0 flex-col items-end justify-center gap-2">
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

        {total > 0 && state !== "fresh" && (
          <div className="w-full">
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
          </div>
        )}
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

function StatPill({
  icon,
  variant = "neutral",
  children,
}: {
  icon: React.ReactNode;
  variant?: "neutral" | "alert" | "ok";
  children: React.ReactNode;
}) {
  const styles = {
    neutral: "border-slate-200 bg-white text-slate-600 [&>svg]:text-slate-400",
    alert: "border-amber-200 bg-amber-50 text-amber-700 [&>svg]:text-amber-600",
    ok: "border-emerald-200 bg-emerald-50 text-emerald-700 [&>svg]:text-emerald-600",
  }[variant];
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium ${styles}`}
    >
      {icon}
      <span>{children}</span>
    </span>
  );
}

function EmptyState({ onCreate, creating }: { onCreate: () => void; creating: boolean }) {
  return (
    <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50/50 p-12 text-center">
      <FileText size={32} className="mx-auto mb-3 text-slate-400" />
      <h3 className="text-base font-semibold text-slate-800">
        ยังไม่มีบันทึก
      </h3>
      <p className="mt-1 mb-4 text-sm text-slate-500">
        เริ่มจดไอเดียแรก · สิ่งที่ตกลงกัน · คำถามที่ค้างใจ
        <br />
        แล้วเลือกบางอันส่งเข้า Board ตอนพร้อม
      </p>
      <button
        type="button"
        onClick={onCreate}
        disabled={creating}
        className="inline-flex items-center gap-1.5 rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
      >
        <Plus size={14} />
        บันทึกใหม่
      </button>
    </div>
  );
}

function ListSkeleton() {
  return (
    <div>
      <Skeleton className="mb-2 h-7 w-40" />
      <Skeleton className="mb-6 h-4 w-72" />
      <div className="flex flex-col gap-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-16 w-full rounded-xl" />
        ))}
      </div>
    </div>
  );
}

function defaultSessionTitle() {
  const d = new Date();
  return `บันทึก ${d.toLocaleDateString("th-TH", {
    day: "numeric",
    month: "short",
    year: "numeric",
  })}`;
}

// Group by "this week / earlier this month / older". Sessions list is small
// (handful per project) so this naive scan beats sorting once + bucketing.
function groupSessions(sessions: PlanningSessionSummary[]) {
  const now = new Date();
  const weekAgo = new Date(now);
  weekAgo.setDate(now.getDate() - 7);
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

  const buckets: Record<string, PlanningSessionSummary[]> = {
    สัปดาห์นี้: [],
    เดือนนี้: [],
    ก่อนหน้า: [],
  };

  for (const s of sessions) {
    const when = new Date(s.meeting_at ?? s.updated_at);
    if (when >= weekAgo) buckets["สัปดาห์นี้"].push(s);
    else if (when >= monthStart) buckets["เดือนนี้"].push(s);
    else buckets["ก่อนหน้า"].push(s);
  }

  return Object.entries(buckets)
    .filter(([, v]) => v.length > 0)
    .map(([label, ss]) => ({ label, sessions: ss }));
}
