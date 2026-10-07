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
// If OverwriteFutureMonths is true, deletes budgets for this category in months after the given month first (same transaction), and the new amount carries forward.
// Otherwise the next month keeps the amount that was in effect before this edit when that month has no row of its own.
func (w *Writer) Set(ctx context.Context, set *BudgetSet) error {
	var prior decimal.Decimal
	pinNext := false
	if set.OverwriteFutureMonths {
		if err := w.deleteByCategoryAndMonthsAfter(ctx, set.CategoryID, set.Month, set.Year); err != nil {
			return err
		}
	} else {
		next := monthYearToTime(set.Month, set.Year).AddDate(0, 1, 0)
		rows, err := w.ListForRange(ctx, int(next.Month()), next.Year(), int(next.Month()), next.Year())
		if err != nil {
			return err
		}
		prior, pinNext = amountToPreserve(rows, set.CategoryID, set.Month, set.Year)
	}

	if _, err := bobgen.Budgets.Insert(
		newBudgetSetter(set.CategoryID, monthYearToTime(set.Month, set.Year), set.Amount),
		im.OnConflict("category_id", "month").DoUpdate(im.SetExcluded("amount")),
	).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !pinNext {
		return nil
	}
	next := monthYearToTime(set.Month, set.Year).AddDate(0, 1, 0)
	_, err := bobgen.Budgets.Insert(
		newBudgetSetter(set.CategoryID, next, prior),
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

// amountToPreserve is the amount to copy onto the month after editedMonth/editedYear.
// The bool is false when that next month already has its own row.
// Otherwise the amount is the latest budget for the category on or before the edited month, or zero when none exists.
func amountToPreserve(rows []*Budget, categoryID uuid.UUID, editedMonth, editedYear int) (decimal.Decimal, bool) {
	edited := monthYearToTime(editedMonth, editedYear)
	next := edited.AddDate(0, 1, 0)
	nextMonth, nextYear := int(next.Month()), next.Year()

	var (
		found  bool
		latest time.Time
		amount decimal.Decimal
	)
	for _, row := range rows {
		if row.CategoryID != categoryID {
			continue
		}
		if row.Month == nextMonth && row.Year == nextYear {
			return decimal.Decimal{}, false
		}
		rowTime := monthYearToTime(row.Month, row.Year)
		if rowTime.After(edited) {
			continue
		}
		if !found || rowTime.After(latest) {
			found = true
			latest = rowTime
			amount = row.Amount
		}
	}
	if !found {
		return decimal.Zero, true
	}
	return amount, true
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
