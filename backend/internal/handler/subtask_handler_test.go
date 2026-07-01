package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/dto"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validCardID and chiCtx/withUserID helpers are defined in board_handler_test.go (same package)

const validSubtaskID = "b2c3d4e5-f6a7-8901-bcde-f12345678901"

// ────────────────────────────────────────────────
// CreateSubtask
// ────────────────────────────────────────────────

func TestCreateSubtask_Success(t *testing.T) {
	svc := &mock.MockSubtaskService{
		CreateSubtaskFn: func(ctx context.Context, arg db.CreateSubtaskParams) (db.CardSubtask, error) {
			assert.Equal(t, validCardID, arg.CardID)
			assert.Equal(t, "Write tests", arg.Title)
			return db.CardSubtask{ID: validSubtaskID, CardID: validCardID, Title: "Write tests", Position: 1.0}, nil
		},
	}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{"title":"Write tests","position":1}`)
	req := httptest.NewRequest(http.MethodPost, "/cards/"+validCardID+"/subtasks", body)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validSubtaskID, res["id"])
}

func TestCreateSubtask_InvalidJSON(t *testing.T) {
	svc := &mock.MockSubtaskService{}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest(http.MethodPost, "/cards/"+validCardID+"/subtasks", body)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSubtask_ServiceError(t *testing.T) {
	svc := &mock.MockSubtaskService{
		CreateSubtaskFn: func(ctx context.Context, arg db.CreateSubtaskParams) (db.CardSubtask, error) {
			return db.CardSubtask{}, errors.New("db error")
		},
	}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{"title":"Write tests","position":1}`)
	req := httptest.NewRequest(http.MethodPost, "/cards/"+validCardID+"/subtasks", body)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateSubtask)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// GetSubtasks
// ────────────────────────────────────────────────

func TestGetSubtasks_Success(t *testing.T) {
	svc := &mock.MockSubtaskService{
		GetSubtasksByCardIDFn: func(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
			assert.Equal(t, validCardID, cardID)
			return []db.CardSubtask{
				{ID: validSubtaskID, CardID: validCardID, Title: "Step 1", Position: 1.0},
			}, nil
		},
	}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID+"/subtasks", nil)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Len(t, res, 1)
	assert.Equal(t, validSubtaskID, res[0]["id"])
}

func TestGetSubtasks_MissingCardID(t *testing.T) {
	svc := &mock.MockSubtaskService{}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/cards/subtasks", nil)
	// No chiCtx injected → chi.URLParam returns "".
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSubtasks_ServiceError(t *testing.T) {
	svc := &mock.MockSubtaskService{
		GetSubtasksByCardIDFn: func(ctx context.Context, cardID string) ([]db.CardSubtask, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID+"/subtasks", nil)
	req = chiCtx(req, "cardID", validCardID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtasks)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// GetSubtask
// ────────────────────────────────────────────────

func TestGetSubtask_Success(t *testing.T) {
	svc := &mock.MockSubtaskService{
		GetSubtaskByIDFn: func(ctx context.Context, subtaskID string) (db.CardSubtask, error) {
			assert.Equal(t, validSubtaskID, subtaskID)
			return db.CardSubtask{ID: validSubtaskID, CardID: validCardID, Title: "Step 1", Position: 1.0}, nil
		},
	}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, nil)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validSubtaskID, res["id"])
}

func TestGetSubtask_MissingSubtaskID(t *testing.T) {
	svc := &mock.MockSubtaskService{}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID+"/subtasks/", nil)
	// No chiCtx injected → chi.URLParam returns "".
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSubtask_NotFound(t *testing.T) {
	svc := &mock.MockSubtaskService{
		GetSubtaskByIDFn: func(ctx context.Context, subtaskID string) (db.CardSubtask, error) {
			return db.CardSubtask{}, errors.New("subtask not found")
		},
	}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, nil)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetSubtask)(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ────────────────────────────────────────────────
// UpdateSubtask
// ────────────────────────────────────────────────

func TestUpdateSubtask_Success(t *testing.T) {
	svc := &mock.MockSubtaskService{
		UpdateSubtaskFn: func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
			assert.Equal(t, validSubtaskID, subtaskID)
			return db.CardSubtask{ID: validSubtaskID, CardID: validCardID, Title: "Done", IsDone: true, Position: 1.0}, nil
		},
	}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{"is_done":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, body)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validSubtaskID, res["id"])
}

func TestUpdateSubtask_MissingSubtaskID(t *testing.T) {
	svc := &mock.MockSubtaskService{}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{"is_done":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID+"/subtasks/", body)
	// No chiCtx injected → chi.URLParam returns "".
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubtask_InvalidJSON(t *testing.T) {
	svc := &mock.MockSubtaskService{}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{bad json}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, body)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSubtask_ServiceError(t *testing.T) {
	svc := &mock.MockSubtaskService{
		UpdateSubtaskFn: func(ctx context.Context, subtaskID string, req dto.UpdateSubtaskRequest) (db.CardSubtask, error) {
			return db.CardSubtask{}, errors.New("db error")
		},
	}
	h := NewSubtaskHandler(svc)
	body := strings.NewReader(`{"is_done":true}`)
	req := httptest.NewRequest(http.MethodPatch, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, body)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.UpdateSubtask)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ────────────────────────────────────────────────
// DeleteSubtask
// ────────────────────────────────────────────────

func TestDeleteSubtask_Success(t *testing.T) {
	called := false
	svc := &mock.MockSubtaskService{
		DeleteSubtaskFn: func(ctx context.Context, subtaskID string) error {
			assert.Equal(t, validSubtaskID, subtaskID)
			called = true
			return nil
		},
	}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodDelete, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, nil)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, called)
}

func TestDeleteSubtask_MissingSubtaskID(t *testing.T) {
	svc := &mock.MockSubtaskService{}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodDelete, "/cards/"+validCardID+"/subtasks/", nil)
	// No chiCtx injected → chi.URLParam returns "".
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteSubtask_ServiceError(t *testing.T) {
	svc := &mock.MockSubtaskService{
		DeleteSubtaskFn: func(ctx context.Context, subtaskID string) error {
			return errors.New("db error")
		},
	}
	h := NewSubtaskHandler(svc)
	req := httptest.NewRequest(http.MethodDelete, "/cards/"+validCardID+"/subtasks/"+validSubtaskID, nil)
	req = chiCtx(req, "subtaskID", validSubtaskID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteSubtask)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
