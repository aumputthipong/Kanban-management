"use client";

import { useEffect, useState } from "react";
import { MyWorkGreeting } from "@/components/my-work/MyWorkGreeting";
import { MyWorkStatCards } from "@/components/my-work/MyWorkStatCards";
import { DashboardGrid, DashboardGridLoading } from "@/components/my-work/DashboardGrid";
import type { BoardMeta } from "@/components/my-work/boardMeta";
import { MyWorkTaskModal } from "@/components/my-work/MyWorkTaskModal";
import { MyWorkSkeleton } from "@/components/my-work/MyWorkSkeleton";
import { apiClient } from "@/lib/apiClient";
import { fetchMyWork } from "@/lib/myWorkApi";
import { useMyWorkActions } from "@/hooks/useMyWorkActions";
import type { Board } from "@/types/board";
import type { MyWorkCard, MyWorkCounts, MyWorkNoteTab, MyWorkResponse } from "@/types/myWork";

interface MeResponse {
  full_name?: string;
}

const EMPTY_COUNTS: MyWorkCounts = {
  overdue: 0,
  today: 0,
  this_week: 0,
  later: 0,
  no_date: 0,
  total: 0,
};

// Stable reference so the project view does not re-render before boards load.
const EMPTY_BOARD_META: Map<string, BoardMeta> = new Map();

function PageShell({ children }: { children: React.ReactNode }) {
  return (
    // bg-slate-50 = design.md `background`; the global --background is white.
    <div className="h-full bg-slate-50 overflow-y-auto lg:overflow-hidden lg:grid lg:grid-cols-[minmax(0,1fr)_21rem] lg:grid-rows-[minmax(0,1fr)]">
      {children}
    </div>
  );
}

export default function MyWorkPage() {
  const [boardMeta, setBoardMeta] = useState<Map<string, BoardMeta> | null>(null);
  const [selectedCard, setSelectedCard] = useState<MyWorkCard | null>(null);
  const [tab, setTab] = useState<MyWorkNoteTab>("today");

  const [data, setData] = useState<MyWorkResponse | null>(null);
  const [counts, setCounts] = useState<MyWorkCounts | null>(null);
  const [fullName, setFullName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const { doneToday, filterPending, handleComplete, handleSnooze } =
    useMyWorkActions({ data, setData, setCounts, setError });

  const initialLoading = counts === null && error === null;
  const bodyLoading = data === null && error === null;

  useEffect(() => {
    const controller = new AbortController();
    apiClient<Board[]>("/boards", { signal: controller.signal })
      .then((boards) =>
        setBoardMeta(
          new Map(boards.map((b) => [b.id, { color: b.color, icon: b.icon }])),
        ),
      )
      .catch(() => {
        setBoardMeta(new Map());
      });
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    apiClient<MeResponse>("/auth/me", { signal: controller.signal })
      .then((me) => {
        if (me?.full_name) setFullName(me.full_name);
      })
      .catch(() => {
        /* non-critical: greeting falls back to "คุณ" */
      });
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;
    fetchMyWork({ signal: controller.signal })
      .then((res) => {
        if (cancelled) return;
        setData({ ...res, cards: filterPending(res.cards) });
        setCounts(res.counts);
      })
      .catch((err: unknown) => {
        if (cancelled || controller.signal.aborted) return;
        setError(err instanceof Error ? err.message : "โหลดงานไม่สำเร็จ");
      });
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [filterPending]);

  if (initialLoading) {
    return (
      <PageShell>
        <MyWorkSkeleton />
      </PageShell>
    );
  }

  const header = (
    <>
      <MyWorkGreeting fullName={fullName} todayLeft={counts?.today ?? null} doneToday={doneToday}>
        {counts && (
          <MyWorkStatCards
            overdue={counts.overdue}
            today={counts.today}
            thisWeek={counts.this_week}
            activeTab={tab}
            onSelectTab={setTab}
          />
        )}
      </MyWorkGreeting>
      {error && (
        <div className="mt-4 px-3 py-2 rounded-lg border border-red-200 bg-red-50 text-xs text-red-700">{error}</div>
      )}
    </>
  );

  return (
    <PageShell>
      {bodyLoading ? (
        <DashboardGridLoading />
      ) : (
        <DashboardGrid
          header={header}
          cards={data?.cards ?? []}
          counts={counts ?? EMPTY_COUNTS}
          boardMeta={boardMeta ?? EMPTY_BOARD_META}
          doneToday={doneToday}
          onOpenCard={setSelectedCard}
          onComplete={handleComplete}
          tab={tab}
          onTabChange={setTab}
        />
      )}

      {selectedCard && (
        <MyWorkTaskModal
          key={selectedCard.id}
          card={selectedCard}
          boardMeta={boardMeta ?? EMPTY_BOARD_META}
          onClose={() => setSelectedCard(null)}
          onComplete={handleComplete}
          onSnooze={handleSnooze}
        />
      )}
    </PageShell>
  );
}
