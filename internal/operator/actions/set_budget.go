package actions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/budget"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

var (
	ErrCategoryNotFoundForBudget = errors.New("category not found")
	ErrInvalidMonth              = errors.New("month must be between 1 and 12")
)

type SetBudget struct {
	CategoryID            uuid.UUID
	Month                 int // 1-12
	Year                  int
	Amount                decimal.Decimal
	OverwriteFutureMonths bool
	IAction
}

func (a *SetBudget) Perform(ctx context.Context, writer *storage.Writer) error {
	if a.Month < 1 || a.Month > 12 {
		return ErrInvalidMonth
	}

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

	return writer.Budget.Set(ctx, &budget.BudgetSet{
		CategoryID:            a.CategoryID,
		Month:                 a.Month,
		Year:                  a.Year,
		Amount:                a.Amount,
		OverwriteFutureMonths: a.OverwriteFutureMonths,
	})
}
