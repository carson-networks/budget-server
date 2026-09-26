package transaction

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

func bobTransactionToTransaction(row *bobgen.Transaction) *Transaction {
	var catID *uuid.UUID
	if v, ok := row.CategoryID.Get(); ok {
		catID = &v
	}
	var merchantName *string
	if v, ok := row.MerchantName.Get(); ok {
		merchantName = &v
	}
	return &Transaction{
		ID:              row.ID,
		AccountID:       row.AccountID,
		CategoryID:      catID,
		Amount:          row.Amount,
		TransactionName: row.TransactionName,
		MerchantName:    merchantName,
		TransactionDate: row.TransactionDate,
		CreatedAt:       row.CreatedAt,
	}
}

// Transaction represents a transaction record.
type Transaction struct {
	ID              uuid.UUID
	AccountID       uuid.UUID
	CategoryID      *uuid.UUID // nil when uncategorized
	Amount          decimal.Decimal
	TransactionName string
	MerchantName    *string // nil when unknown / not provided by Plaid
	TransactionDate time.Time
	CreatedAt       time.Time
}

// TransactionCreate is the input for creating a new transaction.
type TransactionCreate struct {
	ID              *uuid.UUID // if set, use this ID; otherwise let the DB generate one
	AccountID       uuid.UUID
	CategoryID      *uuid.UUID // nil inserts NULL
	Amount          decimal.Decimal
	TransactionName string
	MerchantName    *string   // nil inserts NULL
	TransactionDate time.Time // defaults to now if zero
}

// TransactionUpdate is the input for updating an existing transaction's mutable fields.
// AccountID and CategoryID are intentionally excluded — account never changes, category is user-managed.
type TransactionUpdate struct {
	Amount          decimal.Decimal
	TransactionName string
	MerchantName    *string // nil clears merchant_name
	TransactionDate time.Time
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

// TransactionListResult contains a page of transactions, an optional next cursor,
// and the total number of rows matching the list filter (for numbered pagination).
type TransactionListResult struct {
	Transactions []*Transaction
	NextCursor   *TransactionCursor
	TotalCount   int
}

// CategoryTotal is the sum of transaction amounts for one category within a calendar month.
type CategoryTotal struct {
	CategoryID   uuid.UUID
	CategoryName string
	Total        decimal.Decimal
}

// MonthTotals is one month in a range, with per-category totals.
type MonthTotals struct {
	Year       int
	Month      int
	Categories []CategoryTotal
}

// ITransactionTable defines the interface for transaction storage operations.
// This abstraction allows swapping the implementation (e.g. Bob) without changing callers.
type ITransactionTable interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error)
	List(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error)
}
