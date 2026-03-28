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
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

func validPlaidModifyTransaction() *PlaidModifyTransaction {
	return &PlaidModifyTransaction{
		TransactionID: uuid.Must(uuid.NewV4()),
		AccountID:     uuid.Must(uuid.NewV4()),
		Amount:        decimal.NewFromFloat(-55.00),
		Name:          "Updated Merchant",
		Date:          time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC),
	}
}

func TestPlaidModifyTransaction_Perform_Success(t *testing.T) {
	a := validPlaidModifyTransaction()
	oldAmount := decimal.NewFromFloat(-40.00)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		FindByID(mock.Anything, a.TransactionID).
		Return(&transaction.Transaction{ID: a.TransactionID, Amount: oldAmount}, nil)
	mockTxn.EXPECT().
		Update(mock.Anything, a.TransactionID, &transaction.TransactionUpdate{
			Amount:          a.Amount,
			TransactionName: a.Name,
			TransactionDate: a.Date,
		}).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn

	require.NoError(t, a.Perform(context.Background(), wt))
	mockTxn.AssertExpectations(t)
}

func TestPlaidModifyTransaction_Perform_TransactionNotFound(t *testing.T) {
	a := validPlaidModifyTransaction()

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().FindByID(mock.Anything, a.TransactionID).Return(nil, nil)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn

	assert.Error(t, a.Perform(context.Background(), wt))
	mockTxn.AssertNotCalled(t, "Update")
}

func TestPlaidModifyTransaction_Perform_FindByIDError(t *testing.T) {
	a := validPlaidModifyTransaction()
	findErr := errors.New("db error")

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().FindByID(mock.Anything, a.TransactionID).Return(nil, findErr)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn

	assert.ErrorIs(t, a.Perform(context.Background(), wt), findErr)
}

func TestPlaidModifyTransaction_Perform_UpdateError(t *testing.T) {
	a := validPlaidModifyTransaction()
	updateErr := errors.New("update failed")

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		FindByID(mock.Anything, a.TransactionID).
		Return(&transaction.Transaction{ID: a.TransactionID, Amount: decimal.NewFromFloat(-10.00)}, nil)
	mockTxn.EXPECT().Update(mock.Anything, a.TransactionID, mock.Anything).Return(updateErr)

	wt := storage.NewWriterForTest()
	wt.Transaction = mockTxn

	assert.ErrorIs(t, a.Perform(context.Background(), wt), updateErr)
}
