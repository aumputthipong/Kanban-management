// Mirrors backend core/wsevent.go — `make check-ws-events` fails on drift (docs/adr/0007).
export const WS_EVENT = {
  CardMoved: "CARD_MOVED",
  CardCreated: "CARD_CREATED",
  CardDeleted: "CARD_DELETED",
  CardUpdated: "CARD_UPDATED",
  CardSubtasksUpdated: "CARD_SUBTASKS_UPDATED",
  TagDeleted: "TAG_DELETED",
  ColumnCreated: "COLUMN_CREATED",
  ColumnDeleted: "COLUMN_DELETED",
  ColumnUpdated: "COLUMN_UPDATED",
  BoardMembersUpdated: "BOARD_MEMBERS_UPDATED",
  ActivityCreated: "ACTIVITY_CREATED",
} as const;

export type WsEventType = (typeof WS_EVENT)[keyof typeof WS_EVENT];
