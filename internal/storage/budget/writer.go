package budget

import (
	"context"
	"database/sql"
	"errors"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
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

// Set changes one month, or that month and every following month when requested.
func (w *Writer) Set(ctx context.Context, set *BudgetSet) error {
	monthTime := monthYearToTime(set.Month, set.Year)
	// Serialize writes for a category before reading its carry-forward value.
	if _, err := w.tx.ExecContext(ctx, "SELECT id FROM categories WHERE id = $1 FOR UPDATE", set.CategoryID); err != nil {
		return err
	}
	if set.OverwriteFutureMonths {
		if err := w.deleteByCategoryAndMonthsAfter(ctx, set.CategoryID, set.Month, set.Year); err != nil {
			return err
		}
	} else if err := w.preserveFollowingMonth(ctx, set); err != nil {
		return err
	}
	setter := &bobgen.BudgetSetter{
		CategoryID: omit.From(set.CategoryID),
		Month:      omit.From(monthTime),
		Amount:     omit.From(set.Amount),
	}
	_, err := bobgen.Budgets.Insert(
		setter,
		im.OnConflict("category_id", "month").DoUpdate(im.SetExcluded("amount")),
	).One(ctx, w.tx)
	return err
}

func (w *Writer) preserveFollowingMonth(ctx context.Context, set *BudgetSet) error {
	nextMonth := monthYearToTime(set.Month, set.Year).AddDate(0, 1, 0)
	previous, err := bobgen.Budgets.Query(
		bobgen.SelectWhere.Budgets.CategoryID.EQ(set.CategoryID),
		bobgen.SelectWhere.Budgets.Month.LTE(nextMonth),
		sm.OrderBy(bobgen.Budgets.Columns.Month).Desc(),
		sm.Limit(1),
	).One(ctx, w.tx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	amount := decimal.Zero
	if previous != nil {
		amount = previous.Amount
	}
	_, err = bobgen.Budgets.Insert(&bobgen.BudgetSetter{
		CategoryID: omit.From(set.CategoryID),
		Month:      omit.From(nextMonth),
		Amount:     omit.From(amount),
	}, im.OnConflict("category_id", "month").DoNothing()).Exec(ctx, w.tx)
	return err
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
