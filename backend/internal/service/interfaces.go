// Each Service has a Servicer interface so handlers can be tested against internal/service/mock.
package service

import (
	"context"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
)

type BoardServicer interface {
	// Board
	GetAllBoards(ctx context.Context, userID string) ([]BoardSummaryData, error)
	GetBoardWithCards(ctx context.Context, boardID string) ([]ColumnData, error)
	CreateBoard(ctx context.Context, title string, description, color, icon *string, ownerID string) (string, error)
	UpdateBoard(ctx context.Context, id string, title *string, budget *float64, description, color, icon *string) (db.Board, error)
	StashBoard(ctx context.Context, boardID string) error
	GetStashedBoards(ctx context.Context, userID string) ([]db.GetStashedBoardsForOwnerRow, error)
	HardDeleteBoard(ctx context.Context, id string) error
	RestoreBoard(ctx context.Context, id string) error
	GetBoardMemberRole(ctx context.Context, boardID, userID string) (string, error)
	GetStashedBoardMemberRole(ctx context.Context, boardID, userID string) (string, error)
	TouchBoardMemberAccess(ctx context.Context, boardID, userID string) error
	GetBoardIDByColumn(ctx context.Context, columnID string) (string, error)
	GetBoardIDByCard(ctx context.Context, cardID string) (string, error)

	// My Work (cross-board personal inbox)
	GetMyWork(ctx context.Context, opts MyWorkOptions) (MyWorkResult, error)
	CompleteMyTask(ctx context.Context, cardID, userID string) (CompleteMyTaskResult, error)

	// Member
	GetBoardMembers(ctx context.Context, boardID string) ([]db.GetBoardMembersRow, error)
	AddBoardMemberByEmail(ctx context.Context, boardID, email, role string) error
	RemoveBoardMember(ctx context.Context, boardID, userID string) error
	UpdateMemberRole(ctx context.Context, boardID, userID string, role string) error

	// Card
	GetCard(ctx context.Context, cardID string) (db.Card, error)
	GetCardDetail(ctx context.Context, cardID string) (CardDetailData, error)
	CreateCard(ctx context.Context, arg db.CreateCardParams) (db.CreateCardRow, error)
	UpdateCard(ctx context.Context, arg UpdateCardParams) (UpdateCardResult, error)

	// User
	GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error)
}

type SubtaskServicer interface {
	CreateSubtask(ctx context.Context, cardID, title string) (db.CardSubtask, error)
	GetSubtasksByCardID(ctx context.Context, cardID string) ([]db.CardSubtask, error)
	UpdateSubtask(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error)
	DeleteSubtask(ctx context.Context, subtaskID string) error
	GetSubtaskByID(ctx context.Context, subtaskID string) (db.CardSubtask, error)
}

// Board-scope checks happen in the handler; the service trusts its caller.
type PlanningServicer interface {
	ListSessionsByBoard(ctx context.Context, boardID string) ([]db.ListPlanningSessionsByBoardRow, error)
	GetSession(ctx context.Context, sessionID string) (db.PlanningSession, error)
	GetSessionBoardID(ctx context.Context, sessionID string) (string, error)
	GetItem(ctx context.Context, itemID string) (db.PlanningItem, error)
	GetItemBoardID(ctx context.Context, itemID string) (string, error)
	ListItems(ctx context.Context, sessionID string) ([]db.PlanningItem, error)
	CreateSession(ctx context.Context, boardID, title string, label, meetingAt *string, createdBy string) (db.PlanningSession, error)
	UpdateSession(ctx context.Context, sessionID string, title, label, meetingAt *string) (db.PlanningSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	CreateItem(ctx context.Context, sessionID, itemType, title string, description *string) (db.PlanningItem, error)
	UpdateItem(ctx context.Context, itemID string, itemType, title *string, description *string, status *string, position *float64, acceptanceCriteria, implementationNote *string) (db.PlanningItem, error)
	DeleteItem(ctx context.Context, itemID string) error
	PromoteItem(ctx context.Context, itemID, userID string) (db.PlanningItem, db.CreateCardRow, error)
	GetCardSource(ctx context.Context, cardID string, pendingLimit int32) (*CardSource, error)

	// Item comments
	ListItemComments(ctx context.Context, itemID string) ([]db.ListPlanningItemCommentsRow, error)
	GetComment(ctx context.Context, commentID string) (db.PlanningItemComment, error)
	GetCommentBoardID(ctx context.Context, commentID string) (string, error)
	CreateComment(ctx context.Context, itemID, authorID, body string) (db.PlanningItemComment, error)
	EditComment(ctx context.Context, commentID, body string) (db.PlanningItemComment, error)
	DeleteComment(ctx context.Context, commentID string) error
}

// Kept narrow so handlers don't pull in the read side.
type ActivityRecorder interface {
	Record(ctx context.Context, p RecordParams) (db.Activity, error)
	// Best-effort. Use Record when a broadcast needs the row.
	RecordAsync(p RecordParams)
}

type ActivityLister interface {
	List(ctx context.Context, boardID string, before *time.Time, limit int32) ([]ActivityItem, error)
}

// Get upserts a default row on first read.
type UserSettingsServicer interface {
	Get(ctx context.Context, userID string) (UserSettingsData, error)
	Update(ctx context.Context, userID string, p UpdateUserSettingsParams) (UserSettingsData, error)
}

type AuthServicer interface {
	Register(ctx context.Context, arg RegisterParams) (db.User, error)
	Login(ctx context.Context, email, password string) (db.User, error)
	UpsertOAuthUser(ctx context.Context, email, fullName, provider, providerID string) (db.User, error)
	GetUserByID(ctx context.Context, userID string) (db.GetUserByIDRow, error)
	IssueRefreshToken(ctx context.Context, userID, userAgent, ip string) (string, error)
	RotateRefreshToken(ctx context.Context, rawToken, userAgent, ip string) (RefreshRotationResult, error)
	RevokeRefreshToken(ctx context.Context, rawToken string) error
}

type DemoServicer interface {
	CreateSandbox(ctx context.Context, companionEmail string) (DemoSandbox, error)
	PurgeExpired(ctx context.Context) (int64, error)
}

type InviteServicer interface {
	CreateInvite(ctx context.Context, boardID, creatorID string) (InviteLink, error)
	GetActiveInvite(ctx context.Context, boardID string) (InviteLink, bool, error)
	RevokeInvites(ctx context.Context, boardID string) error
	AcceptInvite(ctx context.Context, token, userID string) (boardID string, joined bool, err error)
}

type TagServicer interface {
	GetTagsByBoard(ctx context.Context, boardID string) ([]db.Tag, error)
	CreateTag(ctx context.Context, boardID, name, color string) (db.Tag, error)
	DeleteTag(ctx context.Context, boardID, tagID string) error
}

// Shared with the WS handlers — see docs/adr/0003 and #197.
type BoardCommandServicer interface {
	VerifyCardInBoard(ctx context.Context, cardID, boardID string) error
	CreateCardWS(ctx context.Context, columnID, creatorID, title, priority string, position float64, assigneeID, dueDate, description *string, subtaskTitles []string) (db.CreateCardRow, []db.CardSubtask, error)
	VerifyColumnInBoard(ctx context.Context, columnID, boardID string) error
	MoveCard(ctx context.Context, cardID, newColumnID string, position float64) (MoveCardResult, error)
	DeleteCard(ctx context.Context, cardID string) (string, error)
	ToggleCardDone(ctx context.Context, cardID, boardID string, isDone bool) (ToggleCardDoneResult, error)
	CreateColumn(ctx context.Context, boardID, title, category string, color *string) (db.CreateColumnRow, error)
	DeleteColumn(ctx context.Context, columnID string) error
	UpdateColumn(ctx context.Context, p UpdateColumnParams) error
}
