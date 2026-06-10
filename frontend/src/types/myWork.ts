// Shape of GET /api/my-tasks (the cross-board personal inbox).
//
// `cards` is pre-filtered server-side; `counts` always reflects the full
// inbox so filter chips can render counters from a single request.

export type MyWorkStatus = "todo" | "in_progress" | "done";

export type MyWorkGroup =
  | "overdue"
  | "today"
  | "this_week"
  | "later"
  | "no_date";

export type MyWorkFilter = "all" | "overdue" | "today" | "this_week" | "no_date";

export interface MyWorkCard {
  id: string;
  title: string;
  board_id: string;
  board_name: string;
  column_name: string;
  priority: "low" | "medium" | "high" | null;
  due_date: string | null;
  estimated_hours: number | null;
  status: MyWorkStatus;
  group: MyWorkGroup;
  total_subtasks: number;
  completed_subtasks: number;
}

export interface MyWorkCounts {
  overdue: number;
  today: number;
  this_week: number;
  later: number;
  no_date: number;
  total: number;
}

export interface MyWorkResponse {
  cards: MyWorkCard[];
  counts: MyWorkCounts;
}

export const VALID_FILTERS: readonly MyWorkFilter[] = [
  "all",
  "overdue",
  "today",
  "this_week",
  "no_date",
] as const;

export function isMyWorkFilter(value: string | null): value is MyWorkFilter {
  return value != null && (VALID_FILTERS as readonly string[]).includes(value);
}
