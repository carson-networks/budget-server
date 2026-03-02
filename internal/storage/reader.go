package storage

import (
	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/stephenafamo/bob"
)

type Reader struct {
	Accounts     *account.Reader
	Transactions *transaction.Reader
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{
		Accounts:     account.NewReader(exec),
		Transactions: transaction.NewReader(exec),
	}
}
