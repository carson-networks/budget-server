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
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

func validPlaidRemoveTransaction() *PlaidRemoveTransaction {
	return &PlaidRemoveTransaction{
		TransactionID:      uuid.Must(uuid.NewV4()),
		AccountID:          uuid.Must(uuid.NewV4()),
		PlaidTransactionID: "plaid-txn-to-remove",
	}
}

func TestPlaidRemoveTransaction_Perform_Success(t *testing.T) {
	a := validPlaidRemoveTransaction()
	amount := decimal.NewFromFloat(-75.00)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Delete(mock.Anything, a.TransactionID).
		Return(&transaction.Transaction{ID: a.TransactionID, Amount: amount}, nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		DeleteTransactionLink(mock.Anything, a.PlaidTransactionID).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	require.NoError(t, a.Perform(context.Background(), wt))
	mockTxn.AssertExpectations(t)
	mockPlaid.AssertExpectations(t)
}

func TestPlaidRemoveTransaction_Perform_AlreadyDeleted(t *testing.T) {
	// Delete returns nil (already gone) — no error expected
	a := validPlaidRemoveTransaction()

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().Delete(mock.Anything, a.TransactionID).Return(nil, nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().DeleteTransactionLink(mock.Anything, a.PlaidTransactionID).Return(nil)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	require.NoError(t, a.Perform(context.Background(), wt))
}

func TestPlaidRemoveTransaction_Perform_DeleteTransactionError(t *testing.T) {
	a := validPlaidRemoveTransaction()
	deleteErr := errors.New("delete failed")

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().Delete(mock.Anything, a.TransactionID).Return(nil, deleteErr)

	mockPlaid := &storage.MockIPlaidWriter{}

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	assert.ErrorIs(t, a.Perform(context.Background(), wt), deleteErr)
	mockPlaid.AssertNotCalled(t, "DeleteTransactionLink")
}

func TestPlaidRemoveTransaction_Perform_DeleteLinkError(t *testing.T) {
	a := validPlaidRemoveTransaction()
	linkErr := errors.New("link delete failed")

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		Delete(mock.Anything, a.TransactionID).
		Return(&transaction.Transaction{ID: a.TransactionID, Amount: decimal.NewFromFloat(-20.00)}, nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().DeleteTransactionLink(mock.Anything, a.PlaidTransactionID).Return(linkErr)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	assert.ErrorIs(t, a.Perform(context.Background(), wt), linkErr)
}
