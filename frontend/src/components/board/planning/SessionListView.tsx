"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, FileText, HelpCircle, Plus } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
import { useToastStore } from "@/store/useToastStore";
import { planningApi } from "@/lib/planningApi";
import type { PlanningSessionSummary } from "@/types/planning";
import { SessionRow } from "./SessionRow";

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

  // Inline rename — optimistic, reverts the single row's title on failure.
  // Owned here because this component holds the sessions list state.
  const handleRename = async (id: string, title: string) => {
    const prevTitle = sessions?.find((s) => s.id === id)?.title ?? "";
    if (title === prevTitle) return;
    setSessions((cur) =>
      cur ? cur.map((s) => (s.id === id ? { ...s, title } : s)) : cur,
    );
    try {
      await planningApi.updateSession(id, { title });
    } catch {
      showToast({ message: "เปลี่ยนชื่อบันทึกไม่สำเร็จ", duration: 4000 });
      setSessions((cur) =>
        cur ? cur.map((s) => (s.id === id ? { ...s, title: prevTitle } : s)) : cur,
      );
    }
  };

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
            <FileText size={20} className="text-primary" />
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              Planning
            </h1>
          </div>
          <p className="mt-1 text-sm text-slate-500">
            บันทึกไอเดียที่คุย แล้วเลือกไปทำต่อบน Board ได้ทันที
          </p>
        </div>
        <button
          type="button"
          onClick={createSession}
          disabled={creating}
          className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-2 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-50"
        >
          <Plus size={14} />
          New Note
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
              Notes
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
                  <SessionRow
                    key={s.id}
                    boardId={boardId}
                    session={s}
                    onRename={handleRename}
                  />
                ))}
              </div>
            </div>
          ))}
        </>
      )}
    </div>
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
        No notes yet
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
        className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-50"
      >
        <Plus size={14} />
        New Note
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

// "Note <date>" — SessionRow.isAutoTitle treats this prefix as "not yet
// named", so keep the two in sync if the format changes.
function defaultSessionTitle() {
  const d = new Date();
  return `Note ${d.toLocaleDateString("en-GB", {
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
