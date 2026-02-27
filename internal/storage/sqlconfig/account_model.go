package sqlconfig

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// Account represents an account record.
type Account struct {
	ID              uuid.UUID
	Name            string
	Type            AccountType
	SubType         string
	Balance         decimal.Decimal
	StartingBalance decimal.Decimal
	CreatedAt       time.Time
}

// AccountCreate is the input for creating a new account.
type AccountCreate struct {
	Name            string
	Type            AccountType
	SubType         string
	Balance         decimal.Decimal
	StartingBalance decimal.Decimal
}

// AccountFilter specifies filters for listing accounts.
type AccountFilter struct {
	Limit  int
	Offset int
}

// IAccountTable defines the interface for account storage operations.
// This abstraction allows swapping the implementation (e.g. Bob) without changing callers.
//
//go:generate mockery --name IAccountTable --output mock_IAccountTable.go
type IAccountTable interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Account, error)
	Insert(ctx context.Context, create *AccountCreate) (uuid.UUID, error)
	List(ctx context.Context, filter *AccountFilter) ([]*Account, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal) error
}

type AccountType int8

const (
	AccountTypeCash AccountType = iota
	AccountTypeCreditCards
	AccountTypeInvestments
	AccountTypeLoans
	AccountTypeAssets
)

func bobAccountToAccount(row *bobgen.Account) *Account {
	return &Account{
		ID:              row.ID,
		Name:            row.Name,
		Type:            AccountType(row.Type),
		SubType:         row.SubType,
		Balance:         row.Balance,
		StartingBalance: row.StartingBalance,
		CreatedAt:       row.CreatedAt,
	}
}
