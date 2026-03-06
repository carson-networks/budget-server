package storage

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
)

// IAccountWriter defines the account write operations used by actions.
type IAccountWriter interface {
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*account.Account, error)
	Create(ctx context.Context, name string, accountType account.AccountType, accountSubType string, startingBalance decimal.Decimal) error
	UpdateBalance(ctx context.Context, id uuid.UUID, balance decimal.Decimal) error
}

// ITransactionWriter defines the transaction write operations used by actions.
type ITransactionWriter interface {
	Insert(ctx context.Context, create *transaction.TransactionCreate) (uuid.UUID, error)
}

// txRunner is the minimal interface for transaction commit/rollback.
// bob.Tx satisfies this interface. Used to allow mocking in tests.
type txRunner interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Writer struct {
	tx          txRunner
	Account     IAccountWriter
	Transaction ITransactionWriter
}

func NewWriter(tx bob.Tx) Writer {
	return Writer{
		tx:          tx,
		Account:     account.NewWriter(tx),
		Transaction: transaction.NewWriter(tx),
	}
}

func NewWriterForTest(tx txRunner, account IAccountWriter, transaction ITransactionWriter) Writer {
	return Writer{
		tx:          tx,
		Account:     account,
		Transaction: transaction,
	}
}

func (w *Writer) Commit() error {
	return w.tx.Commit(context.Background())
}

func (w *Writer) Rollback() error {
	return w.tx.Rollback(context.Background())
}
