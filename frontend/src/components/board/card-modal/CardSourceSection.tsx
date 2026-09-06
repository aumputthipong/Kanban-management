"use client";

// The "source" block in the card detail modal, rendered only for cards promoted from a
// planning session. Hidden while loading and when there is no source so the layout does
// not shift; a Skeleton covers the wait.
import { memo } from "react";
import Link from "next/link";
import { ArrowUpRight, HelpCircle } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
import { useCardSource } from "@/hooks/useCardSource";
import {
  TYPE_CHIP,
  TYPE_TOOLTIP,
} from "@/components/board/planning/planningTypeMeta";
import { formatRelativeFromNow } from "@/components/board/planning/planningFormat";

interface Props {
  cardId: string;
  boardId: string;
}

function CardSourceSectionImpl({ cardId, boardId }: Props) {
  const { source, isLoading } = useCardSource(cardId);

  if (isLoading && source === undefined) {
    return (
      <div className="rounded-lg border border-slate-200 bg-slate-50/40 p-3">
        <Skeleton className="mb-2 h-3 w-16" />
        <Skeleton className="h-4 w-48" />
      </div>
    );
  }

  if (!source) return null;

  const { session, item, pending_questions } = source;
  const sessionHref = `/board/${boardId}/planning/${session.id}#item-${item.id}`;
  const meetingLabel = session.meeting_at
    ? formatRelativeFromNow(session.meeting_at)
    : null;

  return (
    <section className="rounded-lg border border-slate-200 bg-slate-50/40 p-3">
      <p className="mb-2 text-[10px] font-bold uppercase tracking-wider text-slate-500">
        ที่มา
      </p>

      <div className="flex items-start gap-2">
        <span
          className={`shrink-0 rounded border px-1.5 py-0.5 text-[10px] font-bold ${TYPE_CHIP[item.type]}`}
          title={TYPE_TOOLTIP[item.type]}
        >
          {item.type}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-slate-800">
            {item.title}
          </p>
          <p className="mt-0.5 text-xs text-slate-500">
            จาก
            <span className="mx-1 font-semibold text-slate-700">
              {session.title}
            </span>
            {meetingLabel && <span>· {meetingLabel}</span>}
          </p>
        </div>
      </div>

      {pending_questions.length > 0 && (
        <div className="mt-3 rounded-md border border-amber-200 bg-amber-50/60 p-2.5">
          <p className="mb-1.5 flex items-center gap-1 text-[11px] font-semibold text-amber-800">
            <HelpCircle size={12} />
            คำถามค้างใน session นี้ ({pending_questions.length})
          </p>
          <ul className="ml-0.5 space-y-1 text-xs text-amber-900">
            {pending_questions.map((q) => (
              <li key={q.id} className="truncate">
                · {q.title}
              </li>
            ))}
          </ul>
        </div>
      )}

      <Link
        href={sessionHref}
        className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-primary hover:text-primary-hover"
      >
        ดู session เต็ม
        <ArrowUpRight size={12} />
      </Link>
    </section>
  );
}

export const CardSourceSection = memo(CardSourceSectionImpl);
