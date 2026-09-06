import { apiClient } from "@/lib/apiClient";
import type { MyWorkFilter, MyWorkResponse } from "@/types/myWork";

export interface FetchMyWorkOptions {
  filter?: MyWorkFilter;
  includeUnassigned?: boolean;
  signal?: AbortSignal;
}

export function fetchMyWork(opts: FetchMyWorkOptions = {}): Promise<MyWorkResponse> {
  const params = new URLSearchParams();
  if (opts.filter && opts.filter !== "all") params.set("filter", opts.filter);
  if (opts.includeUnassigned) params.set("include_unassigned", "true");
  const qs = params.toString();
  return apiClient<MyWorkResponse>(`/my-tasks${qs ? `?${qs}` : ""}`, {
    signal: opts.signal,
  });
}

export function completeMyTask(cardId: string): Promise<void> {
  return apiClient(`/my-tasks/${cardId}/complete`, { method: "POST" });
}

/**
 * Snooze a card by setting `due_date`. Reuses PATCH /cards/:id — the assignee can
 * already edit their own card through that handler's inline permission gate.
 */
export function snoozeCardDueDate(cardId: string, dueDate: string): Promise<unknown> {
  return apiClient(`/cards/${cardId}`, {
    method: "PATCH",
    data: { due_date: dueDate },
  });
}

/**
 * Compute a YYYY-MM-DD string for "today + offset days" in the user's local
 * tz. Used by the snooze quick options ("tomorrow" = +1, "next week" = +7).
 */
export function relativeDueDate(offsetDays: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}
