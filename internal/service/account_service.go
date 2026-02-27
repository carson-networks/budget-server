package service

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
)

const defaultAccountLimit = 20

// AccountService handles account business logic.
type AccountService struct {
	storage *storage.Storage
}

// NewAccountService creates a new AccountService.
func NewAccountService(store *storage.Storage) *AccountService {
	return &AccountService{storage: store}
}

// CreateAccount creates a new account and returns its ID.
func (s *AccountService) CreateAccount(ctx context.Context, account Account) (uuid.UUID, error) {
	storageCreate := &sqlconfig.AccountCreate{
		Name:    account.Name,
		Type:    accountTypeToStorage(account.Type),
		SubType: account.SubType,
		Balance: account.Balance,
	}

	return s.storage.Accounts.Insert(ctx, storageCreate)
}

// GetAccount retrieves an account by ID.
func (s *AccountService) GetAccount(ctx context.Context, id uuid.UUID) (*Account, error) {
	row, err := s.storage.Accounts.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:      row.ID,
		Name:    row.Name,
		Type:    accountTypeFromStorage(row.Type),
		SubType: row.SubType,
		Balance: row.Balance,
	}, nil
}

// ListAccounts returns a page of accounts using cursor pagination.
func (s *AccountService) ListAccounts(ctx context.Context, cursor *AccountCursor) ([]Account, *AccountCursor, error) {
	limit := defaultAccountLimit
	offset := 0
	if cursor != nil {
		limit = cursor.Limit
		offset = cursor.Position
	}

	filter := &sqlconfig.AccountFilter{
		Limit:  limit,
		Offset: offset,
	}

	var nextCursor *AccountCursor
	accounts, err := s.storage.Accounts.List(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	if len(accounts) == 0 {
		return nil, nil, nil
	}

	if len(accounts) > limit {
		accounts = accounts[:limit]
		nextCursor = &AccountCursor{
			Position: offset + limit,
			Limit:    limit,
		}
	}

	convertedAccounts := make([]Account, len(accounts))
	for i, account := range accounts {
		convertedAccounts[i] = Account{
			ID:      account.ID,
			Name:    account.Name,
			Type:    accountTypeFromStorage(account.Type),
			SubType: account.SubType,
			Balance: account.Balance,
		}
	}

	return convertedAccounts, nextCursor, nil
}

// UpdateBalance updates an account's balance.
func (s *AccountService) UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal) error {
	return s.storage.Accounts.UpdateBalance(ctx, id, balance)
}
