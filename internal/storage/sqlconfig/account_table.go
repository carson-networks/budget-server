package sqlconfig

import (
	"context"
	"database/sql"

	"github.com/aarondl/opt/omit"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
)

// Account represents an account record.
type Account struct {
	ID      uuid.UUID
	Name    string
	Type    AccountType
	SubType string
	Balance decimal.Decimal
}

// AccountCreate is the input for creating a new account.
type AccountCreate struct {
	Name    string
	Type    AccountType
	SubType string
	Balance decimal.Decimal
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

// AccountsTable provides access to the accounts table.
type AccountsTable struct {
	exec bob.Executor
}

// Ensure AccountsTable implements IAccountTable at compile time.
var _ IAccountTable = (*AccountsTable)(nil)

// NewAccountsTable creates an AccountsTable for the given database.
func NewAccountsTable(db *sql.DB) AccountsTable {
	return AccountsTable{exec: bob.NewDB(db)}
}

// FindByID retrieves an account by primary key.
func (t *AccountsTable) FindByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	row, err := bobgen.FindAccount(ctx, t.exec, id)
	if err != nil {
		return nil, err
	}
	return bobAccountToAccount(row), nil
}

// Insert creates a new account and returns its generated ID.
func (t *AccountsTable) Insert(ctx context.Context, create *AccountCreate) (uuid.UUID, error) {
	setter := &bobgen.AccountSetter{
		Name:    omit.From(create.Name),
		Type:    omit.From(int16(create.Type)),
		SubType: omit.From(create.SubType),
		Balance: omit.From(create.Balance),
	}
	row, err := bobgen.Accounts.Insert(setter).One(ctx, t.exec)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, err
}

// List returns accounts matching the filter. Nil filter returns all.
func (t *AccountsTable) List(ctx context.Context, filter *AccountFilter) ([]*Account, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]
	if filter != nil {
		if filter.Limit > 0 {
			queryMods = append(queryMods, sm.Limit(filter.Limit+1))
		}
		if filter.Offset > 0 {
			queryMods = append(queryMods, sm.Offset(filter.Offset))
		}
	}
	queryMods = append(queryMods,
		sm.OrderBy(bobgen.Accounts.Columns.Name).Asc(),
		sm.OrderBy(bobgen.Accounts.Columns.ID).Asc(),
	)
	rows, err := bobgen.Accounts.Query(queryMods...).All(ctx, t.exec)
	if err != nil {
		return nil, err
	}
	result := make([]*Account, len(rows))
	for i, row := range rows {
		result[i] = bobAccountToAccount(row)
	}
	return result, nil
}

// UpdateBalance updates the balance for a given account.
func (t *AccountsTable) UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal) error {
	setter := bobgen.AccountSetter{
		Balance: omit.From(balance),
	}
	_, err := bobgen.Accounts.Update(setter.UpdateMod(), um.Where(bobgen.Accounts.Columns.ID.EQ(psql.Arg(id)))).Exec(ctx, t.exec)
	return err
}

func bobAccountToAccount(row *bobgen.Account) *Account {
	return &Account{
		ID:      row.ID,
		Name:    row.Name,
		Type:    AccountType(row.Type),
		SubType: row.SubType,
		Balance: row.Balance,
	}
}
