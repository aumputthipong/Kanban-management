export interface Tag {
  id: string;
  board_id: string;
  name: string;
  color: string;
}

export interface Card {
  id: string;
  column_id: string;
  title: string;
  position: number;
  description: string | null;
  due_date: string | null;
  assignee_id: string | null;
  assignee_name: string | null;
  priority: "low" | "medium" | "high" | null;
  estimated_hours: number | null;
  is_done: boolean;
  completed_at: string | null;
  created_at: string | null;
  created_by: string | null;
  total_subtasks: number; // from COUNT
  completed_subtasks: number; // from COUNT
  subtasks?: Subtask[];
  tags?: Tag[];
  // null = never set; "" = explicitly cleared.
  acceptance_criteria?: string | null;
  implementation_note?: string | null;
}

export interface Column {
  id: string;
  title: string;
  position: number;
  category: "TODO" | "DONE";
  color?: string | null;
  cards: Card[];
}

export interface Board {
  id: string;
  title: string;
  budget?: number;
  /** Short board description shown on project-list cards. "" = none. */
  description?: string;
  /** Accent colour (hex) for the board glyph / card bar / sidebar dot. */
  color?: string;
  /** Glyph key — one of lib/boardAppearance BOARD_ICON keys. */
  icon?: string;
  created_at: string;
  updated_at: string;
  /** Null for memberships older than the tracking column. */
  last_accessed_at?: string | null;
  total_cards: number;
  done_cards: number;
  members: { user_id: string; full_name: string }[];
}

export interface CreateCardPayload {
  column_id: string;
  title: string;
}

export interface BoardMember {
  id: string;
  role: "owner" | "manager" | "member";
  user_id: string;
  email: string;
  full_name: string;
}

export interface User {
  id: string;
  email: string;
  full_name: string;
}

export interface Subtask {
  id: string;
  card_id: string;
  title: string;
  is_done: boolean;
  position: number;
}

export interface CardUpdateForm {
  title: string;
  description: string;
  due_date: string;
  assignee_id: string;
  priority: string;
  estimated_hours: string;
  tags: Tag[];
  // "" in form state (textareas hate undefined), sent as "" when cleared.
  acceptance_criteria: string;
  implementation_note: string;
}