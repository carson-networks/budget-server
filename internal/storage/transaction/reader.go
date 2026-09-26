package transaction

import (
	"context"
	"database/sql"
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

func listWhereMods(filter *TransactionFilter) []bob.Mod[*dialect.SelectQuery] {
	if filter == nil {
		return nil
	}
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
	switch len(whereMods) {
	case 0:
		return nil
	case 1:
		return []bob.Mod[*dialect.SelectQuery]{whereMods[0]}
	default:
		return []bob.Mod[*dialect.SelectQuery]{psql.WhereAnd(whereMods...)}
	}
}

// pageListResult builds a paginated list result from a limit+1 probe page and total count.
// rows may contain up to limit+1 items; the extra row (if present) only signals a next page.
func pageListResult(rows []*Transaction, limit, offset int, maxCreationTime *time.Time, totalCount int) *TransactionListResult {
	if len(rows) == 0 {
		return &TransactionListResult{Transactions: nil, NextCursor: nil, TotalCount: totalCount}
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

	return &TransactionListResult{
		Transactions: rows,
		NextCursor:   nextCursor,
		TotalCount:   totalCount,
	}
}

func scanListRow(scanner interface {
	Scan(dest ...any) error
}) (*Transaction, int64, error) {
	var (
		id              uuid.UUID
		accountID       uuid.UUID
		categoryID      uuid.NullUUID
		amount          decimal.Decimal
		transactionName string
		transactionDate time.Time
		createdAt       time.Time
		merchantName    sql.NullString
		totalCount      int64
	)
	if err := scanner.Scan(
		&id, &accountID, &categoryID, &amount, &transactionName,
		&transactionDate, &createdAt, &merchantName, &totalCount,
	); err != nil {
		return nil, 0, err
	}
	tx := &Transaction{
		ID:              id,
		AccountID:       accountID,
		Amount:          amount,
		TransactionName: transactionName,
		TransactionDate: transactionDate,
		CreatedAt:       createdAt,
	}
	if categoryID.Valid {
		tx.CategoryID = &categoryID.UUID
	}
	if merchantName.Valid {
		name := merchantName.String
		tx.MerchantName = &name
	}
	return tx, totalCount, nil
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

	cols := bobgen.Transactions.Columns
	// Single query: page rows plus total_count from Postgres COUNT(*) OVER()
	// (window runs over the filtered set before LIMIT/OFFSET).
	queryMods := []bob.Mod[*dialect.SelectQuery]{
		sm.Columns(
			cols.ID,
			cols.AccountID,
			cols.CategoryID,
			cols.Amount,
			cols.TransactionName,
			cols.TransactionDate,
			cols.CreatedAt,
			cols.MerchantName,
			psql.Raw("COUNT(*) OVER()"),
		),
		sm.From(bobgen.Transactions.NameAsExpr()),
	}
	queryMods = append(queryMods, listWhereMods(filter)...)
	queryMods = append(queryMods,
		sm.Limit(limit+1),
		sm.Offset(offset),
		sm.OrderBy(cols.CreatedAt).Desc(),
		sm.OrderBy(cols.ID).Desc(),
	)

	q := psql.Select(queryMods...)
	sqlStr, args, err := q.Build(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.exec.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		result     []*Transaction
		totalCount int64
	)
	for rows.Next() {
		tx, count, err := scanListRow(rows)
		if err != nil {
			return nil, err
		}
		totalCount = count
		result = append(result, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pageListResult(result, limit, offset, maxCreationTime, int(totalCount)), nil
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
		sm.From(bobgen.Transactions.NameAsExpr()),
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
		var nullCatID uuid.NullUUID
		var total decimal.Decimal
		if err := rows.Scan(&scanYear, &scanMonth, &nullCatID, &total); err != nil {
			return nil, err
		}
		catID := uuid.Nil
		if nullCatID.Valid {
			catID = nullCatID.UUID
		}
		t := firstOfMonth(scanYear, scanMonth)
		byMonth[t] = append(byMonth[t], CategoryTotal{CategoryID: catID, Total: total})
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
