package transaction

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/mods"
)

type Reader struct {
	exec bob.Executor
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{exec: exec}
}

func (r *Reader) FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	row, err := bobgen.FindTransaction(ctx, r.exec, id)
	if err != nil {
		return nil, err
	}
	return bobTransactionToTransaction(row), nil
}

func (r *Reader) List(ctx context.Context, filter *TransactionFilter) (*TransactionListResult, error) {
	limit := 20
	offset := 0
	var maxCreationTime *time.Time
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
		}
		offset = filter.Offset
		maxCreationTime = filter.MaxCreationTime
	}

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
	}
	queryMods = append(queryMods,
		sm.Limit(limit+1),
		sm.Offset(offset),
		sm.OrderBy(bobgen.Transactions.Columns.CreatedAt).Desc(),
		sm.OrderBy(bobgen.Transactions.Columns.ID).Desc(),
	)
	rows, err := bobgen.Transactions.Query(queryMods...).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return &TransactionListResult{Transactions: nil, NextCursor: nil}, nil
	}

	var nextCursor *TransactionCursor
	if len(rows) > limit {
		rows = rows[:limit]
		cursorMaxCreationTime := rows[0].CreatedAt
		if maxCreationTime != nil {
			cursorMaxCreationTime = *maxCreationTime
		}
		nextCursor = &TransactionCursor{
			Position:        offset + limit,
			Limit:           limit,
			MaxCreationTime: cursorMaxCreationTime,
		}
	}

	result := make([]*Transaction, len(rows))
	for i, row := range rows {
		result[i] = bobTransactionToTransaction(row)
	}
	return &TransactionListResult{Transactions: result, NextCursor: nextCursor}, nil
}
