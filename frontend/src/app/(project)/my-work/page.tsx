"use client";

import { Search } from "lucide-react";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { FilterChipBar } from "@/components/my-work/FilterChipBar";
import { MyWorkGreeting } from "@/components/my-work/MyWorkGreeting";
import { MyWorkStatCards } from "@/components/my-work/MyWorkStatCards";
import { DashboardGrid } from "@/components/my-work/DashboardGrid";
import { MyWorkSkeleton } from "@/components/my-work/MyWorkSkeleton";
import { apiClient } from "@/lib/apiClient";
import { completeMyTask, fetchMyWork, snoozeCardDueDate } from "@/lib/myWorkApi";
import {
  isMyWorkFilter,
  type MyWorkCounts,
  type MyWorkFilter,
  type MyWorkResponse,
} from "@/types/myWork";

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

// useSearchParams() in a client component forces Next.js 16 to bail out of
// static prerender unless we wrap the consumer in <Suspense>. The real page
// logic lives in MyWorkPageInner so the prerender pass stops at the fallback.
export default function MyWorkPage() {
  return (
    <Suspense fallback={<MyWorkFallback />}>
      <MyWorkPageInner />
    </Suspense>
  );
}

function PageShell({ children }: { children: React.ReactNode }) {
  // Below lg the dashboard stacks to one column and the page scrolls
  // (min-h-full → grows with content). At lg it becomes a fixed-height
  // single viewport (h-full + outer overflow-hidden) so each panel scrolls
  // internally instead of the page.
  return (
    <div className="h-full overflow-y-auto lg:overflow-hidden">
      <div className="mx-auto max-w-[1320px] min-h-full lg:h-full flex flex-col px-6 py-5 lg:px-8 gap-4">
        {children}
      </div>
    </div>
  );
}

function MyWorkFallback() {
  return (
    <PageShell>
      <MyWorkSkeleton />
    </PageShell>
  );
}

function MyWorkPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const filterParam = searchParams.get("filter");
  const filter: MyWorkFilter = isMyWorkFilter(filterParam) ? filterParam : "all";
  const query = (searchParams.get("q") ?? "").trim();

  const [data, setData] = useState<MyWorkResponse | null>(null);
  // Counts cover the full inbox regardless of the active filter, so we keep
  // them across filter switches — the greeting + stat cards stay stable while
  // only the panels below re-fetch.
  const [counts, setCounts] = useState<MyWorkCounts | null>(null);
  const [fullName, setFullName] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  // Tasks the user ticked off today, this session — drives the hero progress
  // meter + greeting. The API doesn't report "done today", so this is a
  // session-local count that resets on reload.
  const [doneToday, setDoneToday] = useState(0);

  // setState-during-render pattern (per AGENTS.md): when the filter URL param
  // changes, drop the cached card list so the body flips to a skeleton before
  // the effect fires. Counts are intentionally preserved.
  const [lastFilter, setLastFilter] = useState(filter);
  if (lastFilter !== filter) {
    setLastFilter(filter);
    setData(null);
    setDoneToday(0);
  }
  const initialLoading = counts === null && error === null;
  const bodyLoading = data === null && error === null;

  // Greeting name — fetched once, independent of the filter.
  useEffect(() => {
    const controller = new AbortController();
    apiClient<MeResponse>("/auth/me", { signal: controller.signal })
      .then((me) => {
        if (me?.full_name) setFullName(me.full_name);
      })
      .catch(() => {
        /* greeting falls back to "คุณ" — non-critical */
      });
    return () => controller.abort();
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;
    fetchMyWork({ filter, signal: controller.signal })
      .then((res) => {
        if (cancelled) return;
        setData(res);
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
  }, [filter]);

  const setFilter = useCallback(
    (next: MyWorkFilter) => {
      const params = new URLSearchParams(searchParams);
      if (next === "all") params.delete("filter");
      else params.set("filter", next);
      const qs = params.toString();
      router.replace(qs ? `/my-work?${qs}` : "/my-work");
    },
    [router, searchParams],
  );

  const setQuery = useCallback(
    (next: string) => {
      const params = new URLSearchParams(searchParams);
      const trimmed = next.trim();
      if (trimmed === "") params.delete("q");
      else params.set("q", trimmed);
      const qs = params.toString();
      router.replace(qs ? `/my-work?${qs}` : "/my-work");
    },
    [router, searchParams],
  );

  const handleComplete = useCallback(
    async (cardId: string) => {
      if (!data) return;
      const prev = data;
      const wasToday = prev.cards.find((c) => c.id === cardId)?.group === "today";
      setData({ ...prev, cards: prev.cards.filter((c) => c.id !== cardId) });
      try {
        await completeMyTask(cardId);
        if (wasToday) setDoneToday((n) => n + 1);
        const refreshed = await fetchMyWork({ filter });
        setData(refreshed);
        setCounts(refreshed.counts);
      } catch (err) {
        setData(prev);
        setError(err instanceof Error ? err.message : "ทำเครื่องหมายเสร็จไม่สำเร็จ");
      }
    },
    [data, filter],
  );

  const handleSnooze = useCallback(
    async (cardId: string, dueDate: string) => {
      if (!data) return;
      const prev = data;
      // Optimistic: drop the card; refetch repopulates it into its new bucket
      // and refreshes the counts so the hero + panels stay in sync.
      setData({ ...prev, cards: prev.cards.filter((c) => c.id !== cardId) });
      try {
        await snoozeCardDueDate(cardId, dueDate);
        const refreshed = await fetchMyWork({ filter });
        setData(refreshed);
        setCounts(refreshed.counts);
      } catch (err) {
        setData(prev);
        setError(err instanceof Error ? err.message : "เลื่อนวันไม่สำเร็จ");
      }
    },
    [data, filter],
  );

  const filteredCards = useMemo(() => {
    const cards = data?.cards ?? [];
    if (!query) return cards;
    const needle = query.toLowerCase();
    return cards.filter(
      (c) =>
        c.title.toLowerCase().includes(needle) ||
        c.board_name.toLowerCase().includes(needle),
    );
  }, [data?.cards, query]);

  const chipCounts = counts ?? EMPTY_COUNTS;

  if (initialLoading) {
    return (
      <PageShell>
        <MyWorkSkeleton />
      </PageShell>
    );
  }

  return (
    <PageShell>
      <div className="flex items-end justify-between gap-7 flex-none dash-reveal d1">
        <MyWorkGreeting fullName={fullName} />
        {counts && (
          <MyWorkStatCards
            overdue={counts.overdue}
            today={counts.today}
            thisWeek={counts.this_week}
          />
        )}
      </div>

      <div className="flex items-center justify-between gap-4 flex-none flex-wrap dash-reveal d2">
        <FilterChipBar active={filter} counts={chipCounts} onChange={setFilter} />
        <SearchInput value={query} onChange={setQuery} />
      </div>

      {error && (
        <div className="flex-none px-3 py-2 rounded-lg border border-red-200 bg-red-50 text-xs text-red-700">
          {error}
        </div>
      )}

      {bodyLoading ? (
        <DashboardLoading />
      ) : (
        <DashboardGrid
          filter={filter}
          cards={filteredCards}
          counts={chipCounts}
          doneToday={doneToday}
          onComplete={handleComplete}
          onSnooze={handleSnooze}
        />
      )}
    </PageShell>
  );
}

function DashboardLoading() {
  return (
    <div className="grid gap-[18px] min-h-0 lg:flex-1 grid-cols-1 lg:[grid-template-columns:minmax(0,1.9fr)_minmax(300px,1fr)]">
      <div className="border border-slate-200 rounded-xl bg-white shadow-sm animate-pulse min-h-40" />
      <div className="grid gap-[18px] lg:[grid-template-rows:auto_minmax(0,1fr)]">
        <div className="border border-slate-200 rounded-xl bg-white shadow-sm animate-pulse min-h-24" />
        <div className="border border-slate-200 rounded-xl bg-white shadow-sm animate-pulse min-h-40" />
      </div>
    </div>
  );
}

function SearchInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (next: string) => void;
}) {
  const [local, setLocal] = useState(value);
  // Keep local input synced with URL changes from filter chip clicks.
  useEffect(() => {
    setLocal(value);
  }, [value]);
  // Debounce URL writes so each keystroke doesn't push a history entry.
  useEffect(() => {
    if (local === value) return;
    const handle = window.setTimeout(() => onChange(local), 250);
    return () => window.clearTimeout(handle);
  }, [local, value, onChange]);
  return (
    <label className="relative flex items-center">
      <Search size={14} className="absolute left-3 text-slate-400" />
      <input
        type="search"
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        placeholder="ค้นหางาน..."
        className="h-[35px] pl-9 pr-3 border border-slate-200 rounded-sm bg-white text-sm text-slate-700 placeholder:text-slate-400 focus:outline-none focus:border-blue-300 focus:ring-3 focus:ring-blue-50 w-60"
      />
    </label>
  );
}
