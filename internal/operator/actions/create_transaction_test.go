package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

func TestCreateTransaction_Perform_Success(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	txnID := uuid.Must(uuid.NewV4())
	existingBalance := decimal.NewFromInt(500)
	amount := decimal.NewFromInt(-50)
	newBalance := existingBalance.Add(amount)
	txnDate := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(&account.Account{
			ID:      accountID,
			Balance: existingBalance,
		}, nil)
	mockAccount.EXPECT().
		UpdateBalance(mock.Anything, accountID, newBalance).
		Return(nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Insert(mock.Anything, &transaction.TransactionCreate{
			AccountID:       accountID,
			CategoryID:      categoryID,
			Amount:          amount,
			TransactionName: "Groceries",
			TransactionDate: txnDate,
		}).
		Return(txnID, nil)

	writer := storage.NewWriterForTest(noopTx{}, mockAccount, mockTxn)
	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: "Groceries",
		TransactionDate: txnDate,
	}

	err := action.Perform(context.Background(), &writer)
	require.NoError(t, err)
	mockAccount.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
}

func TestCreateTransaction_Perform_AccountNotFound(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(nil, nil)

	writer := storage.NewWriterForTest(noopTx{}, mockAccount, &storage.MockITransactionWriter{})

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), &writer)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
	mockAccount.AssertExpectations(t)
}

func TestCreateTransaction_Perform_FindByIDForUpdateError(t *testing.T) {
	findErr := errors.New("db error")
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(nil, findErr)

	writer := storage.NewWriterForTest(noopTx{}, mockAccount, &storage.MockITransactionWriter{})

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), &writer)
	assert.ErrorIs(t, err, findErr)
	mockAccount.AssertExpectations(t)
}

func TestCreateTransaction_Perform_InsertError(t *testing.T) {
	insertErr := errors.New("insert failed")
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(&account.Account{ID: accountID, Balance: decimal.Zero}, nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Insert(mock.Anything, mock.Anything).
		Return(uuid.Nil, insertErr)

	writer := storage.NewWriterForTest(noopTx{}, mockAccount, mockTxn)

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), &writer)
	assert.ErrorIs(t, err, insertErr)
	mockAccount.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
}

func TestCreateTransaction_Perform_UpdateBalanceError(t *testing.T) {
	updateErr := errors.New("update balance failed")
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	txnID := uuid.Must(uuid.NewV4())
	existingBalance := decimal.NewFromInt(100)
	amount := decimal.NewFromInt(50)
	newBalance := existingBalance.Add(amount)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(&account.Account{ID: accountID, Balance: existingBalance}, nil)
	mockAccount.EXPECT().
		UpdateBalance(mock.Anything, accountID, newBalance).
		Return(updateErr)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Insert(mock.Anything, mock.Anything).
		Return(txnID, nil)

	writer := storage.NewWriterForTest(noopTx{}, mockAccount, mockTxn)

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), &writer)
	assert.ErrorIs(t, err, updateErr)
	mockAccount.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
}
