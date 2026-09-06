// Comment thread for one planning item. Lazy on purpose: nothing loads until the
// consumer calls load(). ItemRow mounts one instance per visible row — see
// docs/ARCHITECTURE.md, "Planning section", for why that stays cheap.
import { useCallback, useEffect, useRef, useState } from "react";
import { useToastStore } from "@/store/useToastStore";
import { planningApi } from "@/lib/planningApi";
import type { PlanningComment } from "@/types/planning";

interface State {
  comments: PlanningComment[];
  isLoading: boolean;
  loaded: boolean;
}

const tempId = () => `__pending_${Math.random().toString(36).slice(2)}`;

export function usePlanningComments(itemId: string, currentUserId: string | null) {
  const [state, setState] = useState<State>({ comments: [], isLoading: false, loaded: false });
  const showToast = useToastStore((s) => s.show);
  // ref, not state — a render-triggering flag here would cascade.
  const requestRef = useRef(0);

  const load = useCallback(async () => {
    const reqId = ++requestRef.current;
    setState((prev) => ({ ...prev, isLoading: true }));
    try {
      const rows = await planningApi.listComments(itemId);
      if (requestRef.current !== reqId) return;
      setState({ comments: rows, isLoading: false, loaded: true });
    } catch {
      if (requestRef.current !== reqId) return;
      setState((prev) => ({ ...prev, isLoading: false }));
      showToast({ message: "โหลดความคิดเห็นไม่ได้ ลองอีกครั้ง", duration: 4000 });
    }
  }, [itemId, showToast]);

  // Refetch on tab focus — collaborative enough that a stale thread confuses, but a
  // background poll would burn requests on idle tabs.
  useEffect(() => {
    if (!state.loaded) return;
    const onVisible = () => {
      if (document.visibilityState === "visible") {
        void load();
      }
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [state.loaded, load]);

  const create = useCallback(
    async (body: string) => {
      const trimmed = body.trim();
      if (!trimmed) return;
      const placeholderId = tempId();
      const now = new Date().toISOString();
      const placeholder: PlanningComment = {
        id: placeholderId,
        item_id: itemId,
        author_id: currentUserId ?? "",
        author_name: "",
        body: trimmed,
        created_at: now,
        updated_at: now,
        deleted_at: null,
      };
      setState((prev) => ({ ...prev, comments: [...prev.comments, placeholder] }));
      try {
        const real = await planningApi.createComment(itemId, trimmed);
        setState((prev) => ({
          ...prev,
          comments: prev.comments.map((c) => (c.id === placeholderId ? real : c)),
        }));
      } catch {
        setState((prev) => ({
          ...prev,
          comments: prev.comments.filter((c) => c.id !== placeholderId),
        }));
        showToast({ message: "ส่งความคิดเห็นไม่สำเร็จ", duration: 4000 });
      }
    },
    [itemId, currentUserId, showToast],
  );

  const edit = useCallback(
    async (commentId: string, body: string) => {
      const trimmed = body.trim();
      if (!trimmed) return;
      let previous: PlanningComment | undefined;
      setState((prev) => {
        previous = prev.comments.find((c) => c.id === commentId);
        return {
          ...prev,
          comments: prev.comments.map((c) =>
            c.id === commentId ? { ...c, body: trimmed, updated_at: new Date().toISOString() } : c,
          ),
        };
      });
      try {
        const updated = await planningApi.editComment(commentId, trimmed);
        setState((prev) => ({
          ...prev,
          comments: prev.comments.map((c) => (c.id === commentId ? updated : c)),
        }));
      } catch {
        if (previous) {
          const restore = previous;
          setState((prev) => ({
            ...prev,
            comments: prev.comments.map((c) => (c.id === commentId ? restore : c)),
          }));
        }
        showToast({ message: "แก้ไม่สำเร็จ", duration: 4000 });
      }
    },
    [showToast],
  );

  const remove = useCallback(
    async (commentId: string) => {
      let previous: PlanningComment | undefined;
      setState((prev) => {
        previous = prev.comments.find((c) => c.id === commentId);
        return {
          ...prev,
          comments: prev.comments.map((c) =>
            c.id === commentId ? { ...c, body: null, deleted_at: new Date().toISOString() } : c,
          ),
        };
      });
      try {
        await planningApi.deleteComment(commentId);
      } catch {
        if (previous) {
          const restore = previous;
          setState((prev) => ({
            ...prev,
            comments: prev.comments.map((c) => (c.id === commentId ? restore : c)),
          }));
        }
        showToast({ message: "ลบไม่สำเร็จ", duration: 4000 });
      }
    },
    [showToast],
  );

  return {
    comments: state.comments,
    isLoading: state.isLoading,
    loaded: state.loaded,
    load,
    create,
    edit,
    remove,
  };
}
