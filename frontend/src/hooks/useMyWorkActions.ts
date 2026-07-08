"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import { completeMyTask, fetchMyWork, snoozeCardDueDate } from "@/lib/myWorkApi";
import { useToastStore } from "@/store/useToastStore";
import type {
  MyWorkCard,
  MyWorkCounts,
  MyWorkGroup,
  MyWorkResponse,
} from "@/types/myWork";

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

interface UseMyWorkActionsArgs {
  data: MyWorkResponse | null;
  setData: Dispatch<SetStateAction<MyWorkResponse | null>>;
  setCounts: Dispatch<SetStateAction<MyWorkCounts | null>>;
  setError: Dispatch<SetStateAction<string | null>>;
}

// Owns the two mutating flows of the My Work inbox — mark-done (delayed
// commit + undo) and snooze (optimistic drop + refetch + undo) — plus the
// session-local "done today" counter that drives the hero progress meter.
export function useMyWorkActions({
  data,
  setData,
  setCounts,
  setError,
}: UseMyWorkActionsArgs) {
  const showToast = useToastStore((s) => s.show);

  // Tasks the user ticked off today, this session. The API doesn't report
  // "done today", so this resets on reload.
  const [doneToday, setDoneToday] = useState(0);
  // Tasks ticked done but still inside their undo window — the server call is
  // deferred, so this map holds the card (to restore on undo) + its timer.
  const pendingCompletions = useRef<
    Map<string, { card: MyWorkCard; timer: ReturnType<typeof setTimeout> }>
  >(new Map());

  // Drop any card mid-undo from a fetched list so a refetch doesn't resurrect
  // a row we're about to complete.
  const filterPending = useCallback((cards: MyWorkCard[]) => {
    const pending = pendingCompletions.current;
    return pending.size > 0 ? cards.filter((c) => !pending.has(c.id)) : cards;
  }, []);

  // Keep the inbox counts (stat hero + filter chips) in sync optimistically
  // rather than refetching — a refetch would resurrect other still-pending
  // completions.
  const adjustCounts = useCallback(
    (group: MyWorkGroup, delta: number) => {
      setCounts((c) =>
        c
          ? {
              ...c,
              [GROUP_TO_COUNT[group]]: Math.max(0, c[GROUP_TO_COUNT[group]] + delta),
              total: Math.max(0, c.total + delta),
            }
          : c,
      );
    },
    [setCounts],
  );

  const restorePending = useCallback(
    (card: MyWorkCard) => {
      setData((prev) => (prev ? { ...prev, cards: [...prev.cards, card] } : prev));
      adjustCounts(card.group, +1);
      if (card.group === "today") setDoneToday((n) => Math.max(0, n - 1));
    },
    [adjustCounts, setData],
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
    [restorePending, setError],
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
    [data, adjustCounts, setData, showToast, commitComplete, undoComplete],
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
    [setCounts, setData, setError],
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
    [data, setCounts, setData, setError, showToast, undoSnooze],
  );

  return { doneToday, filterPending, handleComplete, handleSnooze };
}
