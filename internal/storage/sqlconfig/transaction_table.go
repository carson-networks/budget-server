package sqlconfig

import (
	"context"
	"database/sql"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
)

// Transaction represents a transaction record.
type Transaction struct {
	ID              uuid.UUID
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Amount          decimal.Decimal
	TransactionName string
	TransactionDate time.Time
}

// TransactionCreate is the input for creating a new transaction.
type TransactionCreate struct {
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Amount          decimal.Decimal
	TransactionName string
	TransactionDate time.Time // defaults to now if zero
}

// TransactionFilter specifies filters for listing transactions.
type TransactionFilter struct {
	AccountID  *uuid.UUID
	CategoryID *uuid.UUID
}

// ITransactionTable defines the interface for transaction storage operations.
// This abstraction allows swapping the implementation (e.g. Bob) without changing callers.
//
//go:generate mockery --name ITransactionTable --output mock_ITransactionTable.go
type ITransactionTable interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error)
	List(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error)
}

// TransactionsTable provides access to the transactions table.
type TransactionsTable struct {
	exec bob.Executor
}

// Ensure TransactionsTable implements ITransactionTable at compile time.
var _ ITransactionTable = (*TransactionsTable)(nil)

// NewTransactionsTable creates a TransactionsTable for the given database.
func NewTransactionsTable(db *sql.DB) TransactionsTable {
	return TransactionsTable{exec: bob.NewDB(db)}
}

// FindByID retrieves a transaction by primary key.
func (t *TransactionsTable) FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	row, err := bobgen.FindTransaction(ctx, t.exec, id)
	if err != nil {
		return nil, err
	}
	return bobTransactionToTransaction(row), nil
}

// Insert creates a new transaction and returns its generated ID.
func (t *TransactionsTable) Insert(ctx context.Context, create *TransactionCreate) (uuid.UUID, error) {
	setter := &bobgen.TransactionSetter{
		AccountID:       omit.From(create.AccountID),
		CategoryID:      omit.From(create.CategoryID),
		Amount:          omit.From(create.Amount),
		TransactionName: omit.From(create.TransactionName),
	}
	if !create.TransactionDate.IsZero() {
		setter.TransactionDate = omit.From(create.TransactionDate)
	}
	row, err := bobgen.Transactions.Insert(setter).One(ctx, t.exec)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, err
}

// List returns transactions matching the filter. Nil filter returns all.
func (t *TransactionsTable) List(ctx context.Context, filter *TransactionFilter) ([]*Transaction, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]
	if filter != nil {
		var whereMods []mods.Where[*dialect.SelectQuery]
		if filter.AccountID != nil {
			whereMods = append(whereMods, bobgen.SelectWhere.Transactions.AccountID.EQ(*filter.AccountID))
		}
		if filter.CategoryID != nil {
			whereMods = append(whereMods, bobgen.SelectWhere.Transactions.CategoryID.EQ(*filter.CategoryID))
		}
		if len(whereMods) == 1 {
			queryMods = append(queryMods, whereMods[0])
		} else if len(whereMods) > 1 {
			queryMods = append(queryMods, psql.WhereAnd(whereMods...))
		}
	}
	rows, err := bobgen.Transactions.Query(queryMods...).All(ctx, t.exec)
	if err != nil {
		return nil, err
	}
	result := make([]*Transaction, len(rows))
	for i, row := range rows {
		result[i] = bobTransactionToTransaction(row)
	}
	return result, nil
}

func bobTransactionToTransaction(row *bobgen.Transaction) *Transaction {
	return &Transaction{
		ID:              row.ID,
		AccountID:       row.AccountID,
		CategoryID:      row.CategoryID,
		Amount:          row.Amount,
		TransactionName: row.TransactionName,
		TransactionDate: row.TransactionDate,
	}
}
