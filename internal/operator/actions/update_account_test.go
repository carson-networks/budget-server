package actions

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
)

func TestUpdateAccount_Perform_Success(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	newName := "Renamed Checking"
	newSubType := "High Yield"
	newStarting := decimal.NewFromInt(250)
	existing := &account.Account{
		ID:              accID,
		Name:            "Checking",
		Type:            account.AccountTypeCash,
		SubType:         "Checking",
		Balance:         decimal.NewFromInt(100),
		StartingBalance: decimal.Zero,
	}
	// delta = 250 - 0 = 250; new balance = 100 + 250 = 350
	expectedBalance := decimal.NewFromInt(350)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)
	mockAccount.EXPECT().
		Update(mock.Anything, accID, mock.MatchedBy(func(u *account.AccountUpdate) bool {
			return u != nil &&
				u.Name != nil && *u.Name == newName &&
				u.SubType != nil && *u.SubType == newSubType &&
				u.StartingBalance != nil && u.StartingBalance.Equal(newStarting) &&
				u.Balance != nil && u.Balance.Equal(expectedBalance)
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	action := &UpdateAccount{
		ID:              accID,
		Name:            &newName,
		SubType:         &newSubType,
		StartingBalance: &newStarting,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockAccount.AssertExpectations(t)
}

func TestUpdateAccount_Perform_StartingBalanceAdjustsBalanceByDelta(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	newStarting := decimal.NewFromInt(50)
	existing := &account.Account{
		ID:              accID,
		Name:            "Checking",
		Type:            account.AccountTypeCash,
		Balance:         decimal.NewFromInt(200),
		StartingBalance: decimal.NewFromInt(100),
	}
	// delta = 50 - 100 = -50; new balance = 200 - 50 = 150
	expectedBalance := decimal.NewFromInt(150)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)
	mockAccount.EXPECT().
		Update(mock.Anything, accID, mock.MatchedBy(func(u *account.AccountUpdate) bool {
			return u != nil &&
				u.StartingBalance != nil && u.StartingBalance.Equal(newStarting) &&
				u.Balance != nil && u.Balance.Equal(expectedBalance) &&
				u.Name == nil && u.SubType == nil
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	action := &UpdateAccount{
		ID:              accID,
		StartingBalance: &newStarting,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockAccount.AssertExpectations(t)
}

func TestUpdateAccount_Perform_NameOnly(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	newName := "Savings"
	existing := &account.Account{
		ID:              accID,
		Name:            "Old",
		Type:            account.AccountTypeCash,
		Balance:         decimal.NewFromInt(75),
		StartingBalance: decimal.NewFromInt(50),
	}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)
	mockAccount.EXPECT().
		Update(mock.Anything, accID, mock.MatchedBy(func(u *account.AccountUpdate) bool {
			return u != nil && u.Name != nil && *u.Name == newName &&
				u.SubType == nil && u.StartingBalance == nil && u.Balance == nil
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	action := &UpdateAccount{
		ID:   accID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockAccount.AssertExpectations(t)
}

func TestUpdateAccount_Perform_AccountNotFound(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	newName := "Updated"
	action := &UpdateAccount{
		ID:   accID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	mockAccount.AssertExpectations(t)
}

func TestUpdateAccount_Perform_NilExisting(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(nil, nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	newName := "Updated"
	action := &UpdateAccount{
		ID:   accID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	mockAccount.AssertExpectations(t)
}

func TestUpdateAccount_Perform_UpdateError(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	updateErr := errors.New("update failed")
	existing := &account.Account{
		ID:   accID,
		Name: "Checking",
		Type: account.AccountTypeCash,
	}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)
	mockAccount.EXPECT().
		Update(mock.Anything, accID, mock.Anything).
		Return(updateErr)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	newName := "Updated"
	action := &UpdateAccount{
		ID:   accID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, updateErr)
	mockAccount.AssertExpectations(t)
}
