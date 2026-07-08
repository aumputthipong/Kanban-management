// Handlers for column-domain WS messages: each validates the payload, calls
// BoardCommandService, broadcasts, and records an activity.
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/google/uuid"
)

func (c *Client) handleColumnCreated(payload map[string]interface{}) {
	title, ok := payload["title"].(string)
	if !ok || title == "" {
		slog.Warn("invalid COLUMN_CREATED payload", "board_id", c.boardID)
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
		slog.Error("create column failed", "board_id", c.boardID, "err", err)
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
		slog.Error("marshal COLUMN_CREATED broadcast failed", "err", err)
		return
	}

	slog.Debug("column created", "column_id", newCol.ID, "position", newCol.Position)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventColumnCreated, service.EntityColumn, strPtr(newCol.ID), service.ColumnCreatedPayload{
		Title: newCol.Title,
	})
}

func (c *Client) handleColumnRenamed(payload map[string]interface{}) {
	columnIDStr, ok1 := payload["column_id"].(string)
	title, ok2 := payload["title"].(string)
	if !ok1 || !ok2 || title == "" {
		slog.Warn("invalid COLUMN_RENAMED payload", "board_id", c.boardID)
		return
	}
	if _, err := uuid.Parse(columnIDStr); err != nil {
		slog.Warn("invalid column id", "column_id", columnIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		slog.Warn("COLUMN_RENAMED rejected: column not in board", "column_id", columnIDStr, "board_id", c.boardID)
		return
	}

	if err := c.hub.boardCmd.RenameColumn(ctx, columnIDStr, title); err != nil {
		slog.Error("rename column failed", "column_id", columnIDStr, "err", err)
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
		slog.Error("marshal COLUMN_RENAMED broadcast failed", "err", err)
		return
	}

	slog.Debug("column renamed", "column_id", columnIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventColumnRenamed, service.EntityColumn, strPtr(columnIDStr), service.ColumnRenamedPayload{
		NewTitle: title,
	})
}

func (c *Client) handleColumnDeleted(payload map[string]interface{}) {
	columnIDStr, ok := payload["column_id"].(string)
	if !ok {
		slog.Warn("invalid COLUMN_DELETED payload", "board_id", c.boardID)
		return
	}
	if _, err := uuid.Parse(columnIDStr); err != nil {
		slog.Warn("invalid column id", "column_id", columnIDStr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		slog.Warn("COLUMN_DELETED rejected: column not in board", "column_id", columnIDStr, "board_id", c.boardID)
		return
	}

	if err := c.hub.boardCmd.DeleteColumn(ctx, columnIDStr); err != nil {
		slog.Error("delete column failed", "column_id", columnIDStr, "err", err)
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
		slog.Error("marshal COLUMN_DELETED broadcast failed", "err", err)
		return
	}

	slog.Debug("column deleted", "column_id", columnIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}

	c.recordActivity(ctx, service.EventColumnDeleted, service.EntityColumn, strPtr(columnIDStr), service.ColumnDeletedPayload{})
}

func (c *Client) handleColumnUpdated(payload map[string]interface{}) {
	columnIDStr, ok := payload["column_id"].(string)
	if !ok {
		slog.Warn("invalid COLUMN_UPDATED payload", "board_id", c.boardID)
		return
	}
	if _, err := uuid.Parse(columnIDStr); err != nil {
		slog.Warn("invalid column id", "column_id", columnIDStr)
		return
	}

	title, _ := payload["title"].(string)
	category, _ := payload["category"].(string)
	if title == "" || category == "" {
		slog.Warn("COLUMN_UPDATED missing title or category", "column_id", columnIDStr)
		return
	}
	var colorPtr *string
	if colorVal, ok := payload["color"].(string); ok && colorVal != "" {
		colorPtr = &colorVal
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := c.hub.boardCmd.VerifyColumnInBoard(ctx, columnIDStr, c.boardID); err != nil {
		slog.Warn("COLUMN_UPDATED rejected: column not in board", "column_id", columnIDStr, "board_id", c.boardID)
		return
	}

	if err := c.hub.boardCmd.UpdateColumn(ctx, service.UpdateColumnParams{
		ID:       columnIDStr,
		Title:    title,
		Category: category,
		Color:    colorPtr,
	}); err != nil {
		slog.Error("update column failed", "column_id", columnIDStr, "err", err)
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
		slog.Error("marshal COLUMN_UPDATED broadcast failed", "err", err)
		return
	}

	slog.Debug("column updated", "column_id", columnIDStr)
	c.hub.broadcast <- BroadcastMessage{BoardID: c.boardID, Message: msgBytes}
}
