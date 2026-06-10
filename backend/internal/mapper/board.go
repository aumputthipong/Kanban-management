// internal/mapper/board.go
package mapper

import (
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/util"
)

func ToStashedBoardDTO(b db.GetStashedBoardsForOwnerRow) dto.StashedBoardDTO {
	var stashedAt time.Time
	if b.DeletedAt.Valid {
		stashedAt = b.DeletedAt.Time
	}
	return dto.StashedBoardDTO{
		ID:          b.ID,
		Title:       b.Title,
		Description: b.Description,
		Color:       b.Color,
		Icon:        b.Icon,
		StashedAt:   stashedAt,
	}
}

func ToStashedBoardDTOs(boards []db.GetStashedBoardsForOwnerRow) []dto.StashedBoardDTO {
	result := make([]dto.StashedBoardDTO, len(boards))
	for i, b := range boards {
		result[i] = ToStashedBoardDTO(b)
	}
	return result
}

func ToSubtaskResponse(s db.CardSubtask) dto.SubtaskResponse {
	// CreatedAt และ UpdatedAt เป็น *time.Time (nullable TIMESTAMPTZ)
	// ใช้ deref ด้วย zero value ถ้า nil
	var createdAt, updatedAt time.Time
	if s.CreatedAt != nil {
		createdAt = *s.CreatedAt
	}
	if s.UpdatedAt != nil {
		updatedAt = *s.UpdatedAt
	}
	return dto.SubtaskResponse{
		ID:        s.ID,
		CardID:    s.CardID,
		Title:     s.Title,
		IsDone:    s.IsDone,
		Position:  s.Position,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func ToSubtaskResponses(subtasks []db.CardSubtask) []dto.SubtaskResponse {
	result := make([]dto.SubtaskResponse, len(subtasks))
	for i, s := range subtasks {
		result[i] = ToSubtaskResponse(s)
	}
	return result
}

// timePtrToString serializes *time.Time as "YYYY-MM-DD".
// DB schema stores due_date as DATE (no time component), so date-only format is correct.
// Input that includes time-of-day (RFC3339) is accepted by util.PtrStringToTimePtr
// but the time portion is discarded when saved to DB — this matches that behavior.
func timePtrToString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func timePtrToRFC3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func ToCardResponse(card service.CardData) dto.CardResponse {
	tags := make([]dto.TagResponse, len(card.Tags))
	for i, t := range card.Tags {
		tags[i] = dto.TagResponse{ID: t.ID, BoardID: t.BoardID, Name: t.Name, Color: t.Color}
	}
	return dto.CardResponse{
		ID:                card.ID,
		ColumnID:          card.ColumnID,
		Title:             card.Title,
		Description:       card.Description,
		Position:          card.Position,
		DueDate:           timePtrToString(card.DueDate),
		EstimatedHours:    card.EstimatedHours,
		AssigneeID:        card.AssigneeID,
		AssigneeName:      card.AssigneeName,
		Priority:          card.Priority,
		IsDone:            card.IsDone,
		CompletedAt:       timePtrToRFC3339(card.CompletedAt),
		CreatedAt:         timePtrToRFC3339(card.CreatedAt),
		CreatedBy:         card.CreatedBy,
		TotalSubtasks:     card.TotalSubtasks,
		CompletedSubtasks: card.CompletedSubtasks,
		Tags:              tags,
	}
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// ToCardDetailResponse builds the enriched single-card payload (GET /cards/:id).
func ToCardDetailResponse(d service.CardDetailData) dto.CardDetailResponse {
	c := d.Card
	subs := make([]dto.SubtaskResponse, len(d.Subtasks))
	var completed int64
	for i, st := range d.Subtasks {
		subs[i] = dto.SubtaskResponse{
			ID:        st.ID,
			CardID:    st.CardID,
			Title:     st.Title,
			IsDone:    st.IsDone,
			Position:  st.Position,
			CreatedAt: derefTime(st.CreatedAt),
			UpdatedAt: derefTime(st.UpdatedAt),
		}
		if st.IsDone {
			completed++
		}
	}
	tags := make([]dto.TagResponse, len(d.Tags))
	for i, t := range d.Tags {
		tags[i] = dto.TagResponse{ID: t.ID, BoardID: t.BoardID, Name: t.Name, Color: t.Color}
	}
	return dto.CardDetailResponse{
		ID:                 c.ID,
		ColumnID:           c.ColumnID,
		Title:              c.Title,
		Description:        c.Description,
		DueDate:            timePtrToString(c.DueDate),
		EstimatedHours:     util.PgNumericToFloat64Ptr(c.EstimatedHours),
		AssigneeID:         c.AssigneeID,
		AssigneeName:       d.AssigneeName,
		Priority:           c.Priority,
		IsDone:             c.IsDone,
		AcceptanceCriteria: c.AcceptanceCriteria,
		ImplementationNote: c.ImplementationNote,
		TotalSubtasks:      int64(len(d.Subtasks)),
		CompletedSubtasks:  completed,
		Subtasks:           subs,
		Tags:               tags,
	}
}

func ToColumnResponse(col service.ColumnData) dto.ColumnResponse {
	cards := make([]dto.CardResponse, 0, len(col.Cards))
	for _, card := range col.Cards {
		cards = append(cards, ToCardResponse(card))
	}
	return dto.ColumnResponse{
		ID:       col.ID,
		Title:    col.Title,
		Position: col.Position,
		Category: col.Category,
		Color:    col.Color,
		Cards:    cards,
	}
}

func ToColumnResponses(columns []service.ColumnData) []dto.ColumnResponse {
	result := make([]dto.ColumnResponse, 0, len(columns))
	for _, col := range columns {
		result = append(result, ToColumnResponse(col))
	}
	return result
}
