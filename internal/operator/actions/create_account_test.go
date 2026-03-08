package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
)

func TestCreateAccount_Perform_Success(t *testing.T) {
	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		Create(
			mock.Anything,
			"Checking",
			account.AccountTypeCash,
			"test sub type",
			decimal.Zero,
		).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	action := &CreateAccount{
		Name:            "Checking",
		Type:            account.AccountTypeCash,
		SubType:         "test sub type",
		StartingBalance: decimal.Zero,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockAccount.AssertExpectations(t)
}

func TestCreateAccount_Perform_CreateFails(t *testing.T) {
	createErr := errors.New("create failed")
	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		Create(
			mock.Anything,
			"Savings",
			account.AccountTypeAssets,
			"High Yield",
			decimal.NewFromInt(1000),
		).
		Return(createErr)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	action := &CreateAccount{
		Name:            "Savings",
		Type:            account.AccountTypeAssets,
		SubType:         "High Yield",
		StartingBalance: decimal.NewFromInt(1000),
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, createErr)
	mockAccount.AssertExpectations(t)
}
