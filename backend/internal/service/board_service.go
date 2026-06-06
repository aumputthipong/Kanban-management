// internal/service/board_service.go
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

// ColumnData และ CardData ใช้ string สำหรับ ID แทน uuid.UUID
// เพราะ sqlc generate ออกมาเป็น string อยู่แล้ว ไม่ต้องแปลงซ้ำ
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

// CreateBoard สร้าง board ใหม่พร้อม 3 columns เริ่มต้น และกำหนด ownerID เป็น owner
// รับและคืน string แทน uuid.UUID เพื่อให้ตรงกับ type จาก sqlc
func (s *BoardService) CreateBoard(ctx context.Context, title string, ownerID string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("board title cannot be empty")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	board, err := qtx.CreateBoard(ctx, title)
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

	// group members by board ID
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

// GetStashedBoardMemberRole resolves the caller's role only when the board is
// stashed — used by the /api/stash gate so restore / permanent-delete work on
// stashed boards while GetBoardMemberRole keeps 404ing for them everywhere else.
func (s *BoardService) GetStashedBoardMemberRole(ctx context.Context, boardID, userID string) (string, error) {
	return s.queries.GetStashedBoardMemberRole(ctx, db.GetStashedBoardMemberRoleParams{
		BoardID: boardID,
		UserID:  userID,
	})
}

// TouchBoardMemberAccess bumps the membership's last_accessed_at when the
// previous touch is older than the 5-minute throttle window (enforced in
// SQL). Safe to call from a goroutine: best-effort, no business semantics.
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

// MyTaskData mirrors the row shape returned to the frontend's My Work page.
// `Status` is derived in SQL: "todo" for the first TODO column, "in_progress"
// otherwise. `Group` is the date bucket (overdue/today/this_week/later/no_date)
// also computed in SQL against the caller's "today".
type MyTaskData struct {
	ID             string
	Title          string
	BoardID        string
	BoardName      string
	ColumnName     string
	Priority       *string
	DueDate        *time.Time
	EstimatedHours *float64
	IsDone         bool
	Status         string
	Group          string
}

// MyWorkFilter narrows what GetMyWork returns. Counts always reflect the
// unfiltered inbox so the frontend can render filter-chip counters.
type MyWorkFilter string

const (
	MyWorkFilterAll      MyWorkFilter = "all"
	MyWorkFilterOverdue  MyWorkFilter = "overdue"
	MyWorkFilterToday    MyWorkFilter = "today"
	MyWorkFilterThisWeek MyWorkFilter = "this_week"
	MyWorkFilterNoDate   MyWorkFilter = "no_date"
)

// MyWorkCounts mirrors dto.MyWorkCounts but lives in service so the layer
// stays decoupled from the wire shape.
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
	// Today is the caller-local date. The handler injects this in the user's
	// timezone (Asia/Bangkok hardcoded in S.1) so the SQL CASE-bucket aligns
	// with what the user reads as "วันนี้".
	Today time.Time
}

type MyWorkResult struct {
	Cards  []MyTaskData
	Counts MyWorkCounts
}

// defaultTZ is the workspace fallback when a user has no timezone set or the
// configured one fails to load. Bangkok matches the default in user_settings.
var defaultTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

// MyWorkToday returns "today" in the supplied IANA timezone, truncated to
// midnight, ready to pass to GetMyWork as the bucket pivot. An empty or
// unknown tz falls back to Asia/Bangkok.
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

// GetMyWork lists the caller's inbox across boards. The query returns all
// matching cards in one shot; counts are computed in Go (one pass) and the
// filter is applied after counting so the chip totals always reflect the full
// inbox.
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
			ID:             r.ID,
			Title:          r.Title,
			BoardID:        r.BoardID,
			BoardName:      r.BoardName,
			ColumnName:     r.ColumnName,
			Priority:       r.Priority,
			DueDate:        r.DueDate,
			EstimatedHours: util.PgNumericToFloat64Ptr(r.EstimatedHours),
			IsDone:         r.IsDone,
			Status:         r.Status,
			Group:          r.WorkGroup,
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

// CompleteMyTaskResult captures the side data the handler needs to write an
// activity row (board scope, card title for the feed) without forcing a
// second roundtrip.
type CompleteMyTaskResult struct {
	OK        bool
	BoardID   string
	CardTitle string
}

// CompleteMyTask marks a card done + moves it to the board's first DONE column.
// Returns OK=false if the caller is not the assignee (so handler can 404). The
// returned BoardID + CardTitle let the REST handler record an activity row
// after-the-fact without another DB hit.
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
	// Fetch the card up front so we can return the title on success; cheap
	// because cards.id is the primary key. Skip it if the assignee gate
	// would reject — but we need the row to know the title either way.
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
	return CompleteMyTaskResult{
		OK:        rows > 0,
		BoardID:   boardID,
		CardTitle: card.Title,
	}, nil
}

func (s *BoardService) HardDeleteBoard(ctx context.Context, id string) error {
	return s.queries.HardDeleteBoard(ctx, id)
}

func (s *BoardService) RestoreBoard(ctx context.Context, id string) error {
	return s.queries.RestoreStashedBoard(ctx, id)
}

// UpdateBoard read-modify-writes one board. Any nil pointer leaves the
// existing value untouched (omit/null = no change); a non-nil pointer
// overwrites — including Description "" which is a valid clear since the
// column is NOT NULL DEFAULT ”.
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

// GetBoardWithCards ดึง columns, cards และ subtasks ทั้งหมดของ board ใน 3 queries
func (s *BoardService) GetBoardWithCards(ctx context.Context, boardID string) ([]ColumnData, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. ดึงข้อมูล Columns
	columns, err := s.queries.GetColumnsByBoardID(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("fetch columns: %w", err)
	}

	// 🌟 Best Practice: ตรวจสอบความว่างเปล่า (Early Return)
	// ถ้าบอร์ดนี้ยังไม่มี Column เลย ให้ส่ง Array ว่างกลับไปทันที
	if len(columns) == 0 {
		return []ColumnData{}, nil
	}

	columnIDs := make([]string, len(columns))
	for i, col := range columns {
		columnIDs[i] = col.ID
	}

	// 2. ดึงข้อมูล Cards
	// (ถึงบรรทัดนี้ การันตีได้แล้วว่า columnIDs มีค่าอย่างน้อย 1 ตัวแน่นอน)
	cards, err := s.queries.GetCardsByColumnIDs(ctx, columnIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch cards: %w", err)
	}

	// 3. Batch load tags for all cards
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
		// 🌟 Best Practice: เช็คค่า nil ของ map
		// ถ้า Column นั้นไม่มี Card เลย map จะคืนค่า nil
		// เราควรแปลงเป็น Slice ว่าง []CardData{} แทนที่ Frontend จะได้รับเป็น null
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
