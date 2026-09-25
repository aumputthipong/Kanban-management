package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
)

const (
	EventCardCreated     = "card.created"
	EventCardMoved       = "card.moved"
	EventCardUpdated     = "card.updated"
	EventCardDeleted     = "card.deleted"
	EventCardDoneToggled = "card.done_toggled"
	EventColumnCreated   = "column.created"
	EventColumnDeleted   = "column.deleted"
	EventColumnRenamed   = "column.renamed"
	EventMemberAdded     = "member.added"
	EventMemberRemoved   = "member.removed"
	EventMemberLeft      = "member.left"
	EventMemberRole      = "member.role_changed"
	// Only when the last open subtask is ticked — single ticks are noise.
	EventCardSubtasksCompleted = "card.subtasks_completed"

	// PromoteItem logs only the planning side, never a duplicate card event.
	EventPlanningSessionCreated = "planning.session_created"
	EventPlanningSessionUpdated = "planning.session_updated"
	EventPlanningSessionDeleted = "planning.session_deleted"
	EventPlanningItemCreated    = "planning.item_created"
	EventPlanningItemUpdated    = "planning.item_updated"
	EventPlanningItemDeleted    = "planning.item_deleted"
	EventPlanningItemPromoted   = "planning.item_promoted"

	EventPlanningCommentCreated = "planning.comment_created"
	EventPlanningCommentEdited  = "planning.comment_edited"
	EventPlanningCommentDeleted = "planning.comment_deleted"

	// Legacy: no longer emitted, kept for historical rows (see AGENTS.md).
	EventPlanningItemClaimed           = "planning.item_claimed"
	EventPlanningItemReleased          = "planning.item_released"
	EventPlanningItemClaimAutoReleased = "planning.claim_auto_released_on_promote"

	EntityCard            = "card"
	EntityColumn          = "column"
	EntityMember          = "member"
	EntityPlanningSession = "planning_session"
	EntityPlanningItem    = "planning_item"
	EntityPlanningComment = "planning_comment"
)

type ActivityService struct {
	queries *db.Queries
	jobs    chan RecordParams
	stop    chan struct{}
}

const (
	activityQueueSize    = 512
	activityWriteTimeout = 5 * time.Second
)

func NewActivityService(queries *db.Queries) *ActivityService {
	s := &ActivityService{
		queries: queries,
		jobs:    make(chan RecordParams, activityQueueSize),
		stop:    make(chan struct{}),
	}
	go s.worker()
	return s
}

// Fresh background context per job, so a slow insert outlives its request.
func (s *ActivityService) worker() {
	for {
		select {
		case <-s.stop:
			for {
				select {
				case p := <-s.jobs:
					s.writeOne(p)
				default:
					return
				}
			}
		case p := <-s.jobs:
			s.writeOne(p)
		}
	}
}

func (s *ActivityService) writeOne(p RecordParams) {
	ctx, cancel := context.WithTimeout(context.Background(), activityWriteTimeout)
	defer cancel()
	if _, err := s.Record(ctx, p); err != nil {
		slog.Warn("async activity record failed", "event_type", p.EventType, "err", err)
	}
}

// Idempotent. Call after the HTTP listener has drained.
func (s *ActivityService) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
}

// Best-effort: a full queue drops the job. Use Record when a broadcast needs the row.
func (s *ActivityService) RecordAsync(p RecordParams) {
	select {
	case s.jobs <- p:
	default:
		slog.Warn("activity queue full, dropping record", "event_type", p.EventType)
	}
}

type RecordParams struct {
	BoardID    string
	ActorID    string
	EventType  string
	EntityType string
	EntityID   *string
	Payload    any
}

func (s *ActivityService) Record(ctx context.Context, p RecordParams) (db.Activity, error) {
	var payloadBytes []byte
	if p.Payload != nil {
		b, err := json.Marshal(p.Payload)
		if err != nil {
			return db.Activity{}, err
		}
		payloadBytes = b
	} else {
		payloadBytes = []byte("{}")
	}
	return s.queries.CreateActivity(ctx, db.CreateActivityParams{
		BoardID:    p.BoardID,
		ActorID:    p.ActorID,
		EventType:  p.EventType,
		EntityType: p.EntityType,
		EntityID:   p.EntityID,
		Payload:    payloadBytes,
	})
}

type ActivityItem struct {
	ID         string
	BoardID    string
	ActorID    string
	ActorName  *string
	EventType  string
	EntityType string
	EntityID   *string
	Payload    []byte
	CreatedAt  time.Time
}

func (s *ActivityService) List(ctx context.Context, boardID string, before *time.Time, limit int32) ([]ActivityItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if before != nil {
		rows, err := s.queries.ListActivitiesByBoardBefore(ctx, db.ListActivitiesByBoardBeforeParams{
			BoardID:   boardID,
			CreatedAt: *before,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		out := make([]ActivityItem, len(rows))
		for i, r := range rows {
			out[i] = ActivityItem{
				ID: r.ID, BoardID: r.BoardID, ActorID: r.ActorID, ActorName: r.ActorName,
				EventType: r.EventType, EntityType: r.EntityType, EntityID: r.EntityID,
				Payload: r.Payload, CreatedAt: r.CreatedAt,
			}
		}
		return out, nil
	}
	rows, err := s.queries.ListActivitiesByBoard(ctx, db.ListActivitiesByBoardParams{
		BoardID: boardID,
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ActivityItem, len(rows))
	for i, r := range rows {
		out[i] = ActivityItem{
			ID: r.ID, BoardID: r.BoardID, ActorID: r.ActorID, ActorName: r.ActorName,
			EventType: r.EventType, EntityType: r.EntityType, EntityID: r.EntityID,
			Payload: r.Payload, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

type CardCreatedPayload struct {
	Title    string `json:"title"`
	ColumnID string `json:"column_id"`
}

type CardMovedPayload struct {
	Title        string `json:"title"`
	FromColumnID string `json:"from_column_id"`
	ToColumnID   string `json:"to_column_id"`
}

type CardUpdatedPayload struct {
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
}

type CardDeletedPayload struct {
	Title string `json:"title"`
}

type CardDoneToggledPayload struct {
	Title  string `json:"title"`
	IsDone bool   `json:"is_done"`
}

// Carries the name so the feed still reads after the member leaves.
type MemberChangedPayload struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	PreviousRole string `json:"previous_role,omitempty"`
	Via          string `json:"via,omitempty"`
}

type CardSubtasksCompletedPayload struct {
	Title string `json:"title"`
	Total int    `json:"total"`
}

type ColumnCreatedPayload struct {
	Title string `json:"title"`
}

type ColumnDeletedPayload struct {
	Title string `json:"title"`
}

type ColumnRenamedPayload struct {
	OldTitle string `json:"old_title"`
	NewTitle string `json:"new_title"`
}

type PlanningSessionCreatedPayload struct {
	Title string `json:"title"`
}

type PlanningSessionUpdatedPayload struct {
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
}

type PlanningSessionDeletedPayload struct {
	Title string `json:"title"`
}

type PlanningItemCreatedPayload struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

type PlanningCommentCreatedPayload struct {
	ItemID      string `json:"item_id"`
	BodyPreview string `json:"body_preview"`
}

type PlanningCommentEditedPayload struct {
	ItemID      string `json:"item_id"`
	BodyPreview string `json:"body_preview"`
}

type PlanningCommentDeletedPayload struct {
	ItemID string `json:"item_id"`
}

type PlanningItemClaimedPayload struct {
	Title string `json:"title"`
	Type  string `json:"type"`
}

type PlanningItemReleasedPayload struct {
	Title string `json:"title"`
	Type  string `json:"type"`
}

type PlanningItemUpdatedPayload struct {
	Type   string   `json:"type"`
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
	// Only set when "type" is in Fields.
	PreviousType string `json:"previous_type,omitempty"`
}

type PlanningItemDeletedPayload struct {
	Type  string `json:"type"`
	Title string `json:"title"`
}

type PlanningItemPromotedPayload struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	ToCardID string `json:"to_card_id"`
}
