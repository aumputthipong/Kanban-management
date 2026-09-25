// Mirrors backend dto.Planning*. "DROP" is a status, not a fourth type.
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

// `body` is null on soft-deleted rows.
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

// null (not 404) when the card was not promoted.
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
