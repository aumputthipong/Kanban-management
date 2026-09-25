//go:build integration

package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateItem_SecondItem_LandsAfterFirstByTheGap(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	second, err := f.svc.CreateItem(ctx, f.sessID, "DEC", "Second item", nil)
	require.NoError(t, err)
	assert.Greater(t, second.Position, 65536.0, "a second item must land after the first, not collide with it")
}

func TestGetCardSource_NeverPromoted_ReturnsNil(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	cardID := f.seed.Card(ctx, f.todoID)

	source, err := f.svc.GetCardSource(ctx, cardID, 3)
	require.NoError(t, err)
	assert.Nil(t, source, "a card nobody promoted from planning must report no source, not an error")
}

func TestGetCardSource_Promoted_ReturnsOriginatingSessionAndItem(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	_, card, err := f.svc.PromoteItem(ctx, f.itemID, f.userID)
	require.NoError(t, err)

	source, err := f.svc.GetCardSource(ctx, card.ID, 3)
	require.NoError(t, err)
	require.NotNil(t, source)
	assert.Equal(t, f.sessID, source.SessionID)
	assert.Equal(t, f.itemID, source.ItemID)
}
