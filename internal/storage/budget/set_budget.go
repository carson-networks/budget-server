package budget

import (
	"context"

	"github.com/shopspring/decimal"
)

type listBudgetsFunc func(context.Context, int, int, int, int) ([]*Budget, error)
type writeBudgetFunc func(context.Context, *BudgetSet, bool) error

func setBudget(ctx context.Context, set *BudgetSet, list listBudgetsFunc, write writeBudgetFunc, deleteFuture func(context.Context) error) error {
	if set.OverwriteFutureMonths {
		if err := deleteFuture(ctx); err != nil {
			return err
		}
	} else if err := preserveFollowingMonth(ctx, set, list, write); err != nil {
		return err
	}
	return write(ctx, set, true)
}

func preserveFollowingMonth(ctx context.Context, set *BudgetSet, list listBudgetsFunc, write writeBudgetFunc) error {
	nextMonth := monthYearToTime(set.Month, set.Year).AddDate(0, 1, 0)
	month, year := int(nextMonth.Month()), nextMonth.Year()
	budgets, err := list(ctx, month, year, month, year)
	if err != nil {
		return err
	}
	amount := decimal.Zero
	for _, row := range budgets {
		if row.CategoryID != set.CategoryID {
			continue
		}
		if row.Month == month && row.Year == year {
			return nil
		}
		amount = row.Amount
		break
	}
	return write(ctx, &BudgetSet{CategoryID: set.CategoryID, Month: month, Year: year, Amount: amount}, false)
}
