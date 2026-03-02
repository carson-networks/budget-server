package account

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type Reader struct {
	exec bob.Executor
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{exec: exec}
}

func (r *Reader) List(ctx context.Context, filter *AccountFilter) (*AccountListResult, error) {
	limit := 20
	offset := 0
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
		}
		offset = filter.Offset
	}

	queryMods := []bob.Mod[*dialect.SelectQuery]{
		sm.Limit(limit + 1),
		sm.Offset(offset),
		sm.OrderBy(bobgen.Accounts.Columns.Name).Asc(),
		sm.OrderBy(bobgen.Accounts.Columns.ID).Asc(),
	}
	rows, err := bobgen.Accounts.Query(queryMods...).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return &AccountListResult{Accounts: nil, NextCursor: nil}, nil
	}

	var nextCursor *AccountCursor
	if len(rows) > limit {
		rows = rows[:limit]
		nextCursor = &AccountCursor{
			Position: offset + limit,
			Limit:    limit,
		}
	}

	result := make([]*Account, len(rows))
	for i, row := range rows {
		result[i] = bobAccountToAccount(row)
	}
	return &AccountListResult{Accounts: result, NextCursor: nextCursor}, nil
}

func (r *Reader) FindByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	row, err := bobgen.FindAccount(ctx, r.exec, id)
	if err != nil {
		return nil, err
	}
	return bobAccountToAccount(row), nil
}
