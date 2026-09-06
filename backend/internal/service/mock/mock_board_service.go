package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// MockBoardService implements service.BoardServicer with the function-field pattern:
// a test sets only the methods it exercises.
type MockBoardService struct {
	GetAllBoardsFn              func(ctx context.Context, userID string) ([]service.BoardSummaryData, error)
	GetBoardWithCardsFn         func(ctx context.Context, boardID string) ([]service.ColumnData, error)
	CreateBoardFn               func(ctx context.Context, title string, description, color, icon *string, ownerID string) (string, error)
	UpdateBoardFn               func(ctx context.Context, id string, title *string, budget *float64, description, color, icon *string) (db.Board, error)
	StashBoardFn                func(ctx context.Context, boardID string) error
	GetStashedBoardsFn          func(ctx context.Context, userID string) ([]db.GetStashedBoardsForOwnerRow, error)
	HardDeleteBoardFn           func(ctx context.Context, id string) error
	RestoreBoardFn              func(ctx context.Context, id string) error
	GetBoardMemberRoleFn        func(ctx context.Context, boardID, userID string) (string, error)
	GetStashedBoardMemberRoleFn func(ctx context.Context, boardID, userID string) (string, error)
	TouchBoardMemberAccessFn    func(ctx context.Context, boardID, userID string) error
	GetBoardIDByColumnFn        func(ctx context.Context, columnID string) (string, error)
	GetBoardIDByCardFn          func(ctx context.Context, cardID string) (string, error)
	GetMyWorkFn                 func(ctx context.Context, opts service.MyWorkOptions) (service.MyWorkResult, error)
	CompleteMyTaskFn            func(ctx context.Context, cardID, userID string) (service.CompleteMyTaskResult, error)

	GetBoardMembersFn       func(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error)
	AddBoardMemberByEmailFn func(ctx context.Context, boardID, email, role string) error
	RemoveBoardMemberFn     func(ctx context.Context, boardID, userID string) error
	UpdateMemberRoleFn      func(ctx context.Context, boardID, userID string, role string) error

	GetCardFn       func(ctx context.Context, cardID string) (db.Card, error)
	GetCardDetailFn func(ctx context.Context, cardID string) (service.CardDetailData, error)
	CreateCardFn    func(ctx context.Context, arg db.CreateCardParams) (db.CreateCardRow, error)
	UpdateCardFn    func(ctx context.Context, arg service.UpdateCardParams) (db.Card, error)

	GetAllUsersFn func(ctx context.Context) ([]db.GetAllUsersRow, error)
}

func (m *MockBoardService) GetAllBoards(ctx context.Context, userID string) ([]service.BoardSummaryData, error) {
	return m.GetAllBoardsFn(ctx, userID)
}

func (m *MockBoardService) GetBoardWithCards(ctx context.Context, boardID string) ([]service.ColumnData, error) {
	return m.GetBoardWithCardsFn(ctx, boardID)
}

func (m *MockBoardService) CreateBoard(ctx context.Context, title string, description, color, icon *string, ownerID string) (string, error) {
	return m.CreateBoardFn(ctx, title, description, color, icon, ownerID)
}

func (m *MockBoardService) UpdateBoard(ctx context.Context, id string, title *string, budget *float64, description, color, icon *string) (db.Board, error) {
	return m.UpdateBoardFn(ctx, id, title, budget, description, color, icon)
}

func (m *MockBoardService) StashBoard(ctx context.Context, boardID string) error {
	return m.StashBoardFn(ctx, boardID)
}

func (m *MockBoardService) GetStashedBoards(ctx context.Context, userID string) ([]db.GetStashedBoardsForOwnerRow, error) {
	return m.GetStashedBoardsFn(ctx, userID)
}

func (m *MockBoardService) GetBoardMemberRole(ctx context.Context, boardID, userID string) (string, error) {
	return m.GetBoardMemberRoleFn(ctx, boardID, userID)
}

func (m *MockBoardService) GetStashedBoardMemberRole(ctx context.Context, boardID, userID string) (string, error) {
	return m.GetStashedBoardMemberRoleFn(ctx, boardID, userID)
}

func (m *MockBoardService) TouchBoardMemberAccess(ctx context.Context, boardID, userID string) error {
	if m.TouchBoardMemberAccessFn == nil {
		return nil
	}
	return m.TouchBoardMemberAccessFn(ctx, boardID, userID)
}

func (m *MockBoardService) GetBoardIDByColumn(ctx context.Context, columnID string) (string, error) {
	return m.GetBoardIDByColumnFn(ctx, columnID)
}

func (m *MockBoardService) GetBoardIDByCard(ctx context.Context, cardID string) (string, error) {
	return m.GetBoardIDByCardFn(ctx, cardID)
}

func (m *MockBoardService) GetMyWork(ctx context.Context, opts service.MyWorkOptions) (service.MyWorkResult, error) {
	return m.GetMyWorkFn(ctx, opts)
}

func (m *MockBoardService) CompleteMyTask(ctx context.Context, cardID, userID string) (service.CompleteMyTaskResult, error) {
	return m.CompleteMyTaskFn(ctx, cardID, userID)
}

func (m *MockBoardService) HardDeleteBoard(ctx context.Context, id string) error {
	return m.HardDeleteBoardFn(ctx, id)
}

func (m *MockBoardService) RestoreBoard(ctx context.Context, id string) error {
	return m.RestoreBoardFn(ctx, id)
}

func (m *MockBoardService) GetBoardMembers(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error) {
	return m.GetBoardMembersFn(ctx, boardID)
}

func (m *MockBoardService) AddBoardMemberByEmail(ctx context.Context, boardID, email, role string) error {
	return m.AddBoardMemberByEmailFn(ctx, boardID, email, role)
}

func (m *MockBoardService) RemoveBoardMember(ctx context.Context, boardID, userID string) error {
	return m.RemoveBoardMemberFn(ctx, boardID, userID)
}

func (m *MockBoardService) UpdateMemberRole(ctx context.Context, boardID, userID string, role string) error {
	return m.UpdateMemberRoleFn(ctx, boardID, userID, role)
}

func (m *MockBoardService) GetCard(ctx context.Context, cardID string) (db.Card, error) {
	return m.GetCardFn(ctx, cardID)
}

func (m *MockBoardService) GetCardDetail(ctx context.Context, cardID string) (service.CardDetailData, error) {
	return m.GetCardDetailFn(ctx, cardID)
}

func (m *MockBoardService) CreateCard(ctx context.Context, arg db.CreateCardParams) (db.CreateCardRow, error) {
	return m.CreateCardFn(ctx, arg)
}

func (m *MockBoardService) UpdateCard(ctx context.Context, arg service.UpdateCardParams) (db.Card, error) {
	return m.UpdateCardFn(ctx, arg)
}

func (m *MockBoardService) GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error) {
	return m.GetAllUsersFn(ctx)
}
