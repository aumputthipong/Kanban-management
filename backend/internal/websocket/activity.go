// Helper that records an activity and broadcasts ACTIVITY_CREATED to every
// client in the room. A failed record is only logged, not fatal to the flow —
// the activity feed is a secondary concern.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/google/uuid"
)

func (c *Client) recordActivity(ctx context.Context, eventType, entityType string, entityID *string, payload any) {
	if c.hub.activities == nil {
		return
	}
	if _, err := uuid.Parse(c.userID); err != nil {
		return
	}
	act, err := c.hub.activities.Record(ctx, service.RecordParams{
		BoardID:    c.boardID,
		ActorID:    c.userID,
		EventType:  eventType,
		EntityType: entityType,
		EntityID:   entityID,
		Payload:    payload,
	})
	if err != nil {
		log.Printf("Failed to record activity [%s]: %v", eventType, err)
		return
	}

	var payloadRaw json.RawMessage = act.Payload
	if len(payloadRaw) == 0 {
		payloadRaw = json.RawMessage("{}")
	}

	broadcastPayload := map[string]interface{}{
		"id":          act.ID,
		"board_id":    act.BoardID,
		"actor_id":    act.ActorID,
		"event_type":  act.EventType,
		"entity_type": act.EntityType,
		"entity_id":   act.EntityID,
		"payload":     payloadRaw,
		"created_at":  act.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	msg := WSMessage{Type: "ACTIVITY_CREATED", Payload: broadcastPayload}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal ACTIVITY_CREATED: %v", err)
		return
	}
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}
}

// strPtr returns a pointer to s, for passing entity IDs into recordActivity.
func strPtr(s string) *string { return &s }
