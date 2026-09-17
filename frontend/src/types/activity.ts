export type ActivityEventType =
  | "card.created"
  | "card.moved"
  | "card.updated"
  | "card.deleted"
  | "card.done_toggled"
  | "column.created"
  | "column.deleted"
  | "column.renamed"
  | "card.subtasks_completed"
  | "member.added"
  | "member.removed"
  | "member.left"
  | "member.role_changed";

export interface Activity {
  id: string;
  board_id: string;
  actor_id: string;
  actor_name: string | null;
  event_type: ActivityEventType | string;
  entity_type: "card" | "column" | "member" | string;
  entity_id: string | null;
  payload: Record<string, unknown>;
  created_at: string;
}
