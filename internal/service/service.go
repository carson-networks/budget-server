package service

import (
	"github.com/carson-networks/budget-server/internal/storage"
)

// Service holds all business logic services.
type Service struct {
	Transaction *TransactionService
	Account     *AccountService
}

// NewService creates a new Service with the given storage.
func NewService(store *storage.Storage) *Service {
	return &Service{
		Transaction: NewTransactionService(store),
		Account:     NewAccountService(store),
	}
}
