"use client";

import { Search } from "lucide-react";
import { Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { MyWorkGreeting } from "@/components/my-work/MyWorkGreeting";
import { MyWorkStatCards } from "@/components/my-work/MyWorkStatCards";
import { DashboardGrid } from "@/components/my-work/DashboardGrid";
import type { BoardMeta } from "@/components/my-work/boardMeta";
import { MyWorkTaskModal } from "@/components/my-work/MyWorkTaskModal";
import { MyWorkSkeleton } from "@/components/my-work/MyWorkSkeleton";
import { Skeleton } from "@/components/ui/Skeleton";
import { apiClient } from "@/lib/apiClient";
import { fetchMyWork } from "@/lib/myWorkApi";
import { useMyWorkActions } from "@/hooks/useMyWorkActions";
import type { Board } from "@/types/board";
import {
  type MyWorkCard,
  type MyWorkCounts,
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

// Stable reference so the project view does not re-render before boards load.
const EMPTY_BOARD_META: Map<string, BoardMeta> = new Map();

  // useSearchParams in a client component makes Next 16 bail out of static prerender
  // unless the consumer sits behind <Suspense>, hence the split.
export default function MyWorkPage() {
  return (
    <Suspense fallback={<MyWorkFallback />}>
      <MyWorkPageInner />
    </Suspense>
  );
}

function PageShell({ children }: { children: React.ReactNode }) {
  // Below lg the dashboard stacks and the page scrolls; at lg it is a fixed-height
  // viewport so each panel scrolls internally instead of the page.
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

  const query = (searchParams.get("q") ?? "").trim();

  // board_id -> glyph/accent. Cheap second read; avoids extending the my-tasks API.
  const [boardMeta, setBoardMeta] = useState<Map<string, BoardMeta> | null>(null);
  const [selectedCard, setSelectedCard] = useState<MyWorkCard | null>(null);

  const [data, setData] = useState<MyWorkResponse | null>(null);
  // Counts cover the whole inbox, so keep them across filter switches (AGENTS.md).
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
        // Project headers fall back to the default glyph and accent.
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
        /* greeting falls back to "คุณ" — non-critical */
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

      <div className="flex items-center justify-end gap-4 flex-none flex-wrap dash-reveal d2">
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
          cards={filteredCards}
          counts={chipCounts}
          boardMeta={boardMeta ?? EMPTY_BOARD_META}
          doneToday={doneToday}
          onOpenCard={setSelectedCard}
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

function DashboardLoading() {
  return (
    <div className="grid gap-[18px] min-h-0 lg:flex-1 grid-cols-1 lg:[grid-template-columns:minmax(0,1.9fr)_minmax(300px,1fr)]">
      <Skeleton className="rounded-xl min-h-40" />
      <div className="grid gap-[18px] lg:[grid-template-rows:auto_minmax(0,1fr)]">
        <Skeleton className="rounded-xl min-h-24" />
        <Skeleton className="rounded-xl min-h-40" />
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
  // Keep local input synced with URL changes from filter chip clicks —
  // setState-during-render pattern (React 19 forbids synchronous setState in
  // an effect body): track the last seen prop and reset before returning JSX.
  const [lastValue, setLastValue] = useState(value);
  if (value !== lastValue) {
    setLastValue(value);
    setLocal(value);
  }
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
