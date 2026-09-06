// Owns the planning session's items state and every mutation the capture surface
// needs. All mutations are optimistic and fire-and-forget with a toast on failure:
// during a meeting, capture velocity beats confirming trivial writes.
import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useToastStore } from "@/store/useToastStore";
import { planningApi } from "@/lib/planningApi";
import type {
  PlanningItem,
  PlanningItemStatus,
  PlanningItemType,
  PlanningSessionDetail,
} from "@/types/planning";
import type { SessionStats } from "@/components/board/planning/SessionSidebar";

export interface UseSessionItemsResult {
  detail: PlanningSessionDetail | null;
  items: PlanningItem[];
  savedAt: string;
  stats: SessionStats;
  promotedItems: PlanningItem[];
  commitNew: (title: string, type: PlanningItemType) => Promise<void>;
  patchItem: (
    id: string,
    patch: Partial<PlanningItem>,
    optimistic: Partial<PlanningItem>,
  ) => void;
  toggleStatus: (item: PlanningItem, target: PlanningItemStatus) => void;
  changeType: (item: PlanningItem, t: PlanningItemType) => void;
  removeItem: (item: PlanningItem) => Promise<void>;
  promoteMany: (ids: string[]) => Promise<void>;
  promoteOne: (item: PlanningItem) => Promise<void>;
}

export function useSessionItems(
  boardId: string,
  sessionId: string,
): UseSessionItemsResult {
  const router = useRouter();
  const [detail, setDetail] = useState<PlanningSessionDetail | null>(null);
  const [items, setItems] = useState<PlanningItem[]>([]);
  const [savedAt, setSavedAt] = useState<string>("");
  const showToast = useToastStore((s) => s.show);

  // Bounce to the session list on 4xx — usually a deleted session or a stale link.
  useEffect(() => {
    let cancelled = false;
    planningApi
      .getSession(sessionId)
      .then((d) => {
        if (cancelled) return;
        setDetail(d);
        setItems(d.items);
        setSavedAt(d.updated_at);
      })
      .catch(() => {
        if (cancelled) return;
        showToast({ message: "โหลดบันทึกไม่ได้", duration: 4000 });
        router.push(`/board/${boardId}/planning`);
      });
    return () => {
      cancelled = true;
    };
  }, [sessionId, boardId, router, showToast]);

  // Same NOT IN exclusion as the backend session-summary aggregate, or counts drift.
  const stats = useMemo<SessionStats>(() => {
    const s: SessionStats = { REQ: 0, DEC: 0, Q: 0, dropped: 0, promoted: 0, selected: 0 };
    for (const it of items) {
      if (it.status === "dropped") s.dropped++;
      else if (it.status === "promoted") s.promoted++;
      else {
        s[it.type]++;
        if (it.status === "selected") s.selected++;
      }
    }
    return s;
  }, [items]);

  const promotedItems = useMemo(
    () => items.filter((it) => it.status === "promoted"),
    [items],
  );

  const patchItem = useCallback(
    (id: string, patch: Partial<PlanningItem>, optimistic: Partial<PlanningItem>) => {
      setItems((prev) => prev.map((it) => (it.id === id ? { ...it, ...optimistic } : it)));
      setSavedAt(new Date().toISOString());
      planningApi
        .updateItem(id, {
          type: patch.type,
          title: patch.title,
          description: patch.description ?? undefined,
          status: patch.status,
          position: patch.position,
          implementation_note: patch.implementation_note ?? undefined,
        })
        .catch(() => {
          showToast({ message: "บันทึกไม่สำเร็จ", duration: 4000 });
        });
    },
    [showToast],
  );

  const commitNew = useCallback(
    async (title: string, type: PlanningItemType) => {
      const trimmed = title.trim();
      if (!trimmed) return;
      // The API returns the real id; swap in place so later edits target the real row.
      const tempId = `__pending_${Math.random().toString(36).slice(2)}`;
      const placeholder: PlanningItem = {
        id: tempId,
        session_id: sessionId,
        type,
        title: trimmed,
        description: null,
        status: "live",
        promoted_to_card_id: null,
        position: Number.MAX_SAFE_INTEGER,
        created_at: new Date().toISOString(),
      };
      setItems((prev) => [...prev, placeholder]);
      setSavedAt(placeholder.created_at);
      try {
        const real = await planningApi.createItem(sessionId, {
          type,
          title: trimmed,
        });
        setItems((prev) => prev.map((it) => (it.id === tempId ? real : it)));
      } catch {
        setItems((prev) => prev.filter((it) => it.id !== tempId));
        showToast({ message: "เพิ่มไม่ได้ ลองอีกครั้ง", duration: 4000 });
      }
    },
    [sessionId, showToast],
  );

  const toggleStatus = useCallback(
    (item: PlanningItem, target: PlanningItemStatus) => {
      const next: PlanningItemStatus = item.status === target ? "live" : target;
      patchItem(item.id, { status: next }, { status: next });
    },
    [patchItem],
  );

  const changeType = useCallback(
    (item: PlanningItem, t: PlanningItemType) => {
      if (item.type === t) return;
      // The one PATCH the backend can reject (promoted items are frozen), so revert
      // by hand or the chip stays on a type that never landed.
      const previous = item.type;
      setItems((prev) => prev.map((it) => (it.id === item.id ? { ...it, type: t } : it)));
      setSavedAt(new Date().toISOString());
      planningApi
        .updateItem(item.id, { type: t })
        .then(() => {
          showToast({ message: `เปลี่ยนเป็น ${t} แล้ว`, duration: 2500 });
        })
        .catch((err: unknown) => {
          setItems((prev) =>
            prev.map((it) => (it.id === item.id ? { ...it, type: previous } : it)),
          );
          const message =
            err instanceof Error && err.message
              ? err.message
              : "เปลี่ยนประเภทไม่ได้";
          showToast({ message, duration: 4000 });
        });
    },
    [showToast],
  );

  const removeItem = useCallback(
    async (item: PlanningItem) => {
      setItems((prev) => prev.filter((it) => it.id !== item.id));
      try {
        await planningApi.deleteItem(item.id);
      } catch {
        setItems((prev) => [...prev, item]);
        showToast({ message: "ลบไม่ได้ ลองอีกครั้ง", duration: 4000 });
      }
    },
    [showToast],
  );

  // Per-row promote — same endpoint as the batch path, without the select two-step.
  const promoteOne = useCallback(
    async (item: PlanningItem) => {
      try {
        const res = await planningApi.promoteItem(item.id);
        setItems((prev) =>
          prev.map((cur) => (cur.id === item.id ? res.item : cur)),
        );
        showToast({ message: "ส่งเข้า Board แล้ว", duration: 2500 });
      } catch {
        showToast({
          message: `ส่งเข้า Board ไม่ได้: ${item.title}`,
          duration: 4000,
        });
      }
    },
    [showToast],
  );

  // Bulk-promote by explicit ids from select mode. Already-promoted ids are skipped
  // so a stale selection cannot double-promote.
  const promoteMany = useCallback(
    async (ids: string[]) => {
      const idSet = new Set(ids);
      const targets = items.filter(
        (it) => idSet.has(it.id) && it.status !== "promoted",
      );
      if (targets.length === 0) return;
      // Promote sequentially to keep board card positions stable.
      for (const it of targets) {
        try {
          const res = await planningApi.promoteItem(it.id);
          setItems((prev) =>
            prev.map((cur) => (cur.id === it.id ? res.item : cur)),
          );
        } catch {
          showToast({
            message: `ส่งเข้า Board ไม่ได้: ${it.title}`,
            duration: 4000,
          });
        }
      }
      showToast({
        message: `ส่งเข้า Board แล้ว ${targets.length} รายการ`,
        duration: 3000,
      });
    },
    [items, showToast],
  );

  return {
    detail,
    items,
    savedAt,
    stats,
    promotedItems,
    commitNew,
    patchItem,
    toggleStatus,
    changeType,
    removeItem,
    promoteMany,
    promoteOne,
  };
}
