"use client";

import { useMemo, useState } from "react";
import { Plus, Sparkles, User } from "lucide-react";
import { useRouter } from "next/navigation";
import type { Card } from "@/types/board";
import { useBoardStore } from "@/store/useBoardStore";
import {
  CriticalHero,
  RankedRow,
  UpcomingRow,
  CollapsedEmpty,
} from "./TaskTriageRows";

type UpcomingBucket = "today" | "tomorrow" | "thisWeek";

interface TasksTabContentProps {
  boardId: string;
  overdueCards: Card[];
  todayCards: Card[];
  tomorrowCards: Card[];
  thisWeekCards: Card[];
  onSelectCard: (card: Card) => void;
}

const UPCOMING_TH: Record<UpcomingBucket, string> = {
  today: "วันนี้",
  tomorrow: "พรุ่งนี้",
  thisWeek: "สัปดาห์นี้",
};

function upcomingLabel(bucket: UpcomingBucket, card: Card): string {
  if (bucket !== "thisWeek") return UPCOMING_TH[bucket];
  if (card.due_date)
    return new Date(card.due_date).toLocaleDateString("th-TH", { weekday: "short" });
  return "สัปดาห์นี้";
}

const sortByDate = (list: Card[]) =>
  [...list].sort((a, b) => {
    if (!a.due_date) return 1;
    if (!b.due_date) return -1;
    return new Date(a.due_date).getTime() - new Date(b.due_date).getTime();
  });

export function TasksTabContent({
  boardId,
  overdueCards,
  todayCards,
  tomorrowCards,
  thisWeekCards,
  onSelectCard,
}: TasksTabContentProps) {
  const router = useRouter();
  const { columns, currentUserId } = useBoardStore();
  const [mineOnly, setMineOnly] = useState(false);

  const columnTitleById = useMemo(() => {
    const map = new Map<string, string>();
    for (const col of columns) map.set(col.id, col.title);
    return map;
  }, [columns]);

  const hasMine = useMemo(
    () =>
      [...overdueCards, ...todayCards, ...tomorrowCards, ...thisWeekCards].some(
        (c) => c.assignee_id === currentUserId,
      ),
    [overdueCards, todayCards, tomorrowCards, thisWeekCards, currentUserId],
  );

  const buckets = useMemo(() => {
    const filterMine = (list: Card[]) =>
      mineOnly && currentUserId
        ? list.filter((c) => c.assignee_id === currentUserId)
        : list;
    return {
      overdue: sortByDate(filterMine(overdueCards)),
      today: sortByDate(filterMine(todayCards)),
      tomorrow: sortByDate(filterMine(tomorrowCards)),
      thisWeek: sortByDate(filterMine(thisWeekCards)),
    };
  }, [overdueCards, todayCards, tomorrowCards, thisWeekCards, mineOnly, currentUserId]);

  const upcomingTotal =
    buckets.today.length + buckets.tomorrow.length + buckets.thisWeek.length;
  const totalUrgent = buckets.overdue.length + upcomingTotal;

  const [hero, ...restOverdue] = buckets.overdue;
  const upcomingOrder: UpcomingBucket[] = ["today", "tomorrow", "thisWeek"];
  const emptyUpcoming = upcomingOrder.filter((b) => buckets[b].length === 0);

  const colTitle = (card: Card) => columnTitleById.get(card.column_id) ?? "";

  return (
    <div>
      {/* Header — title + urgency summary + actions */}
      <div className="flex items-end gap-4 mb-5">
        <div className="flex-1 min-w-0">
          <h2 className="text-2xl font-bold text-slate-900 leading-tight tracking-tight">
            Tasks
          </h2>
          <div className="flex items-center gap-2 mt-1.5 text-xs text-slate-500 flex-wrap">
            {buckets.overdue.length > 0 ? (
              <span className="inline-flex items-center rounded bg-rose-50 text-rose-700 border border-rose-200 px-1.5 py-0.5 text-[11px] font-semibold">
                {buckets.overdue.length} เลยกำหนด
              </span>
            ) : (
              <span className="font-semibold text-slate-600">ไม่มีงานเลยกำหนด</span>
            )}
            <span aria-hidden className="w-0.5 h-0.5 rounded-full bg-slate-300" />
            <span>{upcomingTotal} ภายในสัปดาห์นี้</span>
            <span aria-hidden className="w-0.5 h-0.5 rounded-full bg-slate-300" />
            <span>เรียงตามความเร่งด่วน</span>
          </div>
        </div>
        {hasMine && (
          <button
            type="button"
            onClick={() => setMineOnly((v) => !v)}
            className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold rounded-lg border transition-colors shrink-0 ${
              mineOnly
                ? "bg-blue-50 border-blue-200 text-blue-700"
                : "bg-white border-slate-300 text-slate-600 hover:bg-slate-50"
            }`}
          >
            <User size={12} />
            Just mine
          </button>
        )}
        <button
          onClick={() => router.push(`/board/${boardId}/tasks`)}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-white border border-slate-300 hover:bg-slate-50 text-slate-700 text-xs font-semibold rounded-lg transition-colors shadow-sm shrink-0"
        >
          <Plus size={13} />
          New Task
        </button>
      </div>

      {totalUrgent === 0 ? (
        <div className="text-center py-16 border border-dashed border-slate-200 rounded-xl">
          <div className="inline-flex items-center justify-center w-10 h-10 rounded-full bg-emerald-50 text-emerald-600 mb-2">
            <Sparkles size={18} />
          </div>
          <p className="text-sm font-semibold text-slate-700">
            All clear — no urgent tasks right now
          </p>
          <p className="text-xs text-slate-500 mt-1">
            {mineOnly
              ? "ไม่มีงานเร่งด่วนของคุณ"
              : "ไม่มีงานเลยกำหนด หรือใกล้กำหนดใน Board นี้"}
          </p>
        </div>
      ) : (
        <>
          {hero && (
            <CriticalHero card={hero} columnTitle={colTitle(hero)} onSelect={onSelectCard} />
          )}

          {restOverdue.length > 0 && (
            <section className="mb-6">
              <GroupLabel
                label={`เลยกำหนด · อีก ${restOverdue.length} งาน`}
                count={restOverdue.length}
              />
              <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
                {restOverdue.map((card) => (
                  <RankedRow
                    key={card.id}
                    card={card}
                    columnTitle={colTitle(card)}
                    onSelect={onSelectCard}
                  />
                ))}
              </div>
            </section>
          )}

          {/* ถัดไป — upcoming buckets: empties collapse, the rest list as rows */}
          <section>
            <GroupLabel label="ถัดไป" />
            {emptyUpcoming.length > 0 && (
              <div className="grid grid-cols-2 lg:grid-cols-3 gap-2.5 mb-2.5">
                {emptyUpcoming.map((b) => (
                  <CollapsedEmpty key={b} label={UPCOMING_TH[b]} />
                ))}
              </div>
            )}
            {upcomingTotal > 0 && (
              <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
                {upcomingOrder.flatMap((b) =>
                  buckets[b].map((card) => (
                    <UpcomingRow
                      key={card.id}
                      card={card}
                      columnTitle={colTitle(card)}
                      whenLabel={upcomingLabel(b, card)}
                      onSelect={onSelectCard}
                    />
                  )),
                )}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function GroupLabel({ label, count }: { label: string; count?: number }) {
  return (
    <div className="flex items-center gap-2.5 mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
      <span>{label}</span>
      {count != null && (
        <span className="inline-flex items-center justify-center min-w-5 h-[19px] px-1.5 rounded text-[11px] font-bold text-slate-600 bg-slate-100">
          {count}
        </span>
      )}
      <span aria-hidden className="flex-1 h-px bg-slate-200" />
    </div>
  );
}
