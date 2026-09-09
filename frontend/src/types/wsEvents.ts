// The `type` tag on a board broadcast. UPPER_SNAKE, and DISTINCT from activity
// `event_type` strings (dotted lower-case) that travel inside an ACTIVITY_CREATED
// payload. Mirrors backend/internal/core/wsevent.go; scripts/check-ws-events.mjs
// fails the build when the two drift.
export const WS_EVENT = {
  CardMoved: "CARD_MOVED",
  CardCreated: "CARD_CREATED",
  CardDeleted: "CARD_DELETED",
  CardUpdated: "CARD_UPDATED",
  ColumnCreated: "COLUMN_CREATED",
  ColumnDeleted: "COLUMN_DELETED",
  ColumnUpdated: "COLUMN_UPDATED",
  ActivityCreated: "ACTIVITY_CREATED",
} as const;

export type WsEventType = (typeof WS_EVENT)[keyof typeof WS_EVENT];
