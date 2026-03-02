package transaction

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

func bobTransactionToTransaction(row *bobgen.Transaction) *Transaction {
	return &Transaction{
		ID:              row.ID,
		AccountID:       row.AccountID,
		CategoryID:      row.CategoryID,
		Amount:          row.Amount,
		TransactionName: row.TransactionName,
		TransactionDate: row.TransactionDate,
		CreatedAt:       row.CreatedAt,
	}
}

// Transaction represents a transaction record.
type Transaction struct {
	ID              uuid.UUID
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Amount          decimal.Decimal
	TransactionName string
	TransactionDate time.Time
	CreatedAt       time.Time
}

// TransactionCreate is the input for creating a new transaction.
type TransactionCreate struct {
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Amount          decimal.Decimal
	TransactionName string
	TransactionDate time.Time // defaults to now if zero
}

// TransactionFilter specifies filters for listing transactions.
type TransactionFilter struct {
	AccountID       *uuid.UUID
	CategoryID      *uuid.UUID
	Limit           int
	Offset          int
	MaxCreationTime *time.Time
}

// TransactionCursor identifies a position in a paginated result set
// and carries the limit and maxCreationTime so subsequent pages are consistent.
type TransactionCursor struct {
	Position        int
	Limit           int
	MaxCreationTime time.Time
}

// TransactionListResult contains a page of transactions and an optional next cursor.
type TransactionListResult struct {
	Transactions []*Transaction
	NextCursor   *TransactionCursor
}

// ITransactionTable defines the interface for transaction storage operations.
// This abstraction allows swapping the implementation (e.g. Bob) without changing callers.
//
//go:generate mockery --name ITransactionTable --output mock_ITransactionTable.go
type ITransactionTable interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error)
	List(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error)
}
