// Handlers for card-domain WS messages. Each handler only validates the
// payload, calls BoardCommandService, composes the broadcast, and records an
// activity; all business logic (isDone derivation, position calc, etc.) lives
// in the service layer.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/google/uuid"
)

func (c *Client) handleCardMoved(payload map[string]interface{}, rawMsg []byte) {
	_ = rawMsg // not re-used — server re-composes broadcast because isDone/completedAt are server-computed
	cardIDStr, ok1 := payload["card_id"].(string)
	newColumnIDStr, ok2 := payload["new_column_id"].(string)
	position, ok3 := payload["position"].(float64)
	if !ok1 || !ok2 || !ok3 {
		log.Println("Invalid payload for CARD_MOVED")
		return
	}

	if _, err := uuid.Parse(cardIDStr); err != nil {
		log.Printf("Invalid card ID: %s", cardIDStr)
		return
	}
	if _, err := uuid.Parse(newColumnIDStr); err != nil {
		log.Printf("Invalid column ID: %s", newColumnIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyCardInBoard(ctx, cardIDStr, c.boardID); err != nil {
		log.Printf("CARD_MOVED rejected: card [%s] not in board [%s]", cardIDStr, c.boardID)
		return
	}
	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, newColumnIDStr, c.boardID); err != nil {
		log.Printf("CARD_MOVED rejected: column [%s] not in board [%s]", newColumnIDStr, c.boardID)
		return
	}

	result, err := c.hub.boardCmd.MoveCard(ctx, cardIDStr, newColumnIDStr, position)
	if err != nil {
		log.Printf("Failed to move card: %v", err)
		return
	}

	broadcastMsg := WSMessage{
		Type: "CARD_MOVED",
		Payload: map[string]interface{}{
			"card_id":       cardIDStr,
			"new_column_id": newColumnIDStr,
			"position":      position,
			"is_done":       result.IsDone,
			"completed_at":  result.CompletedAt,
		},
	}
	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal CARD_MOVED broadcast: %v", err)
		return
	}

	log.Printf("Moved card [%s] to column [%s] at position [%f] (isDone=%v)", cardIDStr, newColumnIDStr, position, result.IsDone)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventCardMoved, service.EntityCard, strPtr(cardIDStr), service.CardMovedPayload{
		Title:      result.CardTitle,
		ToColumnID: newColumnIDStr,
	})
}

func (c *Client) handleCardCreated(payload map[string]interface{}) {
	columnIDStr, ok1 := payload["column_id"].(string)
	title, ok2 := payload["title"].(string)
	if !ok1 || !ok2 {
		log.Println("Invalid payload for CARD_CREATED")
		return
	}

	if _, err := uuid.Parse(columnIDStr); err != nil {
		log.Printf("Invalid column ID: %s", columnIDStr)
		return
	}
	if _, err := uuid.Parse(c.userID); err != nil {
		log.Printf("Invalid creator ID: %s", c.userID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		log.Printf("CARD_CREATED rejected: column [%s] not in board [%s]", columnIDStr, c.boardID)
		return
	}

	priority, _ := payload["priority"].(string)
	position, _ := payload["position"].(float64)

	// Optional fields from the Create Task modal (quick-add omits them).
	var assigneeID *string
	if a, ok := payload["assignee_id"].(string); ok && a != "" {
		if _, err := uuid.Parse(a); err != nil {
			log.Printf("Invalid assignee ID: %s", a)
			return
		}
		assigneeID = &a
	}
	var dueDate *string
	if d, ok := payload["due_date"].(string); ok && d != "" {
		dueDate = &d
	}
	var description *string
	if d, ok := payload["description"].(string); ok && d != "" {
		description = &d
	}
	// Subtasks come from the modal's optional checklist — titles only; the card
	// id doesn't exist yet, so they're created alongside the card server-side.
	subtaskTitles := parseSubtaskTitles(payload["subtasks"])

	newCard, subtasks, err := c.hub.boardCmd.CreateCardWS(ctx, columnIDStr, c.userID, title, priority, position, assigneeID, dueDate, description, subtaskTitles)
	if err != nil {
		log.Printf("Failed to create card: %v", err)
		return
	}

	// Subtasks shaped for the frontend store (matches types/board Subtask).
	subPayload := make([]map[string]interface{}, 0, len(subtasks))
	for _, st := range subtasks {
		subPayload = append(subPayload, map[string]interface{}{
			"id":       st.ID,
			"card_id":  st.CardID,
			"title":    st.Title,
			"is_done":  st.IsDone,
			"position": st.Position,
		})
	}

	broadcastMsg := WSMessage{
		Type: "CARD_CREATED",
		Payload: map[string]interface{}{
			"id":                 newCard.ID,
			"column_id":          newCard.ColumnID,
			"title":              newCard.Title,
			"position":           newCard.Position,
			"priority":           newCard.Priority,
			"created_by":         newCard.CreatedBy,
			"assignee_id":        assigneeID,
			"due_date":           dueDate,
			"description":        description,
			"subtasks":           subPayload,
			"total_subtasks":     len(subPayload),
			"completed_subtasks": 0,
		},
	}

	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal CARD_CREATED broadcast: %v", err)
		return
	}

	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventCardCreated, service.EntityCard, strPtr(newCard.ID), service.CardCreatedPayload{
		Title:    newCard.Title,
		ColumnID: newCard.ColumnID,
	})
}

// parseSubtaskTitles normalises the modal's optional "subtasks" payload (a JSON
// array of strings) into clean titles: trimmed, empties dropped, each capped at
// 200 chars (mirrors UpdateSubtaskRequest's max) and the list capped at 50 so a
// malformed client can't insert an unbounded batch in one card create.
func parseSubtaskTitles(raw interface{}) []string {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	const maxSubtasks = 50
	const maxLen = 200
	titles := make([]string, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if len(s) > maxLen {
			s = s[:maxLen]
		}
		titles = append(titles, s)
		if len(titles) >= maxSubtasks {
			break
		}
	}
	return titles
}

func (c *Client) handleCardDeleted(payload map[string]interface{}, rawMsg []byte) {
	cardIDStr, ok := payload["card_id"].(string)
	if !ok {
		log.Println("Invalid payload for CARD_DELETED")
		return
	}

	if _, err := uuid.Parse(cardIDStr); err != nil {
		log.Printf("Invalid card ID: %s", cardIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyCardInBoard(ctx, cardIDStr, c.boardID); err != nil {
		log.Printf("CARD_DELETED rejected: card [%s] not in board [%s]", cardIDStr, c.boardID)
		return
	}

	cardTitle, err := c.hub.boardCmd.DeleteCard(ctx, cardIDStr)
	if err != nil {
		log.Printf("Failed to delete card: %v", err)
		return
	}

	log.Printf("Deleted card [%s]", cardIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: rawMsg}

	c.recordActivity(ctx, service.EventCardDeleted, service.EntityCard, strPtr(cardIDStr), service.CardDeletedPayload{
		Title: cardTitle,
	})
}

func (c *Client) handleCardUpdated(payload map[string]interface{}, rawMsg []byte) {
	cardIDStr, ok := payload["card_id"].(string)
	if !ok {
		log.Println("Invalid payload for CARD_UPDATED")
		return
	}

	title, _ := payload["title"].(string)
	description, _ := payload["description"].(string)
	dueDate, _ := payload["due_date"].(string)
	assigneeID, _ := payload["assignee_id"].(string)
	priority, _ := payload["priority"].(string)
	estimatedHours, _ := payload["estimated_hours"].(float64)

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyCardInBoard(ctx, cardIDStr, c.boardID); err != nil {
		log.Printf("CARD_UPDATED rejected: card [%s] not in board [%s]", cardIDStr, c.boardID)
		return
	}

	if err := c.hub.boardCmd.UpdateCardBasic(ctx, service.UpdateCardBasicParams{
		ID:             cardIDStr,
		Title:          title,
		Description:    description,
		DueDate:        dueDate,
		AssigneeID:     assigneeID,
		Priority:       priority,
		EstimatedHours: estimatedHours,
	}); err != nil {
		log.Printf("Failed to update card [%s]: %v", cardIDStr, err)
		return
	}

	log.Printf("Updated card [%s] successfully", cardIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: rawMsg}

	// Use the client's changed_fields; an empty list means nothing changed, so
	// skip the activity record.
	var fields []string
	if raw, ok := payload["changed_fields"].([]interface{}); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				fields = append(fields, s)
			}
		}
	}
	if len(fields) == 0 {
		return
	}
	c.recordActivity(ctx, service.EventCardUpdated, service.EntityCard, strPtr(cardIDStr), service.CardUpdatedPayload{
		Title:  title,
		Fields: fields,
	})
}

func (c *Client) handleCardDoneToggled(payload map[string]interface{}) {
	cardIDStr, ok1 := payload["card_id"].(string)
	boardIDStr, ok2 := payload["board_id"].(string)
	isDone, ok3 := payload["is_done"].(bool)
	if !ok1 || !ok2 || !ok3 {
		log.Println("Invalid payload for CARD_DONE_TOGGLED")
		return
	}
	if _, err := uuid.Parse(cardIDStr); err != nil {
		log.Printf("Invalid card ID: %s", cardIDStr)
		return
	}
	if _, err := uuid.Parse(boardIDStr); err != nil {
		log.Printf("Invalid board ID: %s", boardIDStr)
		return
	}
	// Reject mismatched board_id in payload outright — the client must operate
	// on the board it connected to. Pinning to c.boardID also closes the
	// payload-injection path where an attacker passes someone else's board.
	if boardIDStr != c.boardID {
		log.Printf("CARD_DONE_TOGGLED rejected: payload board [%s] != connection board [%s]", boardIDStr, c.boardID)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyCardInBoard(ctx, cardIDStr, c.boardID); err != nil {
		log.Printf("CARD_DONE_TOGGLED rejected: card [%s] not in board [%s]", cardIDStr, c.boardID)
		return
	}

	result, err := c.hub.boardCmd.ToggleCardDone(ctx, cardIDStr, boardIDStr, isDone)
	if err != nil {
		log.Printf("Failed to toggle card done: %v", err)
		return
	}

	// Broadcast as CARD_MOVED so the frontend reuses the same handler.
	broadcastMsg := WSMessage{
		Type: "CARD_MOVED",
		Payload: map[string]interface{}{
			"card_id":       cardIDStr,
			"new_column_id": result.TargetColumnID,
			"position":      0,
			"is_done":       isDone,
			"completed_at":  result.CompletedAt,
		},
	}
	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal CARD_DONE_TOGGLED broadcast: %v", err)
		return
	}

	log.Printf("Toggled card [%s] done=%v → column [%s]", cardIDStr, isDone, result.TargetColumnID)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventCardDoneToggled, service.EntityCard, strPtr(cardIDStr), service.CardDoneToggledPayload{
		Title:  result.CardTitle,
		IsDone: isDone,
	})
}
