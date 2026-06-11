// internal/service/board_command_service.go
//
// BoardCommandService รวม write-operations ที่ถูกเรียกจาก WebSocket layer:
// การย้าย/สร้าง/ลบ/อัปเดต card, การจัดการ column, และ subtask write ops.
// แยกออกจาก BoardService เพื่อไม่ให้ BoardService ใหญ่เกินไป
// และเพื่อให้ WS handlers ไม่ต้องผูกกับ *db.Queries โดยตรง
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

const wsPositionGap = 65536.0

type BoardCommandService struct {
	queries *db.Queries
}

func NewBoardCommandService(queries *db.Queries) *BoardCommandService {
	return &BoardCommandService{queries: queries}
}

// ErrEntityBoardMismatch is returned when a WS handler tries to mutate a card
// or column that does not belong to the board the client is connected to.
// This is a defense-in-depth check on top of route-level board membership —
// without it, a member of board A could send WS messages that mutate cards in
// board B by referencing their UUIDs.
var ErrEntityBoardMismatch = errors.New("entity does not belong to this board")

// VerifyCardInBoard returns nil iff the given card belongs to boardID.
// Returns ErrEntityBoardMismatch for cross-board attempts (whether the card
// is in a different board or simply does not exist) so callers cannot
// distinguish "wrong board" from "no such card".
func (s *BoardCommandService) VerifyCardInBoard(ctx context.Context, cardID, boardID string) error {
	owner, err := s.queries.GetBoardIDByCard(ctx, cardID)
	if err != nil {
		return ErrEntityBoardMismatch
	}
	if owner != boardID {
		return ErrEntityBoardMismatch
	}
	return nil
}

// VerifyColumnInBoard returns nil iff the given column belongs to boardID.
// See VerifyCardInBoard for the rationale on collapsing the error cases.
func (s *BoardCommandService) VerifyColumnInBoard(ctx context.Context, columnID, boardID string) error {
	owner, err := s.queries.GetBoardIDByColumn(ctx, columnID)
	if err != nil {
		return ErrEntityBoardMismatch
	}
	if owner != boardID {
		return ErrEntityBoardMismatch
	}
	return nil
}

// -----------------------------
// Card operations
// -----------------------------

type MoveCardResult struct {
	CardTitle   string
	IsDone      bool
	CompletedAt *time.Time
}

func (s *BoardCommandService) MoveCard(ctx context.Context, cardID, newColumnID string, position float64) (MoveCardResult, error) {
	category, err := s.queries.GetColumnCategory(ctx, newColumnID)
	if err != nil {
		return MoveCardResult{}, fmt.Errorf("get column category: %w", err)
	}
	isDone := category == "DONE"
	var completedAt *time.Time
	if isDone {
		now := time.Now()
		completedAt = &now
	}
	if err := s.queries.UpdateCardColumn(ctx, db.UpdateCardColumnParams{
		ColumnID:    newColumnID,
		Position:    position,
		IsDone:      isDone,
		CompletedAt: util.TimeToTimestamptz(completedAt),
		ID:          cardID,
	}); err != nil {
		return MoveCardResult{}, fmt.Errorf("update card column: %w", err)
	}
	var title string
	if card, err := s.queries.GetCard(ctx, cardID); err == nil {
		title = card.Title
	}
	return MoveCardResult{CardTitle: title, IsDone: isDone, CompletedAt: completedAt}, nil
}

// CreateCardWS — สร้าง card ผ่าน WS flow; คำนวณ position ให้ถ้า frontend ไม่ส่งมา.
// ใช้ชื่อ WS เพื่อไม่ชนกับ BoardService.CreateCard ที่มีอยู่แล้ว
// assigneeID / dueDate เป็น optional (nil = ไม่ระบุ) — รองรับ Create Task modal
// ที่สร้าง card พร้อม assignee + due date ในครั้งเดียว (quick-add ส่ง nil).
func (s *BoardCommandService) CreateCardWS(ctx context.Context, columnID, creatorID, title, priority string, position float64, assigneeID, dueDate *string) (db.CreateCardRow, error) {
	if position <= 0 {
		maxPos, err := s.queries.GetMaxPositionInColumn(ctx, columnID)
		if err == nil {
			if v, ok := maxPos.(float64); ok {
				position = v + wsPositionGap
			} else {
				position = wsPositionGap
			}
		} else {
			position = wsPositionGap
		}
	}
	return s.queries.CreateCard(ctx, db.CreateCardParams{
		ColumnID:   columnID,
		Title:      title,
		Position:   position,
		Priority:   util.StringToPtr(priority),
		AssigneeID: assigneeID,
		DueDate:    util.PtrStringToTimePtr(dueDate),
		CreatedBy:  &creatorID,
	})
}

// DeleteCard — คืน title ของ card ก่อนลบ เพื่อใช้เขียน activity
func (s *BoardCommandService) DeleteCard(ctx context.Context, cardID string) (string, error) {
	var title string
	if card, err := s.queries.GetCard(ctx, cardID); err == nil {
		title = card.Title
	}
	if err := s.queries.DeleteCard(ctx, cardID); err != nil {
		return "", fmt.Errorf("delete card: %w", err)
	}
	return title, nil
}

// UpdateCardBasicParams — field set ที่ WS layer update (ไม่รวม tags)
type UpdateCardBasicParams struct {
	ID             string
	Title          string
	Description    string
	DueDate        string
	AssigneeID     string
	Priority       string
	EstimatedHours float64
}

func (s *BoardCommandService) UpdateCardBasic(ctx context.Context, p UpdateCardBasicParams) error {
	if _, err := s.queries.UpdateCard(ctx, db.UpdateCardParams{
		ID:             p.ID,
		Title:          p.Title,
		Description:    util.StringToPtr(p.Description),
		DueDate:        util.StringToTimePtr(p.DueDate),
		AssigneeID:     util.StringToPtr(p.AssigneeID),
		Priority:       util.StringToPtr(p.Priority),
		EstimatedHours: util.FloatToPgNumeric(p.EstimatedHours),
	}); err != nil {
		return fmt.Errorf("update card: %w", err)
	}
	return nil
}

type ToggleCardDoneResult struct {
	TargetColumnID string
	CardTitle      string
	CompletedAt    *time.Time
}

func (s *BoardCommandService) ToggleCardDone(ctx context.Context, cardID, boardID string, isDone bool) (ToggleCardDoneResult, error) {
	targetCategory := "TODO"
	if isDone {
		targetCategory = "DONE"
	}
	targetCol, err := s.queries.GetColumnByBoardAndCategory(ctx, db.GetColumnByBoardAndCategoryParams{
		BoardID:  boardID,
		Category: targetCategory,
	})
	if err != nil {
		return ToggleCardDoneResult{}, fmt.Errorf("find %s column: %w", targetCategory, err)
	}
	var completedAt *time.Time
	if isDone {
		now := time.Now()
		completedAt = &now
	}
	if err := s.queries.UpdateCardColumn(ctx, db.UpdateCardColumnParams{
		ColumnID:    targetCol.ID,
		Position:    0,
		IsDone:      isDone,
		CompletedAt: util.TimeToTimestamptz(completedAt),
		ID:          cardID,
	}); err != nil {
		return ToggleCardDoneResult{}, fmt.Errorf("toggle card done: %w", err)
	}
	var title string
	if card, err := s.queries.GetCard(ctx, cardID); err == nil {
		title = card.Title
	}
	return ToggleCardDoneResult{
		TargetColumnID: targetCol.ID,
		CardTitle:      title,
		CompletedAt:    completedAt,
	}, nil
}

// -----------------------------
// Column operations
// -----------------------------

// CreateColumn — insert column ใหม่ก่อน DONE column ถ้ามี.
// category + color มาจาก Create-column modal (quick path ส่ง "TODO" + nil).
func (s *BoardCommandService) CreateColumn(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error) {
	if category != "DONE" {
		category = "TODO"
	}
	doneCol, err := s.queries.GetColumnByBoardAndCategory(ctx, db.GetColumnByBoardAndCategoryParams{
		BoardID:  boardID,
		Category: "DONE",
	})

	var position float64
	if err != nil {
		// ไม่มี DONE column → ต่อท้ายปกติ
		maxPos, _ := s.queries.GetMaxColumnPositionInBoard(ctx, boardID)
		if v, ok := maxPos.(float64); ok {
			position = v + wsPositionGap
		} else {
			position = wsPositionGap
		}
	} else {
		prevPos, _ := s.queries.GetMaxColumnPositionBeforeDone(ctx, boardID)
		prevPosF, _ := prevPos.(float64)
		position = (prevPosF + doneCol.Position) / 2
	}

	return s.queries.CreateColumn(ctx, db.CreateColumnParams{
		BoardID:  boardID,
		Title:    title,
		Position: position,
		Category: category,
		Color:    color,
	})
}

func (s *BoardCommandService) RenameColumn(ctx context.Context, columnID, title string) error {
	if err := s.queries.RenameColumn(ctx, db.RenameColumnParams{ID: columnID, Title: title}); err != nil {
		return fmt.Errorf("rename column: %w", err)
	}
	return nil
}

func (s *BoardCommandService) DeleteColumn(ctx context.Context, columnID string) error {
	if err := s.queries.DeleteColumn(ctx, columnID); err != nil {
		return fmt.Errorf("delete column: %w", err)
	}
	return nil
}

type UpdateColumnParams struct {
	ID       string
	Title    string
	Category string
	Color    *string
}

func (s *BoardCommandService) UpdateColumn(ctx context.Context, p UpdateColumnParams) error {
	if err := s.queries.UpdateColumn(ctx, db.UpdateColumnParams{
		ID:       p.ID,
		Title:    p.Title,
		Category: p.Category,
		Color:    p.Color,
	}); err != nil {
		return fmt.Errorf("update column: %w", err)
	}
	return nil
}

// -----------------------------
// Subtask operations (WS write flow)
// -----------------------------

type UpdateSubtaskWSParams struct {
	ID       string
	Title    string
	IsDone   bool
	Position float64
}

func (s *BoardCommandService) UpdateSubtaskAll(ctx context.Context, p UpdateSubtaskWSParams) error {
	if _, err := s.queries.UpdateSubtask(ctx, db.UpdateSubtaskParams{
		ID:       p.ID,
		Title:    p.Title,
		IsDone:   p.IsDone,
		Position: p.Position,
	}); err != nil {
		return fmt.Errorf("update subtask: %w", err)
	}
	return nil
}

func (s *BoardCommandService) ToggleSubtaskDone(ctx context.Context, subtaskID string, isDone bool) error {
	return s.queries.UpdateSubtaskDone(ctx, db.UpdateSubtaskDoneParams{
		ID:     subtaskID,
		IsDone: isDone,
	})
}
