// types/planning.ts
//
// Planning section mirrors backend dto.PlanningSessionSummary / Detail /
// PlanningItemResponse. The three item types (REQ / DEC / Q) reflect the
// three questions any session answers: what we want, what we've decided,
// what we still don't know. "DROP" is a *status* on an item, not a fourth
// type — see backend/internal/service/planning_service.go for the rationale.

export type PlanningItemType = "REQ" | "DEC" | "Q";
export type PlanningItemStatus = "live" | "selected" | "dropped" | "promoted";

export interface PlanningSessionSummary {
  id: string;
  board_id: string;
  title: string;
  label: string | null;
  meeting_at: string | null;
  created_at: string;
  updated_at: string;
  req_count: number;
  dec_count: number;
  q_count: number;
  promoted_count: number;
  dropped_count: number;
}

export interface PlanningItem {
  id: string;
  session_id: string;
  type: PlanningItemType;
  title: string;
  description: string | null;
  status: PlanningItemStatus;
  promoted_to_card_id: string | null;
  position: number;
  created_at: string;
  // Free-text fields surfaced via the row's chevron expand. On promote, the
  // service copies these to the resulting card so the dev opening the card
  // sees the same context the requirement owner captured during planning.
  acceptance_criteria?: string | null;
  implementation_note?: string | null;
}

export interface PlanningSessionDetail {
  id: string;
  board_id: string;
  title: string;
  label: string | null;
  meeting_at: string | null;
  created_at: string;
  updated_at: string;
  items: PlanningItem[];
}

// Returned by GET /cards/:cardID/source. The handler responds with `null`
// (not 404) when a card wasn't promoted from planning, so the modal can
// render its "ที่มา" section conditionally without an error fork.
// One comment on a planning item's thread. Body is null on soft-deleted
// rows — the UI then renders italic "ถูกลบแล้ว" + the original author so
// the thread's position doesn't shift on delete.
export interface PlanningComment {
  id: string;
  item_id: string;
  author_id: string;
  author_name: string;
  body: string | null;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export interface CardSource {
  session: {
    id: string;
    title: string;
    label: string | null;
    meeting_at: string | null;
  };
  item: {
    id: string;
    type: PlanningItemType;
    title: string;
    status: PlanningItemStatus;
  };
  pending_questions: { id: string; title: string }[];
}
