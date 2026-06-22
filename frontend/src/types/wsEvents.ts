// WebSocket envelope message types — the `type` tag the server routes board
// mutations by. These are UPPER_SNAKE and are DISTINCT from activity
// `event_type` strings (dotted lower-case, e.g. "card.moved") which travel
// inside the ACTIVITY_CREATED payload and mirror the Go activity constants.
//
// Centralised so a comparison typo fails at compile time instead of silently
// no-op'ing: a bare `parsedData.type === "CARD_MVOED"` would just never match
// and the mutation would be dropped with no error. Referencing `WS_EVENT.*`
// makes the same typo a TypeScript error.
//
// Keep in sync with the backend broadcast `type` tags (see internal/dto/*_dto.go
// and the websocket dispatcher).
export const WS_EVENT = {
  CardMoved: "CARD_MOVED",
  CardCreated: "CARD_CREATED",
  CardDeleted: "CARD_DELETED",
  CardUpdated: "CARD_UPDATED",
  // Client -> server command (no consumer branch in useWebSocket): the server
  // applies it and broadcasts the resulting CARD_UPDATED / CARD_MOVED back.
  CardDoneToggled: "CARD_DONE_TOGGLED",
  ColumnCreated: "COLUMN_CREATED",
  ColumnRenamed: "COLUMN_RENAMED",
  ColumnDeleted: "COLUMN_DELETED",
  ColumnUpdated: "COLUMN_UPDATED",
  ActivityCreated: "ACTIVITY_CREATED",
} as const;

export type WsEventType = (typeof WS_EVENT)[keyof typeof WS_EVENT];
