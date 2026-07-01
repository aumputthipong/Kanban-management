// Handlers for column-domain WS messages: each validates the payload, calls
// BoardCommandService, broadcasts, and records an activity.
package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/google/uuid"
)

func (c *Client) handleColumnCreated(payload map[string]interface{}) {
	title, ok := payload["title"].(string)
	if !ok || title == "" {
		log.Println("Invalid payload for COLUMN_CREATED")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Optional fields from the Create-column modal (quick path omits them →
	// service falls back to TODO + no colour).
	category, _ := payload["category"].(string)
	var color *string
	if cstr, ok := payload["color"].(string); ok && cstr != "" {
		color = &cstr
	}

	newCol, err := c.hub.boardCmd.CreateColumn(ctx, c.boardID, title, category, color)
	if err != nil {
		log.Printf("Failed to create column: %v", err)
		return
	}

	broadcastMsg := WSMessage{
		Type: "COLUMN_CREATED",
		Payload: map[string]interface{}{
			"id":       newCol.ID,
			"board_id": c.boardID,
			"title":    newCol.Title,
			"position": newCol.Position,
			"category": newCol.Category,
			"color":    newCol.Color,
		},
	}
	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal COLUMN_CREATED broadcast: %v", err)
		return
	}

	log.Printf("Created column [%s] '%s' at position [%f]", newCol.ID, newCol.Title, newCol.Position)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventColumnCreated, service.EntityColumn, strPtr(newCol.ID), service.ColumnCreatedPayload{
		Title: newCol.Title,
	})
}

func (c *Client) handleColumnRenamed(payload map[string]interface{}) {
	columnIDStr, ok1 := payload["column_id"].(string)
	title, ok2 := payload["title"].(string)
	if !ok1 || !ok2 || title == "" {
		log.Println("Invalid payload for COLUMN_RENAMED")
		return
	}
	if _, err := uuid.Parse(columnIDStr); err != nil {
		log.Printf("Invalid column ID: %s", columnIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		log.Printf("COLUMN_RENAMED rejected: column [%s] not in board [%s]", columnIDStr, c.boardID)
		return
	}

	if err := c.hub.boardCmd.RenameColumn(ctx, columnIDStr, title); err != nil {
		log.Printf("Failed to rename column [%s]: %v", columnIDStr, err)
		return
	}

	broadcastMsg := WSMessage{
		Type: "COLUMN_RENAMED",
		Payload: map[string]interface{}{
			"column_id": columnIDStr,
			"title":     title,
		},
	}
	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal COLUMN_RENAMED broadcast: %v", err)
		return
	}

	log.Printf("Renamed column [%s] to '%s'", columnIDStr, title)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventColumnRenamed, service.EntityColumn, strPtr(columnIDStr), service.ColumnRenamedPayload{
		NewTitle: title,
	})
}

func (c *Client) handleColumnDeleted(payload map[string]interface{}) {
	columnIDStr, ok := payload["column_id"].(string)
	if !ok {
		log.Println("Invalid payload for COLUMN_DELETED")
		return
	}
	if _, err := uuid.Parse(columnIDStr); err != nil {
		log.Printf("Invalid column ID: %s", columnIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		log.Printf("COLUMN_DELETED rejected: column [%s] not in board [%s]", columnIDStr, c.boardID)
		return
	}

	if err := c.hub.boardCmd.DeleteColumn(ctx, columnIDStr); err != nil {
		log.Printf("Failed to delete column [%s]: %v", columnIDStr, err)
		return
	}

	broadcastMsg := WSMessage{
		Type: "COLUMN_DELETED",
		Payload: map[string]interface{}{
			"column_id": columnIDStr,
		},
	}
	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal COLUMN_DELETED broadcast: %v", err)
		return
	}

	log.Printf("Deleted column [%s]", columnIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventColumnDeleted, service.EntityColumn, strPtr(columnIDStr), service.ColumnDeletedPayload{})
}

func (c *Client) handleColumnUpdated(payload map[string]interface{}) {
	columnIDStr, ok := payload["column_id"].(string)
	if !ok {
		log.Println("Invalid payload for COLUMN_UPDATED")
		return
	}
	if _, err := uuid.Parse(columnIDStr); err != nil {
		log.Printf("Invalid column ID: %s", columnIDStr)
		return
	}

	title, _ := payload["title"].(string)
	category, _ := payload["category"].(string)
	if title == "" || category == "" {
		log.Println("COLUMN_UPDATED: missing title or category")
		return
	}
	var colorPtr *string
	if colorVal, ok := payload["color"].(string); ok && colorVal != "" {
		colorPtr = &colorVal
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		log.Printf("COLUMN_UPDATED rejected: column [%s] not in board [%s]", columnIDStr, c.boardID)
		return
	}

	if err := c.hub.boardCmd.UpdateColumn(ctx, service.UpdateColumnParams{
		ID:       columnIDStr,
		Title:    title,
		Category: category,
		Color:    colorPtr,
	}); err != nil {
		log.Printf("Failed to update column [%s]: %v", columnIDStr, err)
		return
	}

	broadcastMsg := WSMessage{
		Type: "COLUMN_UPDATED",
		Payload: map[string]interface{}{
			"column_id": columnIDStr,
			"title":     title,
			"category":  category,
			"color":     colorPtr,
		},
	}
	msgBytes, err := json.Marshal(broadcastMsg)
	if err != nil {
		log.Printf("Failed to marshal COLUMN_UPDATED broadcast: %v", err)
		return
	}

	log.Printf("Updated column [%s]", columnIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}
}
