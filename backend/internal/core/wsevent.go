package core

// WSEvent lives in core so handlers can name an event without importing the hub.
type WSEvent string

// Mirrored in frontend/src/types/wsEvents.ts — `make check-ws-events` fails on drift.
const (
	WSCardCreated         WSEvent = "CARD_CREATED"
	WSCardMoved           WSEvent = "CARD_MOVED"
	WSCardUpdated         WSEvent = "CARD_UPDATED"
	WSCardDeleted         WSEvent = "CARD_DELETED"
	WSCardSubtasksUpdated WSEvent = "CARD_SUBTASKS_UPDATED"
	WSTagDeleted          WSEvent = "TAG_DELETED"
	WSColumnCreated       WSEvent = "COLUMN_CREATED"
	WSColumnUpdated       WSEvent = "COLUMN_UPDATED"
	WSColumnDeleted       WSEvent = "COLUMN_DELETED"
	WSBoardMembersUpdated WSEvent = "BOARD_MEMBERS_UPDATED"
	WSActivityCreated     WSEvent = "ACTIVITY_CREATED"
)
