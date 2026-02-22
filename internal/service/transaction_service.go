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
