//go:build integration

// Integration tests for UserSettingsService. Get's first-read-materializes
// behavior and Update's PATCH semantics both hinge on the UpsertUserSettings
// query's own COALESCE-against-column-defaults, which only a real Postgres
// round-trip can confirm.
package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aumputthipong/mini-erp-kanban/backend/internal/db"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/testutil"
)

func newSettingsFixture(t *testing.T) (*service.UserSettingsService, string) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.NewTestDB(t)
	queries := db.New(pool)
	seed := testutil.NewSeed(t, pool)
	return service.NewUserSettingsService(queries), seed.User(ctx)
}

func TestUserSettingsGet_FirstRead_MaterializesDefaults(t *testing.T) {
	ctx := context.Background()
	svc, userID := newSettingsFixture(t)

	settings, err := svc.Get(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "today", settings.DefaultLanding)
	assert.False(t, settings.ShowAllCards)
	assert.Equal(t, "Asia/Bangkok", settings.Timezone)
}

func TestUserSettingsGet_SecondRead_ReturnsSameRowNotReset(t *testing.T) {
	ctx := context.Background()
	svc, userID := newSettingsFixture(t)
	newLanding := "my_work"
	_, err := svc.Update(ctx, userID, service.UpdateUserSettingsParams{DefaultLanding: &newLanding})
	require.NoError(t, err)

	settings, err := svc.Get(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "my_work", settings.DefaultLanding, "a later Get must not re-materialize defaults over a prior Update")
}

func TestUserSettingsUpdate_PartialUpdate_PreservesOtherFields(t *testing.T) {
	ctx := context.Background()
	svc, userID := newSettingsFixture(t)
	_, err := svc.Get(ctx, userID) // materialize defaults first
	require.NoError(t, err)

	showAll := true
	updated, err := svc.Update(ctx, userID, service.UpdateUserSettingsParams{ShowAllCards: &showAll})
	require.NoError(t, err)
	assert.True(t, updated.ShowAllCards)
	assert.Equal(t, "today", updated.DefaultLanding, "an update that only touches show_all_cards must not reset default_landing")
	assert.Equal(t, "Asia/Bangkok", updated.Timezone)
}

func TestUserSettingsUpdate_InvalidLanding_Rejected(t *testing.T) {
	ctx := context.Background()
	svc, userID := newSettingsFixture(t)
	bogus := "not_a_real_landing"

	_, err := svc.Update(ctx, userID, service.UpdateUserSettingsParams{DefaultLanding: &bogus})
	assert.ErrorIs(t, err, service.ErrInvalidLanding)
}
