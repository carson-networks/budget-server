package service

import (
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// AccountType represents an account type in the service layer.
type AccountType int8

const (
	AccountTypeCash AccountType = iota
	AccountTypeCreditCards
	AccountTypeInvestments
	AccountTypeLoans
	AccountTypeAssets
)

// Account represents an account in the service layer.
type Account struct {
	ID      uuid.UUID
	Name    string
	Type    AccountType
	SubType string
	Balance decimal.Decimal
}

// AccountCursor identifies a position in a paginated result set.
type AccountCursor struct {
	Position int
	Limit    int
}

func accountTypeToStorage(t AccountType) sqlconfig.AccountType {
	return sqlconfig.AccountType(t)
}

func accountTypeFromStorage(t sqlconfig.AccountType) AccountType {
	return AccountType(t)
}
