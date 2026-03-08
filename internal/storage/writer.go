package storage

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/carson-networks/budget-server/internal/storage/category"
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

// ICategoryWriter defines the category write operations used by actions.
type ICategoryWriter interface {
	Create(ctx context.Context, create *category.CategoryCreate) (uuid.UUID, error)
	Update(ctx context.Context, id uuid.UUID, update *category.CategoryUpdate) error
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
	Category    ICategoryWriter
}

func NewWriter(tx bob.Tx) Writer {
	return Writer{
		tx:          tx,
		Account:     account.NewWriter(tx),
		Transaction: transaction.NewWriter(tx),
		Category:    category.NewWriter(tx),
	}
}

func NewWriterForTest() *Writer {
	mockAccount := &MockIAccountWriter{}
	mockTxn := &MockITransactionWriter{}
	mockCat := &MockICategoryWriter{}
	return &Writer{
		Account:     mockAccount,
		Transaction: mockTxn,
		Category:    mockCat,
	}
}

// NewWriterForTestWithTx returns a test Writer with the given tx (e.g. for operator tests that assert Commit/Rollback).
func NewWriterForTestWithTx(tx txRunner) *Writer {
	w := NewWriterForTest()
	w.tx = tx
	return w
}

func (w *Writer) Commit() error {
	return w.tx.Commit(context.Background())
}

func (w *Writer) Rollback() error {
	return w.tx.Rollback(context.Background())
}
