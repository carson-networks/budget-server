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

var _ IAccountTable = (*AccountsTable)(nil)

type AccountsTable struct {
	exec bob.Executor
}

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
		Name:            omit.From(create.Name),
		Type:            omit.From(int16(create.Type)),
		SubType:         omit.From(create.SubType),
		Balance:         omit.From(create.Balance),
		StartingBalance: omit.From(create.StartingBalance),
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
