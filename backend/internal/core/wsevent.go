package core

// WSEvent is the `type` tag on a board broadcast. It lives in core, not in the
// websocket package, so a handler can name an event without importing the hub —
// that import is what the Broadcaster interface exists to avoid.
type WSEvent string

// The events the REST write path broadcasts. Every value here must have a
// matching entry in frontend/src/types/wsEvents.ts; scripts/check-ws-events.mjs
// fails the build when the two drift.
const (
	WSCardCreated     WSEvent = "CARD_CREATED"
	WSCardMoved       WSEvent = "CARD_MOVED"
	WSCardUpdated     WSEvent = "CARD_UPDATED"
	WSCardDeleted     WSEvent = "CARD_DELETED"
	WSColumnCreated   WSEvent = "COLUMN_CREATED"
	WSColumnUpdated   WSEvent = "COLUMN_UPDATED"
	WSColumnDeleted   WSEvent = "COLUMN_DELETED"
	WSActivityCreated WSEvent = "ACTIVITY_CREATED"
)
