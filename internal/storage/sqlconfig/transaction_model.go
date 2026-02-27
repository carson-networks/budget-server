package sqlconfig

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

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

// ITransactionTable defines the interface for transaction storage operations.
// This abstraction allows swapping the implementation (e.g. Bob) without changing callers.
//
//go:generate mockery --name ITransactionTable --output mock_ITransactionTable.go
type ITransactionTable interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error)
	List(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error)
}
