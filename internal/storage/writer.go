package storage

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/stephenafamo/bob"
)

type Writer struct {
	tx          bob.Tx
	Account     *account.Writer
	Transaction *transaction.Writer
}

func NewWriter(tx bob.Tx) Writer {
	return Writer{
		tx:          tx,
		Account:     account.NewWriter(tx),
		Transaction: transaction.NewWriter(tx),
	}
}

func (w *Writer) Commit() error {
	return w.tx.Commit(context.Background())
}

func (w *Writer) Rollback() error {
	return w.tx.Rollback(context.Background())
}
