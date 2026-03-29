package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
)

func TestPlaidSetAccountBalance_Perform_Success(t *testing.T) {
	a := &PlaidSetAccountBalance{
		AccountID: uuid.Must(uuid.NewV4()),
		Balance:   decimal.NewFromFloat(1234.56),
	}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		UpdateBalance(mock.Anything, a.AccountID, a.Balance).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount

	require.NoError(t, a.Perform(context.Background(), wt))
	mockAccount.AssertExpectations(t)
}

func TestPlaidSetAccountBalance_Perform_UpdateBalanceError(t *testing.T) {
	a := &PlaidSetAccountBalance{
		AccountID: uuid.Must(uuid.NewV4()),
		Balance:   decimal.NewFromFloat(500.00),
	}
	updateErr := errors.New("update failed")

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		UpdateBalance(mock.Anything, a.AccountID, a.Balance).
		Return(updateErr)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount

	assert.ErrorIs(t, a.Perform(context.Background(), wt), updateErr)
}
