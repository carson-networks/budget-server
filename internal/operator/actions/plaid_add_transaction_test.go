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
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

func validPlaidAddTransaction() *PlaidAddTransaction {
	return &PlaidAddTransaction{
		TransactionID:      uuid.Must(uuid.NewV4()),
		AccountID:          uuid.Must(uuid.NewV4()),
		PlaidTransactionID: "plaid-txn-1",
		PlaidAccountID:     "plaid-acc-1",
		Amount:             decimal.NewFromFloat(-42.00),
		Name:               "Coffee Shop",
		Date:               time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestPlaidAddTransaction_Perform_Success(t *testing.T) {
	a := validPlaidAddTransaction()

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Insert(mock.Anything, &transaction.TransactionCreate{
			ID:              &a.TransactionID,
			AccountID:       a.AccountID,
			Amount:          a.Amount,
			TransactionName: a.Name,
			TransactionDate: a.Date,
		}).
		Return(a.TransactionID, nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		CreateTransactionLink(mock.Anything, &plaidstore.TransactionLink{
			PlaidTransactionID: a.PlaidTransactionID,
			TransactionID:      a.TransactionID,
			PlaidAccountID:     a.PlaidAccountID,
		}).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	require.NoError(t, a.Perform(context.Background(), wt))
	mockTxn.AssertExpectations(t)
	mockPlaid.AssertExpectations(t)
}

func TestPlaidAddTransaction_Perform_InsertError(t *testing.T) {
	a := validPlaidAddTransaction()
	insertErr := errors.New("insert failed")

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().Insert(mock.Anything, mock.Anything).Return(uuid.Nil, insertErr)

	mockPlaid := &storage.MockIPlaidWriter{}

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	assert.ErrorIs(t, a.Perform(context.Background(), wt), insertErr)
	mockPlaid.AssertNotCalled(t, "CreateTransactionLink")
}

func TestPlaidAddTransaction_Perform_CreateTransactionLinkError(t *testing.T) {
	a := validPlaidAddTransaction()
	linkErr := errors.New("link failed")

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().Insert(mock.Anything, mock.Anything).Return(a.TransactionID, nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().CreateTransactionLink(mock.Anything, mock.Anything).Return(linkErr)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	assert.ErrorIs(t, a.Perform(context.Background(), wt), linkErr)
}
