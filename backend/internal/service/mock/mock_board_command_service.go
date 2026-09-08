package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// MockBoardCommandService implements service.BoardCommandServicer with the
// function-field pattern: a test sets only the methods it exercises.
type MockBoardCommandService struct {
	VerifyCardInBoardFn   func(ctx context.Context, cardID, boardID string) error
	CreateCardWSFn        func(ctx context.Context, columnID, creatorID, title, priority string, position float64, assigneeID, dueDate, description *string, subtaskTitles []string) (db.CreateCardRow, []db.CardSubtask, error)
	VerifyColumnInBoardFn func(ctx context.Context, columnID, boardID string) error
	MoveCardFn            func(ctx context.Context, cardID, newColumnID string, position float64) (service.MoveCardResult, error)
	DeleteCardFn          func(ctx context.Context, cardID string) (string, error)
	ToggleCardDoneFn      func(ctx context.Context, cardID, boardID string, isDone bool) (service.ToggleCardDoneResult, error)
	CreateColumnFn        func(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error)
	DeleteColumnFn        func(ctx context.Context, columnID string) error
	UpdateColumnFn        func(ctx context.Context, p service.UpdateColumnParams) error
}

func (m *MockBoardCommandService) VerifyCardInBoard(ctx context.Context, cardID, boardID string) error {
	return m.VerifyCardInBoardFn(ctx, cardID, boardID)
}

func (m *MockBoardCommandService) CreateCardWS(ctx context.Context, columnID, creatorID, title, priority string, position float64, assigneeID, dueDate, description *string, subtaskTitles []string) (db.CreateCardRow, []db.CardSubtask, error) {
	return m.CreateCardWSFn(ctx, columnID, creatorID, title, priority, position, assigneeID, dueDate, description, subtaskTitles)
}

func (m *MockBoardCommandService) VerifyColumnInBoard(ctx context.Context, columnID, boardID string) error {
	return m.VerifyColumnInBoardFn(ctx, columnID, boardID)
}

func (m *MockBoardCommandService) MoveCard(ctx context.Context, cardID, newColumnID string, position float64) (service.MoveCardResult, error) {
	return m.MoveCardFn(ctx, cardID, newColumnID, position)
}

func (m *MockBoardCommandService) DeleteCard(ctx context.Context, cardID string) (string, error) {
	return m.DeleteCardFn(ctx, cardID)
}

func (m *MockBoardCommandService) ToggleCardDone(ctx context.Context, cardID, boardID string, isDone bool) (service.ToggleCardDoneResult, error) {
	return m.ToggleCardDoneFn(ctx, cardID, boardID, isDone)
}

func (m *MockBoardCommandService) CreateColumn(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error) {
	return m.CreateColumnFn(ctx, boardID, title, category, color)
}

func (m *MockBoardCommandService) DeleteColumn(ctx context.Context, columnID string) error {
	return m.DeleteColumnFn(ctx, columnID)
}

func (m *MockBoardCommandService) UpdateColumn(ctx context.Context, p service.UpdateColumnParams) error {
	return m.UpdateColumnFn(ctx, p)
}

// MockBroadcaster records every board room a message was sent to, so a test can
// assert that a mutation actually reached the room rather than only the database.
type MockBroadcaster struct {
	Sent []BroadcastCall
}

type BroadcastCall struct {
	BoardID string
	Message []byte
}

func (m *MockBroadcaster) Broadcast(boardID string, message []byte) {
	m.Sent = append(m.Sent, BroadcastCall{BoardID: boardID, Message: message})
}
