package storage

import (
	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/carson-networks/budget-server/internal/storage/budget"
	"github.com/carson-networks/budget-server/internal/storage/category"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/stephenafamo/bob"
)

type Reader struct {
	Accounts     *account.Reader
	Transactions *transaction.Reader
	Categories   *category.Reader
	Budgets      *budget.Reader
	Plaid        *plaidstore.Reader
	Sync         *syncstore.Reader
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{
		Accounts:     account.NewReader(exec),
		Transactions: transaction.NewReader(exec),
		Categories:   category.NewReader(exec),
		Budgets:      budget.NewReader(exec),
		Plaid:        plaidstore.NewReader(exec),
		Sync:         syncstore.NewReader(exec),
	}
}
