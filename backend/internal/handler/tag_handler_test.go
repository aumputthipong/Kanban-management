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

// chiCtx, withUserID and validBoardID live in board_handler_test.go (same package).

const validTagID = "c3d4e5f6-a7b8-9012-cdef-123456789012"

// newTagRequest builds a request already carrying the chi URL params the tag
// routes declare, so each test only has to say what it is actually varying.
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

// errorMessage pulls the message out of the standard {"error": "..."} body.
func errorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var res httputil.ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	return res.Error
}

// ────────────────────────────────────────────────
// GetBoardTags
// ────────────────────────────────────────────────

func TestGetBoardTags_Success(t *testing.T) {
	// ── ARRANGE ──
	// The mock stands in for the real TagService. Setting only GetTagsByBoardFn
	// is deliberate: if the handler ever calls CreateTag or DeleteTag on this
	// path, the nil stub panics and the test tells us instead of passing.
	svc := &mock.MockTagService{
		GetTagsByBoardFn: func(ctx context.Context, boardID string) ([]db.Tag, error) {
			// Asserting on the argument checks the half of the contract the
			// response body cannot show: that the handler passed the board id
			// from the URL through unchanged.
			assert.Equal(t, validBoardID, boardID)
			return []db.Tag{
				{ID: validTagID, BoardID: validBoardID, Name: "bug", Color: "#EF4444"},
			}, nil
		},
	}
	h := NewTagHandler(svc)

	req := newTagRequest(http.MethodGet, "/boards/"+validBoardID+"/tags", "")
	w := httptest.NewRecorder() // a fake ResponseWriter that records what was written

	// ── ACT ──
	httputil.MakeHandler(h.GetBoardTags)(w, req)

	// ── ASSERT ──
	assert.Equal(t, http.StatusOK, w.Code)

	var res []map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&res))
	require.Len(t, res, 1)
	assert.Equal(t, "bug", res[0]["name"])
	assert.Equal(t, "#EF4444", res[0]["color"])
}

// An empty board must serialise as [] and not null — the frontend maps over
// this array directly.
func TestGetBoardTags_NoTags_ReturnsEmptyArray(t *testing.T) {
	svc := &mock.MockTagService{
		GetTagsByBoardFn: func(ctx context.Context, boardID string) ([]db.Tag, error) {
			return []db.Tag{}, nil
		},
	}
	h := NewTagHandler(svc)

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
	h := NewTagHandler(svc)

	req := newTagRequest(http.MethodGet, "/boards/"+validBoardID+"/tags", "")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardTags)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to fetch tags", errorMessage(t, w))
}

func TestGetBoardTags_InvalidBoardID_Returns400(t *testing.T) {
	// No Fn is set: reaching the service at all would panic, which is exactly
	// the assertion we want — a malformed id must be rejected before any work.
	h := NewTagHandler(&mock.MockTagService{})

	req := newTagRequest(http.MethodGet, "/boards/not-a-uuid/tags", "", "boardID", "not-a-uuid")
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.GetBoardTags)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ────────────────────────────────────────────────
// CreateBoardTag
// ────────────────────────────────────────────────

func TestCreateBoardTag_Success_Returns201(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			assert.Equal(t, validBoardID, boardID)
			assert.Equal(t, "bug", name)
			assert.Equal(t, "#EF4444", color)
			return db.Tag{ID: validTagID, BoardID: boardID, Name: name, Color: color}, nil
		},
	}
	h := NewTagHandler(svc)

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

// A blank-but-not-empty name is the only way ErrTagNameEmpty is reachable: the
// validator's `min=1` already rejects "", so the service's own trim-then-check
// is what catches "   ". Both layers matter, and this test pins the second one.
func TestCreateBoardTag_BlankName_Returns422(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			return db.Tag{}, service.ErrTagNameEmpty
		},
	}
	h := NewTagHandler(svc)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"   ","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	// Sentinel text is deliberately surfaced verbatim — it is written for users.
	assert.Equal(t, "tag name cannot be empty", errorMessage(t, w))
}

func TestCreateBoardTag_NameTooLong_Returns422(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			return db.Tag{}, service.ErrTagNameTooLong
		},
	}
	h := NewTagHandler(svc)

	// 20 Thai characters: 20 runes, 60 bytes. The validator's max=50 counts
	// runes so this passes, while TagService's len(name) counts bytes so it
	// rejects — the service is the layer that decides here.
	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"`+strings.Repeat("ก", 20)+`","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "tag name too long (max 50 chars)", errorMessage(t, w))
}

// The two sentinels above are safe to echo; anything else is not. This pins the
// default branch so a future refactor can't start leaking driver text.
func TestCreateBoardTag_ServiceError_Returns500WithoutLeakingDBText(t *testing.T) {
	svc := &mock.MockTagService{
		CreateTagFn: func(ctx context.Context, boardID, name, color string) (db.Tag, error) {
			return db.Tag{}, errors.New(`pq: duplicate key value violates unique constraint "tags_board_id_name_key"`)
		},
	}
	h := NewTagHandler(svc)

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"bug","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	msg := errorMessage(t, w)
	assert.Equal(t, "Failed to create tag", msg)
	assert.NotContains(t, msg, "constraint", "raw DB error must not reach the client")
}

// The validator runs before the service, so a name that fails the DTO rules is
// a 400 and never becomes a service call at all.
func TestCreateBoardTag_EmptyName_RejectedByValidator_Returns400(t *testing.T) {
	h := NewTagHandler(&mock.MockTagService{}) // no Fn set: a service call would panic

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"","color":"#EF4444"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// color carries `validate:"hexcolor"`, so a plain word is rejected at the DTO
// boundary too.
func TestCreateBoardTag_NonHexColor_Returns400(t *testing.T) {
	h := NewTagHandler(&mock.MockTagService{})

	req := newTagRequest(http.MethodPost, "/boards/"+validBoardID+"/tags",
		`{"name":"bug","color":"red"}`)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.CreateBoardTag)(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ────────────────────────────────────────────────
// DeleteBoardTag
// ────────────────────────────────────────────────

func TestDeleteBoardTag_Success_Returns204(t *testing.T) {
	called := false
	svc := &mock.MockTagService{
		DeleteTagFn: func(ctx context.Context, boardID, tagID string) error {
			called = true
			// Both ids must be forwarded: the query scopes the delete by board
			// so a tag id from another board cannot be removed.
			assert.Equal(t, validBoardID, boardID)
			assert.Equal(t, validTagID, tagID)
			return nil
		},
	}
	h := NewTagHandler(svc)

	req := newTagRequest(http.MethodDelete, "/boards/"+validBoardID+"/tags/"+validTagID, "",
		"boardID", validBoardID, "tagID", validTagID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteBoardTag)(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String(), "204 must carry no body")
	assert.True(t, called, "the delete must actually reach the service")
}

func TestDeleteBoardTag_InvalidTagID_Returns400(t *testing.T) {
	h := NewTagHandler(&mock.MockTagService{}) // a service call here would panic

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
	h := NewTagHandler(svc)

	req := newTagRequest(http.MethodDelete, "/boards/"+validBoardID+"/tags/"+validTagID, "",
		"boardID", validBoardID, "tagID", validTagID)
	w := httptest.NewRecorder()

	httputil.MakeHandler(h.DeleteBoardTag)(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "Failed to delete tag", errorMessage(t, w))
}
