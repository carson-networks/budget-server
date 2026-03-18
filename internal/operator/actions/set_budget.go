package actions

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/budget"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

var (
	ErrCategoryNotFoundForBudget = errors.New("category not found")
)

type SetBudget struct {
	CategoryID            uuid.UUID
	Month                 time.Time
	Amount                decimal.Decimal
	OverwriteFutureMonths bool
	IAction
}

func (a *SetBudget) Perform(ctx context.Context, writer *storage.Writer) error {
	cat, err := writer.Category.GetByID(ctx, a.CategoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCategoryNotFoundForBudget
		}
		return err
	}
	if cat.IsParent {
		return ErrCategoryIsParent
	}

	if a.OverwriteFutureMonths {
		err = writer.Budget.DeleteByCategoryAndMonthsAfter(ctx, a.CategoryID, a.Month)
		if err != nil {
			return err
		}
	}

	return writer.Budget.Set(ctx, &budget.BudgetSet{
		CategoryID: a.CategoryID,
		Month:      a.Month,
		Amount:     a.Amount,
	})
}
