package account

import (
	"context"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

type Writer struct {
	tx bob.Tx
	Reader
}

func NewWriter(tx bob.Tx) *Writer {
	return &Writer{
		tx: tx,
		Reader: Reader{
			exec: tx,
		},
	}
}

func (w *Writer) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Account, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]
	queryMods = append(queryMods,
		bobgen.SelectWhere.Accounts.ID.EQ(id),
		sm.ForUpdate(),
	)
	bobgen.Accounts.Query()

	row, err := bobgen.FindAccount(ctx, w.tx, id)
	if err != nil {
		return nil, err
	}
	return bobAccountToAccount(row), nil
}

func (w *Writer) Create(ctx context.Context, name string, accountType AccountType, accountSubType string, startingBalance decimal.Decimal) error {
	setter := &bobgen.AccountSetter{
		Name:            omit.From(name),
		Type:            omit.From(int16(accountType)),
		SubType:         omit.From(accountSubType),
		Balance:         omit.From(startingBalance),
		StartingBalance: omit.From(startingBalance),
	}
	_, err := bobgen.Accounts.Insert(setter).One(ctx, w.tx)
	if err != nil {
		return err
	}
	return err
}

func (w *Writer) UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal) error {
	setter := bobgen.AccountSetter{
		Balance: omit.From(balance),
	}
	_, err := bobgen.Accounts.Update(setter.UpdateMod(), um.Where(bobgen.Accounts.Columns.ID.EQ(psql.Arg(id)))).Exec(ctx, w.tx)
	return err
}
