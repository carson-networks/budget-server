package service

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// Transaction represents a transaction in the service layer.
type Transaction struct {
	ID              uuid.UUID
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Amount          decimal.Decimal
	TransactionName string
	TransactionDate time.Time
	CreatedAt       time.Time
}

// TransactionCursor identifies a position in a paginated result set
// and carries the limit and maxCreationTime so subsequent pages are consistent.
type TransactionCursor struct {
	Position        int
	Limit           int
	MaxCreationTime time.Time
}
