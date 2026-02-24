package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
)

func newTestService(t *testing.T) (*TransactionService, *sqlconfig.MockITransactionTable) {
	t.Helper()
	mockTable := sqlconfig.NewMockITransactionTable(t)
	store := &storage.Storage{Transactions: mockTable}
	svc := NewTransactionService(store)
	return svc, mockTable
}

// -- CreateTransaction tests --

func TestCreateTransaction_Success(t *testing.T) {
	svc, mockTable := newTestService(t)

	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	amount := decimal.RequireFromString("42.50")
	txDate := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	expectedID := uuid.Must(uuid.NewV4())

	mockTable.EXPECT().Insert(mock.Anything, mock.MatchedBy(func(c *sqlconfig.TransactionCreate) bool {
		return c.AccountID == accountID &&
			c.CategoryID == categoryID &&
			c.Amount.Equal(amount) &&
			c.TransactionName == "Groceries" &&
			c.TransactionDate.Equal(txDate)
	})).Return(expectedID, nil)

	id, err := svc.CreateTransaction(context.Background(), Transaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: "Groceries",
		TransactionDate: txDate,
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
}

func TestCreateTransaction_StorageError(t *testing.T) {
	svc, mockTable := newTestService(t)

	mockTable.EXPECT().Insert(mock.Anything, mock.Anything).
		Return(uuid.Nil, errors.New("connection refused"))

	id, err := svc.CreateTransaction(context.Background(), Transaction{
		AccountID:       uuid.Must(uuid.NewV4()),
		CategoryID:      uuid.Must(uuid.NewV4()),
		Amount:          decimal.RequireFromString("10.00"),
		TransactionName: "Test",
	})

	assert.Error(t, err)
	assert.Equal(t, "connection refused", err.Error())
	assert.Equal(t, uuid.Nil, id)
}

// -- ListTransactions tests --

func makeStorageRows(n int, createdAt time.Time) []*sqlconfig.Transaction {
	rows := make([]*sqlconfig.Transaction, n)
	for i := range rows {
		rows[i] = &sqlconfig.Transaction{
			ID:              uuid.Must(uuid.NewV4()),
			AccountID:       uuid.Must(uuid.NewV4()),
			CategoryID:      uuid.Must(uuid.NewV4()),
			Amount:          decimal.RequireFromString("5.00"),
			TransactionName: "Item",
			TransactionDate: createdAt,
			CreatedAt:       createdAt,
		}
	}
	return rows
}

func TestListTransactions_NoResults(t *testing.T) {
	svc, mockTable := newTestService(t)

	mockTable.EXPECT().List(mock.Anything, mock.Anything).
		Return([]*sqlconfig.Transaction{}, nil)

	result, err := svc.ListTransactions(context.Background(), TransactionListQuery{Limit: 10})

	assert.NoError(t, err)
	assert.Empty(t, result.Transactions)
	assert.True(t, result.MaxCreationTime.IsZero())
	assert.Nil(t, result.NextCursor)
}

func TestListTransactions_SinglePage(t *testing.T) {
	svc, mockTable := newTestService(t)

	now := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	rows := makeStorageRows(2, now)

	mockTable.EXPECT().List(mock.Anything, mock.MatchedBy(func(f *sqlconfig.TransactionFilter) bool {
		return f.Limit == 5 && f.Offset == 0 && f.MaxCreationTime == nil
	})).Return(rows, nil)

	result, err := svc.ListTransactions(context.Background(), TransactionListQuery{Limit: 5})

	assert.NoError(t, err)
	assert.Len(t, result.Transactions, 2)
	assert.Nil(t, result.NextCursor)
	assert.Equal(t, now, result.MaxCreationTime, "derived from first row")

	tx := result.Transactions[0]
	assert.Equal(t, rows[0].ID, tx.ID)
	assert.Equal(t, rows[0].AccountID, tx.AccountID)
	assert.Equal(t, rows[0].CategoryID, tx.CategoryID)
	assert.True(t, rows[0].Amount.Equal(tx.Amount))
	assert.Equal(t, rows[0].TransactionName, tx.TransactionName)
	assert.Equal(t, rows[0].TransactionDate, tx.TransactionDate)
	assert.Equal(t, rows[0].CreatedAt, tx.CreatedAt)
}

func TestListTransactions_HasNextPage(t *testing.T) {
	svc, mockTable := newTestService(t)

	now := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	rows := makeStorageRows(3, now) // limit=2 → storage returns limit+1=3

	mockTable.EXPECT().List(mock.Anything, mock.Anything).Return(rows, nil)

	result, err := svc.ListTransactions(context.Background(), TransactionListQuery{Limit: 2})

	assert.NoError(t, err)
	assert.Len(t, result.Transactions, 2, "truncated to limit")
	assert.Equal(t, now, result.MaxCreationTime)

	assert.NotNil(t, result.NextCursor)
	assert.Equal(t, 2, result.NextCursor.Position)
	assert.Equal(t, 2, result.NextCursor.Limit)
	assert.Equal(t, now, result.NextCursor.MaxCreationTime)
}

func TestListTransactions_WithCursor(t *testing.T) {
	svc, mockTable := newTestService(t)

	cursorTime := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	rowTime := time.Date(2025, 6, 10, 8, 0, 0, 0, time.UTC) // older than cursor time
	rows := makeStorageRows(3, rowTime)                     // limit=2, returns 3 → has next page

	mockTable.EXPECT().List(mock.Anything, mock.MatchedBy(func(f *sqlconfig.TransactionFilter) bool {
		return f.Limit == 2 &&
			f.Offset == 20 &&
			f.MaxCreationTime != nil &&
			f.MaxCreationTime.Equal(cursorTime)
	})).Return(rows, nil)

	result, err := svc.ListTransactions(context.Background(), TransactionListQuery{
		Limit: 2,
		Cursor: &TransactionCursor{
			Position:        20,
			Limit:           2,
			MaxCreationTime: cursorTime,
		},
	})

	assert.NoError(t, err)
	assert.Len(t, result.Transactions, 2)
	assert.Equal(t, cursorTime, result.MaxCreationTime, "echoed from cursor, not overridden by row data")

	assert.NotNil(t, result.NextCursor)
	assert.Equal(t, 22, result.NextCursor.Position)
	assert.Equal(t, 2, result.NextCursor.Limit)
	assert.Equal(t, cursorTime, result.NextCursor.MaxCreationTime)
}

func TestListTransactions_StorageError(t *testing.T) {
	svc, mockTable := newTestService(t)

	mockTable.EXPECT().List(mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable"))

	result, err := svc.ListTransactions(context.Background(), TransactionListQuery{Limit: 10})

	assert.Error(t, err)
	assert.Equal(t, "database unavailable", err.Error())
	assert.Nil(t, result)
}
