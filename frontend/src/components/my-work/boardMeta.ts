// board_id → visual identity (accent + glyph key), resolved from the boards
// list and consumed by the project containers in DashboardGrid so My Work
// mirrors the sidebar/header look. Kept tiny + standalone so both the page and
// the grid can share the type without a component import.
export interface BoardMeta {
  color?: string;
  icon?: string;
}
