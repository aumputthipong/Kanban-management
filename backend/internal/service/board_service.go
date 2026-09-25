package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BoardService struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// String ids to match what sqlc generates.
type ColumnData struct {
	ID       string
	Title    string
	Position float64
	Category string
	Color    *string
	Cards    []CardData
}

type SubtaskData struct {
	ID       string
	CardID   string
	Title    string
	IsDone   bool
	Position float64
}

type TagData struct {
	ID      string
	BoardID string
	Name    string
	Color   string
}

type CardData struct {
	ID                string
	ColumnID          string
	Title             string
	Description       *string
	Position          float64
	DueDate           *time.Time
	EstimatedHours    *float64
	AssigneeID        *string
	AssigneeName      *string
	Priority          *string
	IsDone            bool
	CompletedAt       *time.Time
	CreatedAt         *time.Time
	CreatedBy         *string
	Subtasks          []SubtaskData
	TotalSubtasks     int64
	CompletedSubtasks int64
	Tags              []TagData
}

type BoardSummaryData struct {
	ID             string
	Title          string
	Description    string
	Color          string
	Icon           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastAccessedAt *time.Time
	TotalCards     int
	DoneCards      int
	Members        []dto.MemberSummary
}

func NewBoardService(pool *pgxpool.Pool, queries *db.Queries) *BoardService {
	return &BoardService{
		pool:    pool,
		queries: queries,
	}
}

// Board, default columns and owner in one transaction; nil appearance uses the column default.
func (s *BoardService) CreateBoard(ctx context.Context, title string, description, color, icon *string, ownerID string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("board title cannot be empty")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	board, err := qtx.CreateBoard(ctx, db.CreateBoardParams{
		Title:       title,
		Description: description,
		Color:       color,
		Icon:        icon,
	})
	if err != nil {
		return "", fmt.Errorf("create board: %w", err)
	}

	defaultColumns := []struct {
		Title    string
		Position float64
		Category string
	}{
		{"To Do", 65536.0, "TODO"},
		{"In Progress", 131072.0, "TODO"},
		{"Review", 196608.0, "TODO"},
		{"Done", 262144.0, "DONE"},
	}

	for _, col := range defaultColumns {
		_, err := qtx.CreateColumn(ctx, db.CreateColumnParams{
			BoardID:  board.ID,
			Title:    col.Title,
			Position: col.Position,
			Category: col.Category,
		})
		if err != nil {
			return "", fmt.Errorf("create column %q: %w", col.Title, err)
		}
	}

	if _, err := qtx.AddBoardMember(ctx, db.AddBoardMemberParams{
		BoardID: board.ID,
		UserID:  ownerID,
		Role:    "owner",
	}); err != nil {
		return "", fmt.Errorf("add owner to board: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}

	return board.ID, nil
}

func (s *BoardService) GetAllBoards(ctx context.Context, userID string) ([]BoardSummaryData, error) {
	stats, err := s.queries.GetActiveBoardsWithStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	memberRows, err := s.queries.GetMembersForActiveBoards(ctx, userID)
	if err != nil {
		return nil, err
	}

	membersByBoard := make(map[string][]dto.MemberSummary, len(stats))
	for _, m := range memberRows {
		membersByBoard[m.BoardID] = append(membersByBoard[m.BoardID], dto.MemberSummary{
			UserID:   m.UserID,
			FullName: m.FullName,
		})
	}

	result := make([]BoardSummaryData, 0, len(stats))
	for _, b := range stats {
		members := membersByBoard[b.ID]
		if members == nil {
			members = []dto.MemberSummary{}
		}
		result = append(result, BoardSummaryData{
			ID:             b.ID,
			Title:          b.Title,
			Description:    b.Description,
			Color:          b.Color,
			Icon:           b.Icon,
			CreatedAt:      b.CreatedAt.Time,
			UpdatedAt:      b.UpdatedAt.Time,
			LastAccessedAt: b.LastAccessedAt,
			TotalCards:     int(b.TotalCards),
			DoneCards:      int(b.DoneCards),
			Members:        members,
		})
	}
	return result, nil
}

func (s *BoardService) GetColumnsByBoardID(ctx context.Context, boardID string) ([]db.Column, error) {
	return s.queries.GetColumnsByBoardID(ctx, boardID)
}

func (s *BoardService) StashBoard(ctx context.Context, boardID string) error {
	return s.queries.StashBoard(ctx, boardID)
}

func (s *BoardService) GetStashedBoards(ctx context.Context, userID string) ([]db.GetStashedBoardsForOwnerRow, error) {
	return s.queries.GetStashedBoardsForOwner(ctx, userID)
}

func (s *BoardService) GetBoardMemberRole(ctx context.Context, boardID, userID string) (string, error) {
	return s.queries.GetBoardMemberRole(ctx, db.GetBoardMemberRoleParams{
		BoardID: boardID,
		UserID:  userID,
	})
}

// Matches stashed boards only (the /api/stash gate).
func (s *BoardService) GetStashedBoardMemberRole(ctx context.Context, boardID, userID string) (string, error) {
	return s.queries.GetStashedBoardMemberRole(ctx, db.GetStashedBoardMemberRoleParams{
		BoardID: boardID,
		UserID:  userID,
	})
}

// Throttled to 5 minutes in SQL. Best-effort.
func (s *BoardService) TouchBoardMemberAccess(ctx context.Context, boardID, userID string) error {
	return s.queries.TouchBoardMemberIfStale(ctx, db.TouchBoardMemberIfStaleParams{
		BoardID: boardID,
		UserID:  userID,
	})
}

func (s *BoardService) GetBoardIDByColumn(ctx context.Context, columnID string) (string, error) {
	return s.queries.GetBoardIDByColumn(ctx, columnID)
}

func (s *BoardService) GetBoardIDByCard(ctx context.Context, cardID string) (string, error) {
	return s.queries.GetBoardIDByCard(ctx, cardID)
}

// Status and Group are derived in SQL.
type MyTaskData struct {
	ID                string
	Title             string
	BoardID           string
	BoardName         string
	ColumnName        string
	Priority          *string
	DueDate           *time.Time
	EstimatedHours    *float64
	IsDone            bool
	Status            string
	Group             string
	TotalSubtasks     int64
	CompletedSubtasks int64
}

// Counts always cover the unfiltered inbox.
type MyWorkFilter string

const (
	MyWorkFilterAll      MyWorkFilter = "all"
	MyWorkFilterOverdue  MyWorkFilter = "overdue"
	MyWorkFilterToday    MyWorkFilter = "today"
	MyWorkFilterThisWeek MyWorkFilter = "this_week"
	MyWorkFilterNoDate   MyWorkFilter = "no_date"
)

type MyWorkCounts struct {
	Overdue  int
	Today    int
	ThisWeek int
	Later    int
	NoDate   int
	Total    int
}

type MyWorkOptions struct {
	UserID            string
	IncludeUnassigned bool
	Filter            MyWorkFilter
	// Caller-local date, so the SQL buckets match the user's "today".
	Today time.Time
}

type MyWorkResult struct {
	Cards  []MyTaskData
	Counts MyWorkCounts
}

var defaultTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

// Unknown or empty tz falls back to Asia/Bangkok.
func MyWorkToday(now time.Time, tz string) time.Time {
	loc := defaultTZ
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func (s *BoardService) GetMyWork(ctx context.Context, opts MyWorkOptions) (MyWorkResult, error) {
	rows, err := s.queries.GetMyTasks(ctx, db.GetMyTasksParams{
		Today:             opts.Today,
		UserID:            opts.UserID,
		IncludeUnassigned: opts.IncludeUnassigned,
	})
	if err != nil {
		return MyWorkResult{}, fmt.Errorf("get my tasks: %w", err)
	}

	cards := make([]MyTaskData, 0, len(rows))
	var counts MyWorkCounts
	for _, r := range rows {
		switch r.WorkGroup {
		case "overdue":
			counts.Overdue++
		case "today":
			counts.Today++
		case "this_week":
			counts.ThisWeek++
		case "later":
			counts.Later++
		case "no_date":
			counts.NoDate++
		}
		counts.Total++

		if !matchesFilter(opts.Filter, r.WorkGroup) {
			continue
		}
		cards = append(cards, MyTaskData{
			ID:                r.ID,
			Title:             r.Title,
			BoardID:           r.BoardID,
			BoardName:         r.BoardName,
			ColumnName:        r.ColumnName,
			Priority:          r.Priority,
			DueDate:           r.DueDate,
			EstimatedHours:    util.PgNumericToFloat64Ptr(r.EstimatedHours),
			IsDone:            r.IsDone,
			Status:            r.Status,
			Group:             r.WorkGroup,
			TotalSubtasks:     r.TotalSubtasks,
			CompletedSubtasks: r.CompletedSubtasks,
		})
	}
	return MyWorkResult{Cards: cards, Counts: counts}, nil
}

func matchesFilter(f MyWorkFilter, group string) bool {
	switch f {
	case "", MyWorkFilterAll:
		return true
	case MyWorkFilterOverdue:
		return group == "overdue"
	case MyWorkFilterToday:
		return group == "today"
	case MyWorkFilterThisWeek:
		return group == "this_week"
	case MyWorkFilterNoDate:
		return group == "no_date"
	default:
		return true
	}
}

type CompleteMyTaskResult struct {
	OK        bool
	BoardID   string
	CardTitle string
	// Zero when !OK.
	ColumnID    string
	Position    float64
	CompletedAt *time.Time
}

// OK=false when the caller isn't the assignee (the handler 404s).
func (s *BoardService) CompleteMyTask(ctx context.Context, cardID, userID string) (CompleteMyTaskResult, error) {
	boardID, err := s.queries.GetBoardIDByCard(ctx, cardID)
	if err != nil {
		return CompleteMyTaskResult{}, fmt.Errorf("resolve board: %w", err)
	}
	doneCol, err := s.queries.GetColumnByBoardAndCategory(ctx, db.GetColumnByBoardAndCategoryParams{
		BoardID:  boardID,
		Category: "DONE",
	})
	if err != nil {
		return CompleteMyTaskResult{}, fmt.Errorf("find DONE column: %w", err)
	}
	card, err := s.queries.GetCard(ctx, cardID)
	if err != nil {
		return CompleteMyTaskResult{}, fmt.Errorf("load card: %w", err)
	}
	rows, err := s.queries.CompleteCardAsAssignee(ctx, db.CompleteCardAsAssigneeParams{
		ID:         cardID,
		ColumnID:   doneCol.ID,
		AssigneeID: &userID,
	})
	if err != nil {
		return CompleteMyTaskResult{}, fmt.Errorf("complete card: %w", err)
	}
	if rows == 0 {
		return CompleteMyTaskResult{BoardID: boardID, CardTitle: card.Title}, nil
	}
	// Re-read: completed_at is set by NOW() in SQL.
	done, err := s.queries.GetCard(ctx, cardID)
	if err != nil {
		return CompleteMyTaskResult{}, fmt.Errorf("reload completed card: %w", err)
	}
	return CompleteMyTaskResult{
		OK:          true,
		BoardID:     boardID,
		CardTitle:   done.Title,
		ColumnID:    done.ColumnID,
		Position:    done.Position,
		CompletedAt: util.TimestamptzToTimePtr(done.CompletedAt),
	}, nil
}

func (s *BoardService) HardDeleteBoard(ctx context.Context, id string) error {
	return s.queries.HardDeleteBoard(ctx, id)
}

func (s *BoardService) RestoreBoard(ctx context.Context, id string) error {
	return s.queries.RestoreStashedBoard(ctx, id)
}

// nil = no change; Description "" is a valid clear.
func (s *BoardService) UpdateBoard(ctx context.Context, id string, title *string, budget *float64, description, color, icon *string) (db.Board, error) {
	existingBoard, err := s.queries.GetBoardByID(ctx, id)
	if err != nil {
		return db.Board{}, err
	}

	newTitle := existingBoard.Title
	newBudget := existingBoard.Budget
	newDescription := existingBoard.Description
	newColor := existingBoard.Color
	newIcon := existingBoard.Icon

	if title != nil {
		newTitle = *title
	}
	if budget != nil {
		newBudget = util.PtrFloatToPgNumeric(budget)
	}
	if description != nil {
		newDescription = *description
	}
	if color != nil {
		newColor = *color
	}
	if icon != nil {
		newIcon = *icon
	}

	return s.queries.UpdateBoard(ctx, db.UpdateBoardParams{
		ID:          id,
		Title:       newTitle,
		Budget:      newBudget,
		Description: newDescription,
		Color:       newColor,
		Icon:        newIcon,
	})
}

// Batched queries — no per-card round-trips.
func (s *BoardService) GetBoardWithCards(ctx context.Context, boardID string) ([]ColumnData, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	columns, err := s.queries.GetColumnsByBoardID(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("fetch columns: %w", err)
	}
	if len(columns) == 0 {
		return []ColumnData{}, nil
	}

	columnIDs := make([]string, len(columns))
	for i, col := range columns {
		columnIDs[i] = col.ID
	}

	cards, err := s.queries.GetCardsByColumnIDs(ctx, columnIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch cards: %w", err)
	}

	cardIDs := make([]string, len(cards))
	for i, card := range cards {
		cardIDs[i] = card.ID
	}
	tagRows, err := s.queries.GetTagsByCardIDs(ctx, cardIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch card tags: %w", err)
	}
	tagsByCard := make(map[string][]TagData)
	for _, row := range tagRows {
		tagsByCard[row.CardID] = append(tagsByCard[row.CardID], TagData{
			ID:      row.ID,
			BoardID: row.BoardID,
			Name:    row.Name,
			Color:   row.Color,
		})
	}

	cardsByColumn := make(map[string][]CardData)
	for _, card := range cards {
		tags := tagsByCard[card.ID]
		if tags == nil {
			tags = []TagData{}
		}
		cardsByColumn[card.ColumnID] = append(cardsByColumn[card.ColumnID], CardData{
			ID:                card.ID,
			ColumnID:          card.ColumnID,
			Title:             card.Title,
			Description:       card.Description,
			Position:          card.Position,
			DueDate:           card.DueDate,
			EstimatedHours:    util.PgNumericToFloat64Ptr(card.EstimatedHours),
			AssigneeID:        card.AssigneeID,
			AssigneeName:      card.AssigneeName,
			Priority:          card.Priority,
			IsDone:            card.IsDone,
			CompletedAt:       util.TimestamptzToTimePtr(card.CompletedAt),
			CreatedAt:         util.TimestamptzToTimePtr(card.CreatedAt),
			CreatedBy:         card.CreatedBy,
			TotalSubtasks:     card.TotalSubtasks,
			CompletedSubtasks: card.CompletedSubtasks,
			Tags:              tags,
		})
	}

	result := make([]ColumnData, 0, len(columns))
	for _, col := range columns {
		// nil → [] so the frontend never gets null.
		colCards := cardsByColumn[col.ID]
		if colCards == nil {
			colCards = []CardData{}
		}

		result = append(result, ColumnData{
			ID:       col.ID,
			Title:    col.Title,
			Position: col.Position,
			Category: col.Category,
			Color:    col.Color,
			Cards:    colCards,
		})
	}

	return result, nil
}
