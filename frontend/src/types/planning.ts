// Mirrors backend dto.PlanningSession* / PlanningItemResponse. The three item types
// answer what we want, what we decided, and what we still do not know. "DROP" is a
// status on an item, not a fourth type.

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
  // Copied onto the resulting card on promote, so the dev sees the context the
  // requirement owner captured during planning.
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

// One comment on an item thread. `body` is null on soft-deleted rows; the UI shows an
// italic placeholder plus the original author so the thread does not shift.
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

// Returned by GET /cards/:cardID/source. The handler responds with null (not 404)
// when the card was not promoted, so the modal renders the section without a fork.
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
