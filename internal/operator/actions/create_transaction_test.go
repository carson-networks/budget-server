package actions

import (
	"context"
	"database/sql"
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
	"github.com/carson-networks/budget-server/internal/storage/category"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

func validCategoryForTransaction(id uuid.UUID) *category.Category {
	return &category.Category{
		ID: id, Name: "Food", IsGroup: false, IsDisabled: false,
		ShouldBeBudgeted: true, CategoryType: category.CatergoryType_Expense,
	}
}

func TestCreateTransaction_Perform_Success(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	txnID := uuid.Must(uuid.NewV4())
	existingBalance := decimal.NewFromInt(500)
	amount := decimal.NewFromInt(-50)
	newBalance := existingBalance.Add(amount)
	txnDate := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validCategoryForTransaction(categoryID), nil)

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

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Account = mockAccount
	wt.Transaction = mockTxn
	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: "Groceries",
		TransactionDate: txnDate,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
}

func TestCreateTransaction_Perform_CategoryNotFound(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryNotFoundForTransaction)
	mockCat.AssertExpectations(t)
}

func TestCreateTransaction_Perform_CategoryDisabled(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	cat := validCategoryForTransaction(categoryID)
	cat.IsDisabled = true

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(cat, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryDisabled)
	mockCat.AssertExpectations(t)
}

func TestCreateTransaction_Perform_CategoryIsGroup(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	cat := validCategoryForTransaction(categoryID)
	cat.IsGroup = true

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(cat, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryIsGroup)
	mockCat.AssertExpectations(t)
}

func TestCreateTransaction_Perform_AccountNotFound(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validCategoryForTransaction(categoryID), nil)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(nil, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Account = mockAccount

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
	mockCat.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestCreateTransaction_Perform_FindByIDForUpdateError(t *testing.T) {
	findErr := errors.New("db error")
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validCategoryForTransaction(categoryID), nil)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(nil, findErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Account = mockAccount

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, findErr)
	mockCat.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestCreateTransaction_Perform_InsertError(t *testing.T) {
	insertErr := errors.New("insert failed")
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validCategoryForTransaction(categoryID), nil)

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accountID).
		Return(&account.Account{ID: accountID, Balance: decimal.Zero}, nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Insert(mock.Anything, mock.Anything).
		Return(uuid.Nil, insertErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Account = mockAccount
	wt.Transaction = mockTxn

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          decimal.NewFromInt(100),
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, insertErr)
	mockCat.AssertExpectations(t)
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

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validCategoryForTransaction(categoryID), nil)

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

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Account = mockAccount
	wt.Transaction = mockTxn

	action := &CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: "Test",
		TransactionDate: time.Now(),
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, updateErr)
	mockCat.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
}
