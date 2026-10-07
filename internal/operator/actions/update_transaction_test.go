package actions

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func existingTransactionForUpdate() *transaction.Transaction {
	categoryID := uuid.Must(uuid.NewV4())
	merchant := "Cafe"
	return &transaction.Transaction{
		ID: uuid.Must(uuid.NewV4()), AccountID: uuid.Must(uuid.NewV4()), CategoryID: &categoryID,
		Amount: decimal.NewFromInt(-10), TransactionName: "Lunch", MerchantName: &merchant,
		TransactionDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestUpdateTransaction_CategoryOnly(t *testing.T) {
	existing := existingTransactionForUpdate()
	categoryID := uuid.Must(uuid.NewV4())
	writer := storage.NewWriterForTest()
	tx := storage.NewMockITransactionWriter(t)
	cat := storage.NewMockICategoryWriter(t)
	writer.Transaction, writer.Category = tx, cat
	tx.EXPECT().FindByIDForUpdate(mock.Anything, existing.ID).Return(existing, nil).Once()
	cat.EXPECT().GetByID(mock.Anything, categoryID).Return(validCategoryForTransaction(categoryID), nil).Once()
	tx.EXPECT().Patch(mock.Anything, existing.ID, &transaction.TransactionPatch{CategoryID: &categoryID}).Return(nil).Once()
	require.NoError(t, (&UpdateTransaction{ID: existing.ID, CategoryID: &categoryID}).Perform(context.Background(), writer))
}

func TestUpdateTransaction_AmountBalanceDelta(t *testing.T) {
	for _, newAmount := range []string{"-25", "0", "10", "-10"} {
		t.Run(newAmount, func(t *testing.T) {
			existing := existingTransactionForUpdate()
			amount := decimal.RequireFromString(newAmount)
			writer := storage.NewWriterForTest()
			tx := storage.NewMockITransactionWriter(t)
			acc := storage.NewMockIAccountWriter(t)
			writer.Transaction, writer.Account = tx, acc
			tx.EXPECT().FindByIDForUpdate(mock.Anything, existing.ID).Return(existing, nil).Once()
			tx.EXPECT().Patch(mock.Anything, existing.ID, &transaction.TransactionPatch{Amount: &amount}).Return(nil).Once()
			if !amount.Equal(existing.Amount) {
				acc.EXPECT().AdjustBalance(mock.Anything, existing.AccountID, amount.Sub(existing.Amount)).Return(nil).Once()
			}
			require.NoError(t, (&UpdateTransaction{ID: existing.ID, Amount: &amount}).Perform(context.Background(), writer))
		})
	}
}

func TestUpdateTransaction_MultipleFields(t *testing.T) {
	existing := existingTransactionForUpdate()
	categoryID := uuid.Must(uuid.NewV4())
	amount := decimal.NewFromInt(0)
	name, merchant := "Breakfast", ""
	date := existing.TransactionDate.AddDate(0, 1, 0)
	writer := storage.NewWriterForTest()
	tx := storage.NewMockITransactionWriter(t)
	cat := storage.NewMockICategoryWriter(t)
	acc := storage.NewMockIAccountWriter(t)
	writer.Transaction, writer.Category, writer.Account = tx, cat, acc
	tx.EXPECT().FindByIDForUpdate(mock.Anything, existing.ID).Return(existing, nil).Once()
	cat.EXPECT().GetByID(mock.Anything, categoryID).Return(validCategoryForTransaction(categoryID), nil).Once()
	tx.EXPECT().Patch(mock.Anything, existing.ID, &transaction.TransactionPatch{
		CategoryID: &categoryID, Amount: &amount, TransactionName: &name, MerchantName: &merchant, TransactionDate: &date,
	}).Return(nil).Once()
	acc.EXPECT().AdjustBalance(mock.Anything, existing.AccountID, decimal.NewFromInt(10)).Return(nil).Once()
	require.NoError(t, (&UpdateTransaction{
		ID: existing.ID, CategoryID: &categoryID, Amount: &amount,
		TransactionName: &name, MerchantName: &merchant, TransactionDate: &date,
	}).Perform(context.Background(), writer))
}

func TestUpdateTransaction_MissingOrFailedLookup(t *testing.T) {
	lookupErr := errors.New("lookup failed")
	for _, test := range []struct {
		name         string
		lookup, want error
	}{
		{"no rows", sql.ErrNoRows, ErrTransactionNotFound},
		{"nil row", nil, ErrTransactionNotFound},
		{"database error", lookupErr, lookupErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := uuid.Must(uuid.NewV4())
			writer := storage.NewWriterForTest()
			tx := storage.NewMockITransactionWriter(t)
			writer.Transaction = tx
			tx.EXPECT().FindByIDForUpdate(mock.Anything, id).Return(nil, test.lookup).Once()
			assert.ErrorIs(t, (&UpdateTransaction{ID: id}).Perform(context.Background(), writer), test.want)
		})
	}
}

func TestUpdateTransaction_InvalidCategoryDoesNotWrite(t *testing.T) {
	lookupErr := errors.New("category lookup failed")
	for _, test := range []struct {
		name                      string
		missing, disabled, parent bool
		lookup, want              error
	}{
		{name: "missing", missing: true, lookup: sql.ErrNoRows, want: ErrCategoryNotFoundForTransaction},
		{name: "nil category", missing: true, want: ErrCategoryNotFoundForTransaction},
		{name: "disabled", disabled: true, want: ErrCategoryDisabled},
		{name: "parent", parent: true, want: ErrCategoryIsParent},
		{name: "database error", missing: true, lookup: lookupErr, want: lookupErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			existing := existingTransactionForUpdate()
			categoryID := uuid.Must(uuid.NewV4())
			writer := storage.NewWriterForTest()
			tx := storage.NewMockITransactionWriter(t)
			cat := storage.NewMockICategoryWriter(t)
			writer.Transaction, writer.Category = tx, cat
			tx.EXPECT().FindByIDForUpdate(mock.Anything, existing.ID).Return(existing, nil).Once()
			value := validCategoryForTransaction(categoryID)
			value.IsDisabled, value.IsParent = test.disabled, test.parent
			if test.missing {
				value = nil
			}
			cat.EXPECT().GetByID(mock.Anything, categoryID).Return(value, test.lookup).Once()
			assert.ErrorIs(t, (&UpdateTransaction{ID: existing.ID, CategoryID: &categoryID}).Perform(context.Background(), writer), test.want)
		})
	}
}

func TestUpdateTransaction_WriteErrors(t *testing.T) {
	writeErr := errors.New("write failed")
	for _, test := range []struct {
		name                       string
		patchErr, balanceErr, want error
	}{
		{"patch", writeErr, nil, writeErr},
		{"balance", nil, writeErr, writeErr},
		{"missing account", nil, sql.ErrNoRows, ErrAccountNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			existing := existingTransactionForUpdate()
			amount := decimal.NewFromInt(0)
			writer := storage.NewWriterForTest()
			tx := storage.NewMockITransactionWriter(t)
			acc := storage.NewMockIAccountWriter(t)
			writer.Transaction, writer.Account = tx, acc
			tx.EXPECT().FindByIDForUpdate(mock.Anything, existing.ID).Return(existing, nil).Once()
			tx.EXPECT().Patch(mock.Anything, existing.ID, mock.Anything).Return(test.patchErr).Once()
			if test.patchErr == nil {
				acc.EXPECT().AdjustBalance(mock.Anything, existing.AccountID, decimal.NewFromInt(10)).Return(test.balanceErr).Once()
			}
			assert.ErrorIs(t, (&UpdateTransaction{ID: existing.ID, Amount: &amount}).Perform(context.Background(), writer), test.want)
		})
	}
}
