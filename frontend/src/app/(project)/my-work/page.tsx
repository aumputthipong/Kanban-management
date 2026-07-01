"use client";

import { Search } from "lucide-react";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { MyWorkGreeting } from "@/components/my-work/MyWorkGreeting";
import { MyWorkStatCards } from "@/components/my-work/MyWorkStatCards";
import { DashboardGrid } from "@/components/my-work/DashboardGrid";
import type { BoardMeta } from "@/components/my-work/boardMeta";
import { MyWorkTaskModal } from "@/components/my-work/MyWorkTaskModal";
import { MyWorkSkeleton } from "@/components/my-work/MyWorkSkeleton";
import { apiClient } from "@/lib/apiClient";
import { completeMyTask, fetchMyWork, snoozeCardDueDate } from "@/lib/myWorkApi";
import { useToastStore } from "@/store/useToastStore";
import type { Board } from "@/types/board";
import {
  type MyWorkCard,
  type MyWorkCounts,
  type MyWorkGroup,
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

// Stable empty map so the project view doesn't get a new reference each render
// before the boards list has loaded.
const EMPTY_BOARD_META: Map<string, BoardMeta> = new Map();

// How long a ticked-off task stays undoable before its server call fires.
// Delayed commit ("Undo Send" pattern): an accidental tick on this daily-use
// page costs nothing because nothing is sent until the window elapses — and a
// completion actually moves the card into a DONE column, which has no clean
// reversal, so we'd rather never send it than try to undo it server-side.
const COMPLETE_UNDO_MS = 5000;

const GROUP_TO_COUNT: Record<MyWorkGroup, keyof Omit<MyWorkCounts, "total">> = {
  overdue: "overdue",
  today: "today",
  this_week: "this_week",
  later: "later",
  no_date: "no_date",
};

// Keep the toast readable — long card titles get truncated so the message +
// "back" action still fit on one line.
function shortTitle(title: string, max = 40): string {
  const t = title.trim();
  return t.length > max ? `${t.slice(0, max).trimEnd()}…` : t;
}

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

  const query = (searchParams.get("q") ?? "").trim();

  // board_id → glyph/accent for the project containers. Fetched client-side
  // (the layout already loads /boards for the sidebar; this is a cheap second
  // read that avoids extending the my-tasks API).
  const [boardMeta, setBoardMeta] = useState<Map<string, BoardMeta> | null>(null);
  // Task whose detail modal is open (clicking a row opens it instead of
  // navigating to the board).
  const [selectedCard, setSelectedCard] = useState<MyWorkCard | null>(null);

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
  // Tasks ticked done but still inside their undo window — the server call is
  // deferred, so this map holds the card (to restore on undo) + its timer.
  const pendingCompletions = useRef<
    Map<string, { card: MyWorkCard; timer: ReturnType<typeof setTimeout> }>
  >(new Map());

  const initialLoading = counts === null && error === null;
  const bodyLoading = data === null && error === null;

  // Load board glyph/accent for the project containers (once on mount).
  useEffect(() => {
    const controller = new AbortController();
    apiClient<Board[]>("/boards", { signal: controller.signal })
      .then((boards) =>
        setBoardMeta(
          new Map(boards.map((b) => [b.id, { color: b.color, icon: b.icon }])),
        ),
      )
      .catch(() => {
        // Graceful: project headers fall back to the default glyph + accent.
        setBoardMeta(new Map());
      });
    return () => controller.abort();
  }, []);

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
    fetchMyWork({ signal: controller.signal })
      .then((res) => {
        if (cancelled) return;
        // Drop any card mid-undo so a refetch doesn't resurrect a row we're
        // about to complete.
        const pending = pendingCompletions.current;
        const cards =
          pending.size > 0 ? res.cards.filter((c) => !pending.has(c.id)) : res.cards;
        setData({ ...res, cards });
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
  }, []);

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

  const showToast = useToastStore((s) => s.show);

  // ── Complete a task: delayed commit + undo ────────────────────────────────
  // Keep the inbox counts (stat hero + filter chips) in sync optimistically
  // rather than refetching — a refetch would resurrect other still-pending
  // completions.
  const adjustCounts = useCallback((group: MyWorkGroup, delta: number) => {
    setCounts((c) =>
      c
        ? {
            ...c,
            [GROUP_TO_COUNT[group]]: Math.max(0, c[GROUP_TO_COUNT[group]] + delta),
            total: Math.max(0, c.total + delta),
          }
        : c,
    );
  }, []);

  const restorePending = useCallback(
    (card: MyWorkCard) => {
      setData((prev) => (prev ? { ...prev, cards: [...prev.cards, card] } : prev));
      adjustCounts(card.group, +1);
      if (card.group === "today") setDoneToday((n) => Math.max(0, n - 1));
    },
    [adjustCounts],
  );

  // Fires when the undo window elapses — actually send the completion.
  const commitComplete = useCallback(
    async (cardId: string) => {
      const entry = pendingCompletions.current.get(cardId);
      if (!entry) return; // undone before the timer fired
      pendingCompletions.current.delete(cardId);
      try {
        await completeMyTask(cardId);
      } catch (err) {
        restorePending(entry.card);
        setError(err instanceof Error ? err.message : "ทำเครื่องหมายเสร็จไม่สำเร็จ");
      }
    },
    [restorePending],
  );

  const undoComplete = useCallback(
    (cardId: string) => {
      const entry = pendingCompletions.current.get(cardId);
      if (!entry) return; // already committed
      clearTimeout(entry.timer);
      pendingCompletions.current.delete(cardId);
      restorePending(entry.card);
    },
    [restorePending],
  );

  const handleComplete = useCallback(
    (cardId: string) => {
      if (!data || pendingCompletions.current.has(cardId)) return;
      const card = data.cards.find((c) => c.id === cardId);
      if (!card) return;

      // Optimistic: drop the row, decrement its count bucket, bump today's
      // progress. Nothing is sent yet — the timer below commits it.
      setData({ ...data, cards: data.cards.filter((c) => c.id !== cardId) });
      adjustCounts(card.group, -1);
      if (card.group === "today") setDoneToday((n) => n + 1);

      const timer = setTimeout(() => void commitComplete(cardId), COMPLETE_UNDO_MS);
      pendingCompletions.current.set(cardId, { card, timer });

      showToast({
        message: `ทำ "${shortTitle(card.title)}" เสร็จแล้ว`,
        actionLabel: "ย้อนกลับ",
        onAction: () => undoComplete(cardId),
        duration: COMPLETE_UNDO_MS,
      });
    },
    [data, adjustCounts, showToast, commitComplete, undoComplete],
  );

  // Commit any task still in its undo window when the page unmounts, so
  // navigating away finishes the action instead of silently dropping it.
  useEffect(() => {
    const pending = pendingCompletions.current;
    return () => {
      for (const [cardId, entry] of pending) {
        clearTimeout(entry.timer);
        void completeMyTask(cardId);
      }
      pending.clear();
    };
  }, []);

  // Revert a snooze back to the card's previous due_date. "" restores a card
  // that had no date (the backend treats an empty due_date as a clear). Reuses
  // the same PATCH endpoint as the forward snooze.
  const undoSnooze = useCallback(
    async (cardId: string, originalDueDate: string) => {
      try {
        await snoozeCardDueDate(cardId, originalDueDate);
        const refreshed = await fetchMyWork({});
        setData(refreshed);
        setCounts(refreshed.counts);
      } catch (err) {
        setError(err instanceof Error ? err.message : "ย้อนกลับไม่สำเร็จ");
      }
    },
    [],
  );

  const handleSnooze = useCallback(
    async (cardId: string, dueDate: string, label: string) => {
      if (!data) return;
      const prev = data;
      // Capture the original date before the optimistic drop so Undo can
      // restore it (the refetch below replaces `data`, losing the old value).
      const original = prev.cards.find((c) => c.id === cardId)?.due_date ?? "";
      // Optimistic: drop the card; refetch repopulates it into its new bucket
      // and refreshes the counts so the hero + panels stay in sync.
      setData({ ...prev, cards: prev.cards.filter((c) => c.id !== cardId) });
      try {
        await snoozeCardDueDate(cardId, dueDate);
        const refreshed = await fetchMyWork({});
        setData(refreshed);
        setCounts(refreshed.counts);
        showToast({
          message: `เลื่อนไป${label}แล้ว`,
          actionLabel: "ย้อนกลับ",
          onAction: () => undoSnooze(cardId, original),
          duration: 5000,
        });
      } catch (err) {
        setData(prev);
        setError(err instanceof Error ? err.message : "เลื่อนวันไม่สำเร็จ");
      }
    },
    [data, showToast, undoSnooze],
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
