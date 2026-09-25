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
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/httputil"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validTagID = "c3d4e5f6-a7b8-9012-cdef-123456789012"

func newTagRequest(method, target, body string, params ...string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	if len(params) == 0 {
		params = []string{"boardID", validBoardID}
	}
	return chiCtx(r, params...)
}

func errorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var res httputil.ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	return res.Error
}

// GetBoardTags

func TestGetBoardTags_Success(t *testing.T) {
	// Only GetTagsByBoardFn is set — any other call panics.
	svc := &mock.MockTagService{
		GetTagsByBoardFn: func(ctx context.Context, boardID string) ([]db.Tag, error) {
			assert.Equal(t, validBoardID, boardID)
			return []db.Tag{
				{ID: validTagID, BoardID: validBoardID, Name: "bug", Color: "#EF4444"},
			}, nil
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodGet, "/boards/"+validBoardID+"/tags", "")
	w := httptest.NewRecorder() // a fake ResponseWriter that records what was written

	httputil.MakeHandler(h.GetBoardTags)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var res []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	require.Len(t, res, 1)
	assert.Equal(t, "bug", res[0]["name"])
	assert.Equal(t, "#EF4444", res[0]["color"])
}

// Must serialise as [], not null.
func TestGetBoardTags_NoTags_ReturnsEmptyArray(t *testing.T) {
	svc := &mock.MockTagService{
		GetTagsByBoardFn: func(ctx context.Context, boardID string) ([]db.Tag, error) {
			return []db.Tag{}, nil
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodGet, "/boards/"+validBoardID+"/tags", "")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardTags)(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `[]`, w.Body.String())
}

func TestGetBoardTags_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockTagService{
		GetTagsByBoardFn: func(ctx context.Context, boardID string) ([]db.Tag, error) {
			return nil, errors.New(`pq: relation "tags" does not exist`)
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodGet, "/boards/"+validBoardID+"/tags", "")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardTags)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to fetch tags", errorMessage(t, w))
}

func TestGetBoardTags_InvalidBoardID_Returns400(t *testing.T) {
	// No Fn set: a malformed id must be rejected before the service.
	h := NewTagHandler(&mock.MockTagService{}, nil)

	req := newTagRequest(http.MethodGet, "/boards/not-a-uuid/tags", "", "boardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardTags)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// CreateBoardTag

func TestCreateBoardTag_Success_Returns201(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			assert.Equal(t, validBoardID, boardID)
			assert.Equal(t, "bug", name)
			assert.Equal(t, "#EF4444", color)
			return db.Tag{ID: validTagID, BoardID: boardID, Name: name, Color: color}, nil
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"bug","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	assert.Equal(t, validTagID, res["id"])
	assert.Equal(t, "bug", res["name"])
}

// "   " passes `min=1`; the service's trim-then-check catches it.
func TestCreateBoardTag_BlankName_Returns422(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			return db.Tag{}, service.ErrTagNameEmpty
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"   ","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "tag name cannot be empty", errorMessage(t, w))
}

func TestCreateBoardTag_NameTooLong_Returns422(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			return db.Tag{}, service.ErrTagNameTooLong
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"release","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "tag name too long (max 50 chars)", errorMessage(t, w))
}

// Only the two sentinels are echoed — never driver text.
func TestCreateBoardTag_ServiceError_Returns500WithoutLeakingDBText(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			return db.Tag{}, errors.New(`pq: duplicate key value violates unique constraint "tags_board_id_name_key"`)
		},
	}
	h := NewTagHandler(svc, nil)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"bug","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	msg := errorMessage(t, w)
	assert.Equal(t, "Failed to create tag", msg)
	assert.NotContains(t, msg, "constraint", "raw DB error must not reach the client")
}

func TestCreateBoardTag_EmptyName_RejectedByValidator_Returns400(t *testing.T) {
	h := NewTagHandler(&mock.MockTagService{}, nil)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateBoardTag_NonHexColor_Returns400(t *testing.T) {
	h := NewTagHandler(&mock.MockTagService{}, nil)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"bug","color":"red"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// DeleteBoardTag

func TestDeleteBoardTag_Success_Returns204(t *testing.T) {
	called := false
	svc := &mock.MockTagService{
		DeleteTagFn: func(ctx context.Context, boardID, tagID string) error {
			called = true
			// Scoped by board, so a foreign tag id can't be deleted.
			assert.Equal(t, validBoardID, boardID)
			assert.Equal(t, validTagID, tagID)
			return nil
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewTagHandler(svc, bc)

	req := newTagRequest(http.MethodDelete, "/boards/"+validBoardID+"/tags/"+validTagID, "",
		"boardID", validBoardID, "tagID", validTagID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteBoardTag)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String(), "204 must carry no body")
	assert.True(t, called, "the delete must actually reach the service")

	require.Len(t, bc.Sent, 1)
	assert.Equal(t, validBoardID, bc.Sent[0].BoardID)
	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			TagID string `json:"tag_id"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(bc.Sent[0].Message, &msg))
	assert.Equal(t, "TAG_DELETED", msg.Type)
	assert.Equal(t, validTagID, msg.Payload.TagID)
}

func TestDeleteBoardTag_InvalidTagID_Returns400(t *testing.T) {
	h := NewTagHandler(&mock.MockTagService{}, nil)

	req := newTagRequest(http.MethodDelete, "/boards/"+validBoardID+"/tags/not-a-uuid", "",
		"boardID", validBoardID, "tagID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteBoardTag)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Invalid tag ID", errorMessage(t, w))
}

func TestDeleteBoardTag_ServiceError_Returns500(t *testing.T) {
	svc := &mock.MockTagService{
		DeleteTagFn: func(ctx context.Context, boardID, tagID string) error {
			return errors.New("db error")
		},
	}
	bc := &mock.MockBroadcaster{}
	h := NewTagHandler(svc, bc)

	req := newTagRequest(http.MethodDelete, "/boards/"+validBoardID+"/tags/"+validTagID, "",
		"boardID", validBoardID, "tagID", validTagID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteBoardTag)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to delete tag", errorMessage(t, w))
	assert.Empty(t, bc.Sent, "a failed delete must not tell clients to drop the tag")
}
