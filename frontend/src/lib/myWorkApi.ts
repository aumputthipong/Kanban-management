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

/** Reuses PATCH /cards/:id — the assignee can already edit their own card. */
export function snoozeCardDueDate(cardId: string, dueDate: string): Promise<unknown> {
  return apiClient(`/cards/${cardId}`, {
    method: "PATCH",
    data: { due_date: dueDate },
  });
}

export function relativeDueDate(offsetDays: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}
