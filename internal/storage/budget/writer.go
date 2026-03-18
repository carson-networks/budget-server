package budget

import (
	"context"
	"time"

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

// Set inserts or updates a budget row (upsert on category_id, month).
func (w *Writer) Set(ctx context.Context, set *BudgetSet) error {
	setter := &bobgen.BudgetSetter{
		CategoryID: omit.From(set.CategoryID),
		Month:      omit.From(set.Month),
		Amount:     omit.From(set.Amount),
	}
	_, err := bobgen.Budgets.Insert(
		setter,
		im.OnConflict("category_id", "month").DoUpdate(im.SetExcluded("amount")),
	).One(ctx, w.tx)
	return err
}

// DeleteByCategoryAndMonthsAfter deletes all budgets for the given category where month > the given month.
func (w *Writer) DeleteByCategoryAndMonthsAfter(ctx context.Context, categoryID uuid.UUID, month time.Time) error {
	_, err := bobgen.Budgets.Delete(
		dm.Where(psql.And(
			bobgen.Budgets.Columns.CategoryID.EQ(psql.Arg(categoryID)),
			bobgen.Budgets.Columns.Month.GT(psql.Arg(month)),
		)),
	).Exec(ctx, w.tx)
	return err
}
