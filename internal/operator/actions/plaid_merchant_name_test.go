package actions

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

func TestPlaidAddTransaction_Perform_NilMerchantName(t *testing.T) {
	a := validPlaidAddTransaction()
	a.MerchantName = nil

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Insert(mock.Anything, &transaction.TransactionCreate{
			ID:              &a.TransactionID,
			AccountID:       a.AccountID,
			Amount:          a.Amount,
			TransactionName: a.Name,
			MerchantName:    nil,
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
}

func TestPlaidModifyTransaction_Perform_ClearsMerchantName(t *testing.T) {
	a := &PlaidModifyTransaction{
		TransactionID: uuid.Must(uuid.NewV4()),
		AccountID:     uuid.Must(uuid.NewV4()),
		Amount:        decimal.NewFromFloat(-10.00),
		Name:          "ACH PAYMENT",
		MerchantName:  nil,
		Date:          time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
	}

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		FindByID(mock.Anything, a.TransactionID).
		Return(&transaction.Transaction{ID: a.TransactionID}, nil)
	mockTxn.EXPECT().
		Update(mock.Anything, a.TransactionID, &transaction.TransactionUpdate{
			Amount:          a.Amount,
			TransactionName: a.Name,
			MerchantName:    nil,
			TransactionDate: a.Date,
		}).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn

	require.NoError(t, a.Perform(context.Background(), wt))
	mockTxn.AssertExpectations(t)
}
