package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
)

func TestPlaidUpdateCursor_Perform_Success(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		UpdateCursor(mock.Anything, itemID, "next-cursor-abc").
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Plaid = mockPlaid

	a := &PlaidUpdateCursor{ItemID: itemID, NextCursor: "next-cursor-abc"}
	require.NoError(t, a.Perform(context.Background(), wt))
	mockPlaid.AssertExpectations(t)
}

func TestPlaidUpdateCursor_Perform_Error(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())
	updateErr := errors.New("cursor update failed")

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		UpdateCursor(mock.Anything, itemID, "next-cursor-abc").
		Return(updateErr)

	wt := storage.NewWriterForTest()
	wt.Plaid = mockPlaid

	a := &PlaidUpdateCursor{ItemID: itemID, NextCursor: "next-cursor-abc"}
	assert.ErrorIs(t, a.Perform(context.Background(), wt), updateErr)
}
