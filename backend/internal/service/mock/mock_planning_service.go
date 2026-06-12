package mock

import (
	"context"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// MockPlanningService implements service.PlanningServicer using the same
// "function field per method" pattern as MockBoardService. Each test sets
// only the Fn fields it needs; unset methods panic if called so missing
// stubs surface loudly instead of returning zero values silently.
type MockPlanningService struct {
	ListSessionsByBoardFn func(ctx context.Context, boardID string) ([]db.ListPlanningSessionsByBoardRow, error)
	GetSessionFn          func(ctx context.Context, sessionID string) (db.PlanningSession, error)
	GetSessionBoardIDFn   func(ctx context.Context, sessionID string) (string, error)
	GetItemFn             func(ctx context.Context, itemID string) (db.PlanningItem, error)
	GetItemBoardIDFn      func(ctx context.Context, itemID string) (string, error)
	ListItemsFn           func(ctx context.Context, sessionID string) ([]db.PlanningItem, error)
	CreateSessionFn       func(ctx context.Context, boardID, title string, label, meetingAt *string, createdBy string) (db.PlanningSession, error)
	UpdateSessionFn       func(ctx context.Context, sessionID string, title, label, meetingAt *string) (db.PlanningSession, error)
	DeleteSessionFn       func(ctx context.Context, sessionID string) error
	CreateItemFn          func(ctx context.Context, sessionID, itemType, title string, description *string) (db.PlanningItem, error)
	UpdateItemFn          func(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error)
	DeleteItemFn          func(ctx context.Context, itemID string) error
	PromoteItemFn         func(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error)
	GetCardSourceFn       func(ctx context.Context, cardID string, pendingLimit int32) (*service.CardSource, error)
	ListItemCommentsFn    func(ctx context.Context, itemID string) ([]db.ListPlanningItemCommentsRow, error)
	GetCommentFn          func(ctx context.Context, commentID string) (db.PlanningItemComment, error)
	GetCommentBoardIDFn   func(ctx context.Context, commentID string) (string, error)
	CreateCommentFn       func(ctx context.Context, itemID, authorID, body string) (db.PlanningItemComment, error)
	EditCommentFn         func(ctx context.Context, commentID, body string) (db.PlanningItemComment, error)
	DeleteCommentFn       func(ctx context.Context, commentID string) error
}

func (m *MockPlanningService) ListSessionsByBoard(ctx context.Context, boardID string) ([]db.ListPlanningSessionsByBoardRow, error) {
	return m.ListSessionsByBoardFn(ctx, boardID)
}

func (m *MockPlanningService) GetSession(ctx context.Context, sessionID string) (db.PlanningSession, error) {
	return m.GetSessionFn(ctx, sessionID)
}

func (m *MockPlanningService) GetSessionBoardID(ctx context.Context, sessionID string) (string, error) {
	return m.GetSessionBoardIDFn(ctx, sessionID)
}

func (m *MockPlanningService) GetItem(ctx context.Context, itemID string) (db.PlanningItem, error) {
	return m.GetItemFn(ctx, itemID)
}

func (m *MockPlanningService) GetItemBoardID(ctx context.Context, itemID string) (string, error) {
	return m.GetItemBoardIDFn(ctx, itemID)
}

func (m *MockPlanningService) ListItems(ctx context.Context, sessionID string) ([]db.PlanningItem, error) {
	return m.ListItemsFn(ctx, sessionID)
}

func (m *MockPlanningService) CreateSession(ctx context.Context, boardID, title string, label, meetingAt *string, createdBy string) (db.PlanningSession, error) {
	return m.CreateSessionFn(ctx, boardID, title, label, meetingAt, createdBy)
}

func (m *MockPlanningService) UpdateSession(ctx context.Context, sessionID string, title, label, meetingAt *string) (db.PlanningSession, error) {
	return m.UpdateSessionFn(ctx, sessionID, title, label, meetingAt)
}

func (m *MockPlanningService) DeleteSession(ctx context.Context, sessionID string) error {
	return m.DeleteSessionFn(ctx, sessionID)
}

func (m *MockPlanningService) CreateItem(ctx context.Context, sessionID, itemType, title string, description *string) (db.PlanningItem, error) {
	return m.CreateItemFn(ctx, sessionID, itemType, title, description)
}

func (m *MockPlanningService) UpdateItem(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error) {
	return m.UpdateItemFn(ctx, itemID, itemType, title, description, status, position, acceptanceCriteria, implementationNote)
}

func (m *MockPlanningService) DeleteItem(ctx context.Context, itemID string) error {
	return m.DeleteItemFn(ctx, itemID)
}

func (m *MockPlanningService) PromoteItem(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error) {
	return m.PromoteItemFn(ctx, itemID, userID)
}

func (m *MockPlanningService) GetCardSource(ctx context.Context, cardID string, pendingLimit int32) (*service.CardSource, error) {
	return m.GetCardSourceFn(ctx, cardID, pendingLimit)
}

func (m *MockPlanningService) ListItemComments(ctx context.Context, itemID string) ([]db.ListPlanningItemCommentsRow, error) {
	return m.ListItemCommentsFn(ctx, itemID)
}

func (m *MockPlanningService) GetComment(ctx context.Context, commentID string) (db.PlanningItemComment, error) {
	return m.GetCommentFn(ctx, commentID)
}

func (m *MockPlanningService) GetCommentBoardID(ctx context.Context, commentID string) (string, error) {
	return m.GetCommentBoardIDFn(ctx, commentID)
}

func (m *MockPlanningService) CreateComment(ctx context.Context, itemID, authorID, body string) (db.PlanningItemComment, error) {
	return m.CreateCommentFn(ctx, itemID, authorID, body)
}

func (m *MockPlanningService) EditComment(ctx context.Context, commentID, body string) (db.PlanningItemComment, error) {
	return m.EditCommentFn(ctx, commentID, body)
}

func (m *MockPlanningService) DeleteComment(ctx context.Context, commentID string) error {
	return m.DeleteCommentFn(ctx, commentID)
}

// MockActivityRecorder records each Record() invocation in a slice so tests
// can assert "exactly one activity row was written for event_type X". The
// pattern matches the function-field convention but keeping a capture slice
// is simpler than reimplementing arg-equality checks per test.
type MockActivityRecorder struct {
	Calls []service.RecordParams
	// RecordFn is optional — set it to override the default (record + return
	// a fake Activity with the call's event_type echoed back).
	RecordFn func(ctx context.Context, p service.RecordParams) (db.Activity, error)
}

func (m *MockActivityRecorder) Record(ctx context.Context, p service.RecordParams) (db.Activity, error) {
	m.Calls = append(m.Calls, p)
	if m.RecordFn != nil {
		return m.RecordFn(ctx, p)
	}
	return db.Activity{EventType: p.EventType, EntityType: p.EntityType}, nil
}

// RecordAsync mirrors Record for capture purposes — tests still inspect
// m.Calls regardless of whether the handler chose the sync or async path.
func (m *MockActivityRecorder) RecordAsync(p service.RecordParams) {
	m.Calls = append(m.Calls, p)
}
