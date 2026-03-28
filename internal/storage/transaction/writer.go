package transaction

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
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

func (w *Writer) Delete(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	row, err := bobgen.FindTransaction(ctx, w.tx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	txn := bobTransactionToTransaction(row)
	if _, err := bobgen.Transactions.Delete(bobgen.DeleteWhere.Transactions.ID.EQ(id)).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return txn, nil
}

// Update updates the mutable fields of an existing transaction. CategoryID and AccountID are not touched.
func (w *Writer) Update(ctx context.Context, id uuid.UUID, update *TransactionUpdate) error {
	setter := bobgen.TransactionSetter{
		Amount:          omit.From(update.Amount),
		TransactionName: omit.From(update.TransactionName),
	}
	if !update.TransactionDate.IsZero() {
		setter.TransactionDate = omit.From(update.TransactionDate)
	}
	_, err := bobgen.Transactions.Update(
		setter.UpdateMod(),
		um.Where(bobgen.Transactions.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, w.tx)
	return err
}

func (w *Writer) Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error) {
	setter := &bobgen.TransactionSetter{
		AccountID:       omit.From(create.AccountID),
		CategoryID:      omitnull.FromPtr(create.CategoryID),
		Amount:          omit.From(create.Amount),
		TransactionName: omit.From(create.TransactionName),
	}
	if create.ID != nil {
		setter.ID = omit.From(*create.ID)
	}
	if !create.TransactionDate.IsZero() {
		setter.TransactionDate = omit.From(create.TransactionDate)
	}
	row, err := bobgen.Transactions.Insert(setter).One(ctx, w.tx)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}
