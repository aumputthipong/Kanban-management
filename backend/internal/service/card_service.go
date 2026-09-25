package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

// Set=false: no change. Set=true with nil Value: clear to NULL.
type FieldPatch[T any] struct {
	Set   bool
	Value *T
}

type UpdateCardParams struct {
	ID                 string
	Title              *string
	Description        *string
	DueDate            FieldPatch[time.Time]
	AssigneeID         FieldPatch[string]
	Priority           FieldPatch[string]
	EstimatedHours     FieldPatch[float64]
	TagIDs             *[]string
	AcceptanceCriteria *string
	ImplementationNote *string
}

func (s *BoardService) GetCard(ctx context.Context, cardID string) (db.Card, error) {
	return s.queries.GetCard(ctx, cardID)
}

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
		if u, uerr := s.queries.GetUserByID(ctx, *card.AssigneeID); uerr == nil {
			name := u.FullName
			assigneeName = &name
		}
	}
	return CardDetailData{Card: card, AssigneeName: assigneeName, Subtasks: subs, Tags: tags}, nil
}

// Tags are always loaded, so a broadcast never clobbers them.
type UpdateCardResult struct {
	Card db.Card
	Tags []TagData
}

func (s *BoardService) UpdateCard(ctx context.Context, arg UpdateCardParams) (UpdateCardResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return UpdateCardResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	card, err := qtx.UpdateCard(ctx, db.UpdateCardParams{
		ID:                 arg.ID,
		Title:              arg.Title,
		Description:        arg.Description,
		SetDueDate:         arg.DueDate.Set,
		DueDate:            arg.DueDate.Value,
		SetAssigneeID:      arg.AssigneeID.Set,
		AssigneeID:         arg.AssigneeID.Value,
		SetPriority:        arg.Priority.Set,
		Priority:           arg.Priority.Value,
		SetEstimatedHours:  arg.EstimatedHours.Set,
		EstimatedHours:     util.PtrFloatToPgNumeric(arg.EstimatedHours.Value),
		AcceptanceCriteria: arg.AcceptanceCriteria,
		ImplementationNote: arg.ImplementationNote,
	})
	if err != nil {
		return UpdateCardResult{}, fmt.Errorf("update card: %w", err)
	}

	if arg.TagIDs != nil {
		if len(*arg.TagIDs) > 5 {
			return UpdateCardResult{}, fmt.Errorf("card cannot have more than 5 tags")
		}
		if err := qtx.ClearCardTags(ctx, arg.ID); err != nil {
			return UpdateCardResult{}, fmt.Errorf("clear card tags: %w", err)
		}
		for _, tagID := range *arg.TagIDs {
			if err := qtx.InsertCardTag(ctx, db.InsertCardTagParams{CardID: arg.ID, TagID: tagID}); err != nil {
				return UpdateCardResult{}, fmt.Errorf("insert card tag: %w", err)
			}
		}
	}

	tagRows, err := qtx.GetTagsByCardIDs(ctx, []string{arg.ID})
	if err != nil {
		return UpdateCardResult{}, fmt.Errorf("fetch card tags: %w", err)
	}
	tags := make([]TagData, len(tagRows))
	for i, row := range tagRows {
		tags[i] = TagData{ID: row.ID, BoardID: row.BoardID, Name: row.Name, Color: row.Color}
	}

	if err := tx.Commit(ctx); err != nil {
		return UpdateCardResult{}, fmt.Errorf("commit tx: %w", err)
	}
	return UpdateCardResult{Card: card, Tags: tags}, nil
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
