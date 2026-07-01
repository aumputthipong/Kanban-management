// Routes incoming WebSocket messages to the correct handler by type. Split from
// client.go (connection management) so each file has a single responsibility.
package websocket

import "log"

// WSMessage is the standard envelope for every client <-> server message.
type WSMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// dispatch routes an incoming message to the handler for its domain.
func (c *Client) dispatch(wsMsg WSMessage, rawMsg []byte) {
	switch wsMsg.Type {
	case "CARD_MOVED":
		c.handleCardMoved(wsMsg.Payload, rawMsg)
	case "CARD_CREATED":
		c.handleCardCreated(wsMsg.Payload)
	case "CARD_DELETED":
		c.handleCardDeleted(wsMsg.Payload, rawMsg)
	case "CARD_UPDATED":
		c.handleCardUpdated(wsMsg.Payload, rawMsg)
	case "CARD_DONE_TOGGLED":
		c.handleCardDoneToggled(wsMsg.Payload)
	case "COLUMN_CREATED":
		c.handleColumnCreated(wsMsg.Payload)
	case "COLUMN_RENAMED":
		c.handleColumnRenamed(wsMsg.Payload)
	case "COLUMN_DELETED":
		c.handleColumnDeleted(wsMsg.Payload)
	case "COLUMN_UPDATED":
		c.handleColumnUpdated(wsMsg.Payload)
	default:
		log.Printf("Unknown message type: %s", wsMsg.Type)
	}
}
