package budget

import (
	"context"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

type Writer struct {
	tx bob.Tx
	Reader
}

func NewWriter(tx bob.Tx) *Writer {
	return &Writer{
		tx: tx,
		Reader: Reader{
			exec: tx,
		},
	}
}

// Set inserts or updates a budget row (upsert on category_id, month).
// If OverwriteFutureMonths is true, deletes budgets for this category in months after the given month first (same transaction).
// Otherwise the following month keeps the amount already in effect, so this edit does not carry forward.
func (w *Writer) Set(ctx context.Context, set *BudgetSet) error {
	if set.OverwriteFutureMonths {
		if err := w.deleteByCategoryAndMonthsAfter(ctx, set.CategoryID, set.Month, set.Year); err != nil {
			return err
		}
	} else if err := w.preserveNextMonth(ctx, set); err != nil {
		return err
	}

	_, err := bobgen.Budgets.Insert(
		newBudgetSetter(set.CategoryID, monthYearToTime(set.Month, set.Year), set.Amount),
		im.OnConflict("category_id", "month").DoUpdate(im.SetExcluded("amount")),
	).Exec(ctx, w.tx)
	return err
}

// preserveNextMonth records the amount already in effect for the next month when that month has no row.
func (w *Writer) preserveNextMonth(ctx context.Context, set *BudgetSet) error {
	next := monthYearToTime(set.Month, set.Year).AddDate(0, 1, 0)
	month, year := int(next.Month()), next.Year()
	rows, err := w.ListForRange(ctx, month, year, month, year)
	if err != nil {
		return err
	}
	amount, ok := amountToPreserve(rows, set.CategoryID, month, year)
	if !ok {
		return nil
	}
	_, err = bobgen.Budgets.Insert(
		newBudgetSetter(set.CategoryID, next, amount),
		im.OnConflict("category_id", "month").DoNothing(),
	).Exec(ctx, w.tx)
	return err
}

func newBudgetSetter(categoryID uuid.UUID, month time.Time, amount decimal.Decimal) *bobgen.BudgetSetter {
	return &bobgen.BudgetSetter{
		CategoryID: omit.From(categoryID),
		Month:      omit.From(month),
		Amount:     omit.From(amount),
	}
}

// amountToPreserve is the amount to copy onto month/year so an edit of the previous month does not carry forward.
// The bool is false when that month already has its own row.
func amountToPreserve(rows []*Budget, categoryID uuid.UUID, month, year int) (decimal.Decimal, bool) {
	for _, row := range rows {
		if row.CategoryID != categoryID {
			continue
		}
		if row.Month == month && row.Year == year {
			return decimal.Decimal{}, false
		}
		return row.Amount, true
	}
	return decimal.Zero, true
}

// deleteByCategoryAndMonthsAfter deletes all budgets for the given category where month > the given month/year.
func (w *Writer) deleteByCategoryAndMonthsAfter(ctx context.Context, categoryID uuid.UUID, month, year int) error {
	monthTime := monthYearToTime(month, year)
	_, err := bobgen.Budgets.Delete(
		dm.Where(psql.And(
			bobgen.Budgets.Columns.CategoryID.EQ(psql.Arg(categoryID)),
			bobgen.Budgets.Columns.Month.GT(psql.Arg(monthTime)),
		)),
	).Exec(ctx, w.tx)
	return err
}
