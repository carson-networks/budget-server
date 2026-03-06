package transaction

import (
	"context"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
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

func (w *Writer) Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error) {
	setter := &bobgen.TransactionSetter{
		AccountID:       omit.From(create.AccountID),
		CategoryID:      omit.From(create.CategoryID),
		Amount:          omit.From(create.Amount),
		TransactionName: omit.From(create.TransactionName),
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
