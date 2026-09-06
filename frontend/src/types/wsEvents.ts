// The `type` tag the server routes board mutations by. UPPER_SNAKE, and DISTINCT
// from activity `event_type` strings (dotted lower-case) that travel inside an
// ACTIVITY_CREATED payload. Keep in sync with the backend broadcast tags.
export const WS_EVENT = {
  CardMoved: "CARD_MOVED",
  CardCreated: "CARD_CREATED",
  CardDeleted: "CARD_DELETED",
  CardUpdated: "CARD_UPDATED",
  // Client -> server only: the server applies it and broadcasts the result back.
  CardDoneToggled: "CARD_DONE_TOGGLED",
  ColumnCreated: "COLUMN_CREATED",
  ColumnRenamed: "COLUMN_RENAMED",
  ColumnDeleted: "COLUMN_DELETED",
  ColumnUpdated: "COLUMN_UPDATED",
  ActivityCreated: "ACTIVITY_CREATED",
} as const;

export type WsEventType = (typeof WS_EVENT)[keyof typeof WS_EVENT];
