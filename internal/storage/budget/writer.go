package budget

import (
	"context"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
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

// Set changes one month, or that month and every following month when requested.
func (w *Writer) Set(ctx context.Context, set *BudgetSet) error {
	return setBudget(ctx, set, w.ListForRange, w.write, func(ctx context.Context) error {
		return w.deleteByCategoryAndMonthsAfter(ctx, set.CategoryID, set.Month, set.Year)
	})
}

func (w *Writer) write(ctx context.Context, set *BudgetSet, replaceExisting bool) error {
	setter := &bobgen.BudgetSetter{
		CategoryID: omit.From(set.CategoryID),
		Month:      omit.From(monthYearToTime(set.Month, set.Year)),
		Amount:     omit.From(set.Amount),
	}
	conflict := im.OnConflict("category_id", "month").DoNothing()
	if replaceExisting {
		conflict = im.OnConflict("category_id", "month").DoUpdate(im.SetExcluded("amount"))
	}
	_, err := bobgen.Budgets.Insert(setter, conflict).Exec(ctx, w.tx)
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
