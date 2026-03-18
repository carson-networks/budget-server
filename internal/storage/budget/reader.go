package budget

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type Reader struct {
	exec bob.Executor
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{exec: exec}
}

// monthYearToTime returns the first day of the given month/year in UTC.
func monthYearToTime(month, year int) time.Time {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
}

// ListForRange returns budgets for [startMonth/startYear, endMonth/endYear]: one "starting" row per category (latest with month <= start) plus all rows in (start, end].
func (r *Reader) ListForRange(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]*Budget, error) {
	start := monthYearToTime(startMonth, startYear)
	end := monthYearToTime(endMonth, endYear)

	startingRows, err := bobgen.Budgets.Query(
		bobgen.SelectWhere.Budgets.Month.LTE(start),
		sm.OrderBy(bobgen.Budgets.Columns.CategoryID).Asc(),
		sm.OrderBy(bobgen.Budgets.Columns.Month).Desc(),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}

	starting := deduplicateToLatestPerCategory(startingRows)

	inRangeRows, err := bobgen.Budgets.Query(
		psql.WhereAnd(
			bobgen.SelectWhere.Budgets.Month.GT(start),
			bobgen.SelectWhere.Budgets.Month.LTE(end),
		),
		sm.OrderBy(bobgen.Budgets.Columns.CategoryID).Asc(),
		sm.OrderBy(bobgen.Budgets.Columns.Month).Asc(),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}

	inRange := make([]*Budget, len(inRangeRows))
	for i, row := range inRangeRows {
		inRange[i] = bobBudgetToBudget(row)
	}

	result := make([]*Budget, 0, len(starting)+len(inRange))
	result = append(result, starting...)
	result = append(result, inRange...)
	return result, nil
}

// deduplicateToLatestPerCategory returns one row per category from rows ordered by (category_id, month DESC).
// The first row per category_id is kept (it has the latest month).
func deduplicateToLatestPerCategory(rows []*bobgen.Budget) []*Budget {
	seenCategory := make(map[uuid.UUID]bool)
	var result []*Budget
	for _, row := range rows {
		if !seenCategory[row.CategoryID] {
			seenCategory[row.CategoryID] = true
			result = append(result, bobBudgetToBudget(row))
		}
	}
	return result
}
