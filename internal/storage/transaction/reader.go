package transaction

import (
	"context"
	"sort"
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
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

func firstOfMonth(year, month int) time.Time {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
}

func (r *Reader) TotalsByMonthAndCategory(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]MonthTotals, error) {
	start := firstOfMonth(startYear, startMonth)
	endExclusive := firstOfMonth(endYear, endMonth).AddDate(0, 1, 0)

	cols := bobgen.Transactions.Columns
	year := psql.Cast(psql.Raw("EXTRACT(YEAR FROM ? AT TIME ZONE 'UTC')", cols.TransactionDate), "int")
	month := psql.Cast(psql.Raw("EXTRACT(MONTH FROM ? AT TIME ZONE 'UTC')", cols.TransactionDate), "int")
	q := psql.Select(
		sm.Columns(year, month, cols.CategoryID, psql.F("SUM", cols.Amount)),
		sm.From(bobgen.Transactions),
		psql.WhereAnd(
			bobgen.SelectWhere.Transactions.TransactionDate.GTE(start),
			bobgen.SelectWhere.Transactions.TransactionDate.LT(endExclusive),
		),
		sm.GroupBy(year),
		sm.GroupBy(month),
		sm.GroupBy(cols.CategoryID),
		sm.OrderBy(year),
		sm.OrderBy(month),
		sm.OrderBy(cols.CategoryID),
	)

	sqlStr, args, err := q.Build(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.exec.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byMonth := make(map[time.Time][]CategoryTotal)
	for rows.Next() {
		var scanYear, scanMonth int
		var scanCatergoryID uuid.UUID
		var total decimal.Decimal
		if err := rows.Scan(&scanYear, &scanMonth, &scanCatergoryID, &total); err != nil {
			return nil, err
		}
		t := firstOfMonth(scanYear, scanMonth)
		byMonth[t] = append(byMonth[t], CategoryTotal{CategoryID: scanCatergoryID, Total: total})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var months []time.Time
	for t := start; t.Before(endExclusive); t = t.AddDate(0, 1, 0) {
		months = append(months, t)
	}
	if len(months) == 0 {
		return nil, nil
	}
	out := make([]MonthTotals, 0, len(months))
	for _, t := range months {
		cats := byMonth[t]
		sort.Slice(cats, func(i, j int) bool {
			return cats[i].CategoryID.String() < cats[j].CategoryID.String()
		})
		out = append(out, MonthTotals{
			Year:       t.Year(),
			Month:      int(t.Month()),
			Categories: cats,
		})
	}
	return out, nil
}
