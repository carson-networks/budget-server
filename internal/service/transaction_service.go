package service

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
	"github.com/gofrs/uuid/v5"
)

const defaultLimit = 20

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
func (s *TransactionService) ListTransactions(ctx context.Context, cursor *TransactionCursor) ([]Transaction, *TransactionCursor, error) {
	limit := defaultLimit
	offset := 0
	var maxCreationTime *time.Time
	if cursor != nil {
		limit = cursor.Limit
		offset = cursor.Position
		maxCreationTime = &cursor.MaxCreationTime
	}

	filter := &sqlconfig.TransactionFilter{
		Limit:           limit,
		Offset:          offset,
		MaxCreationTime: maxCreationTime,
	}

	rows, err := s.storage.Transactions.List(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	if len(rows) == 0 {
		return nil, nil, nil
	}

	var nextCursor *TransactionCursor
	if len(rows) > limit {
		rows = rows[:limit]

		cursorMaxCreationTime := rows[0].CreatedAt
		if maxCreationTime != nil {
			cursorMaxCreationTime = *maxCreationTime
		}

		nextCursor = &TransactionCursor{
			Position:        offset + limit,
			Limit:           limit,
			MaxCreationTime: cursorMaxCreationTime,
		}
	}

	convertedTransactions := make([]Transaction, len(rows))
	for i, row := range rows {
		convertedTransactions[i] = Transaction{
			ID:              row.ID,
			AccountID:       row.AccountID,
			CategoryID:      row.CategoryID,
			Amount:          row.Amount,
			TransactionName: row.TransactionName,
			TransactionDate: row.TransactionDate,
			CreatedAt:       row.CreatedAt,
		}
	}

	return convertedTransactions, nextCursor, nil
}
