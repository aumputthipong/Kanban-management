package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

// UpdateCardParams is the card-update input passed from the handler: string
// IDs, *string for nullable fields.
type UpdateCardParams struct {
	ID             string
	Title          string
	Description    *string
	DueDate        *time.Time
	AssigneeID     *string
	Priority       *string
	EstimatedHours *float64
	TagIDs         *[]string // nil = don't touch, &[]string{} = clear all
	// AcceptanceCriteria and ImplementationNote follow PATCH semantics
	// (nil = no change, &"" = clear). Unlike the fields above, the SQL
	// uses COALESCE for these two so a "title only" edit doesn't wipe
	// out values that PromoteItem copied over from the planning row.
	AcceptanceCriteria *string
	ImplementationNote *string
}

type CardService struct {
	queries *db.Queries
}

func (s *BoardService) GetCard(ctx context.Context, cardID string) (db.Card, error) {
	return s.queries.GetCard(ctx, cardID)
}

// CardDetailData is a fully enriched card for the detail view. GetCard alone returns the
// raw row; this also resolves the assignee name and loads subtasks and tags so the
// response carries everything the modal shows.
type CardDetailData struct {
	Card         db.Card
	AssigneeName *string
	Subtasks     []db.CardSubtask
	Tags         []db.GetTagsByCardIDsRow
}

func (s *BoardService) GetCardDetail(ctx context.Context, cardID string) (CardDetailData, error) {
	card, err := s.queries.GetCard(ctx, cardID)
	if err != nil {
		return CardDetailData{}, err
	}
	subs, err := s.queries.GetSubtasksByCardID(ctx, cardID)
	if err != nil {
		return CardDetailData{}, fmt.Errorf("load subtasks: %w", err)
	}
	tags, err := s.queries.GetTagsByCardIDs(ctx, []string{cardID})
	if err != nil {
		return CardDetailData{}, fmt.Errorf("load tags: %w", err)
	}
	var assigneeName *string
	if card.AssigneeID != nil {
		// Best-effort: a missing user just leaves the name nil.
		if u, uerr := s.queries.GetUserByID(ctx, *card.AssigneeID); uerr == nil {
			name := u.FullName
			assigneeName = &name
		}
	}
	return CardDetailData{Card: card, AssigneeName: assigneeName, Subtasks: subs, Tags: tags}, nil
}

func (s *BoardService) UpdateCard(ctx context.Context, arg UpdateCardParams) (db.Card, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Card{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	card, err := qtx.UpdateCard(ctx, db.UpdateCardParams{
		ID:                 arg.ID,
		Title:              arg.Title,
		Description:        arg.Description,
		Priority:           arg.Priority,
		DueDate:            arg.DueDate,
		AssigneeID:         arg.AssigneeID,
		EstimatedHours:     util.PtrFloatToPgNumeric(arg.EstimatedHours),
		AcceptanceCriteria: arg.AcceptanceCriteria,
		ImplementationNote: arg.ImplementationNote,
	})
	if err != nil {
		return db.Card{}, fmt.Errorf("update card: %w", err)
	}

	if arg.TagIDs != nil {
		if len(*arg.TagIDs) > 5 {
			return db.Card{}, fmt.Errorf("card cannot have more than 5 tags")
		}
		if err := qtx.ClearCardTags(ctx, arg.ID); err != nil {
			return db.Card{}, fmt.Errorf("clear card tags: %w", err)
		}
		for _, tagID := range *arg.TagIDs {
			if err := qtx.InsertCardTag(ctx, db.InsertCardTagParams{CardID: arg.ID, TagID: tagID}); err != nil {
				return db.Card{}, fmt.Errorf("insert card tag: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Card{}, fmt.Errorf("commit tx: %w", err)
	}
	return card, nil
}

func (s *BoardService) GetAllUsers(ctx context.Context) ([]db.GetAllUsersRow, error) {
	return s.queries.GetAllUsers(ctx)
}

func (s *BoardService) GetCardsByColumnIDs(ctx context.Context, columnIDs []string) ([]db.GetCardsByColumnIDsRow, error) {
	return s.queries.GetCardsByColumnIDs(ctx, columnIDs)
}

func (s *BoardService) CreateCard(ctx context.Context, arg db.CreateCardParams) (db.CreateCardRow, error) {
	return s.queries.CreateCard(ctx, arg)
}
