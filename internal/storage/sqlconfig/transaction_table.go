package sqlconfig

import (
	"context"
	"database/sql"

	"github.com/aarondl/opt/omit"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/mods"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
)

var _ ITransactionTable = (*TransactionsTable)(nil)

type TransactionsTable struct {
	exec bob.Executor
}

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
		if filter.MaxCreationTime != nil {
			whereMods = append(whereMods, bobgen.SelectWhere.Transactions.CreatedAt.LTE(*filter.MaxCreationTime))
		}
		if len(whereMods) == 1 {
			queryMods = append(queryMods, whereMods[0])
		} else if len(whereMods) > 1 {
			queryMods = append(queryMods, psql.WhereAnd(whereMods...))
		}
		if filter.Limit > 0 {
			queryMods = append(queryMods, sm.Limit(filter.Limit+1))
		}
		if filter.Offset > 0 {
			queryMods = append(queryMods, sm.Offset(filter.Offset))
		}
	}
	queryMods = append(queryMods,
		sm.OrderBy(bobgen.Transactions.Columns.CreatedAt).Desc(),
		sm.OrderBy(bobgen.Transactions.Columns.ID).Desc(),
	)
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
		CreatedAt:       row.CreatedAt,
	}
}
