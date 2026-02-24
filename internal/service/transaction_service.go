package service

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
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

// TransactionListQuery is the input for listing transactions with cursor pagination.
type TransactionListQuery struct {
	Limit  int
	Cursor *TransactionCursor
}

// TransactionListResult is the output of a paginated transaction list.
type TransactionListResult struct {
	Transactions    []Transaction
	MaxCreationTime time.Time
	NextCursor      *TransactionCursor
}

// TransactionService handles transaction business logic.
type TransactionService struct {
	storage *storage.Storage
}

// NewTransactionService creates a new TransactionService.
func NewTransactionService(store *storage.Storage) *TransactionService {
	return &TransactionService{storage: store}
}

// CreateTransaction creates a new transaction and returns its ID.
func (s *TransactionService) CreateTransaction(ctx context.Context, transaction Transaction) (uuid.UUID, error) {
	storageCreate := &sqlconfig.TransactionCreate{
		AccountID:       transaction.AccountID,
		CategoryID:      transaction.CategoryID,
		Amount:          transaction.Amount,
		TransactionName: transaction.TransactionName,
		TransactionDate: transaction.TransactionDate,
	}

	return s.storage.Transactions.Insert(ctx, storageCreate)
}

// ListTransactions returns a page of transactions using cursor-based pagination.
func (s *TransactionService) ListTransactions(ctx context.Context, query TransactionListQuery) (*TransactionListResult, error) {
	offset := 0
	var maxCreationTime *time.Time
	if query.Cursor != nil {
		offset = query.Cursor.Position
		maxCreationTime = &query.Cursor.MaxCreationTime
	}

	filter := &sqlconfig.TransactionFilter{
		Limit:           query.Limit,
		Offset:          offset,
		MaxCreationTime: maxCreationTime,
	}

	rows, err := s.storage.Transactions.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := &TransactionListResult{}

	if maxCreationTime != nil {
		result.MaxCreationTime = *maxCreationTime
	}

	if len(rows) == 0 {
		return result, nil
	}

	// If no maxCreationTime was provided, use the first row (most recent due to DESC order).
	if result.MaxCreationTime.IsZero() {
		result.MaxCreationTime = rows[0].CreatedAt
	}

	if query.Limit > 0 && len(rows) > query.Limit {
		rows = rows[:query.Limit]
		result.NextCursor = &TransactionCursor{
			Position:        offset + query.Limit,
			Limit:           query.Limit,
			MaxCreationTime: result.MaxCreationTime,
		}
	}

	result.Transactions = make([]Transaction, len(rows))
	for i, row := range rows {
		result.Transactions[i] = Transaction{
			ID:              row.ID,
			AccountID:       row.AccountID,
			CategoryID:      row.CategoryID,
			Amount:          row.Amount,
			TransactionName: row.TransactionName,
			TransactionDate: row.TransactionDate,
			CreatedAt:       row.CreatedAt,
		}
	}

	return result, nil
}
