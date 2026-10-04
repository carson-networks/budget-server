package budget

import (
	"context"
	"sort"
	"testing"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetRecords_AreEffectiveUntilTheNextRecord(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	rows := []*bobgen.Budget{
		{CategoryID: id, Month: monthYearToTime(2, 2025), Amount: decimal.NewFromInt(400)},
		{CategoryID: id, Month: monthYearToTime(6, 2025), Amount: decimal.NewFromInt(800)},
		{CategoryID: id, Month: monthYearToTime(1, 2026), Amount: decimal.NewFromInt(1000)},
		{CategoryID: other, Month: monthYearToTime(2, 2025), Amount: decimal.NewFromInt(200)},
	}
	budgetAt := func(month, year int) []*Budget {
		var starting []*bobgen.Budget
		for _, row := range rows {
			if !row.Month.After(monthYearToTime(month, year)) {
				starting = append(starting, row)
			}
		}
		sort.Slice(starting, func(i, j int) bool {
			if starting[i].CategoryID == starting[j].CategoryID {
				return starting[i].Month.After(starting[j].Month)
			}
			return starting[i].CategoryID.String() < starting[j].CategoryID.String()
		})
		return deduplicateToLatestPerCategory(starting)
	}
	check := func(month, year int, amount string) {
		t.Helper()
		for _, budget := range budgetAt(month, year) {
			if budget.CategoryID == id {
				assert.Equal(t, amount, budget.Amount.String())
				return
			}
		}
		t.Fatal("missing budget")
	}
	check(3, 2025, "400")
	check(5, 2025, "400")
	check(6, 2025, "800")
	check(12, 2025, "800")
	check(1, 2026, "1000")

	set := &BudgetSet{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(450)}
	list := func(_ context.Context, startMonth, startYear, _, _ int) ([]*Budget, error) {
		return budgetAt(startMonth, startYear), nil
	}
	write := func(_ context.Context, change *BudgetSet, replaceExisting bool) error {
		for _, row := range rows {
			if row.CategoryID == change.CategoryID && row.Month.Equal(monthYearToTime(change.Month, change.Year)) {
				if replaceExisting {
					row.Amount = change.Amount
				}
				return nil
			}
		}
		rows = append(rows, &bobgen.Budget{CategoryID: change.CategoryID, Month: monthYearToTime(change.Month, change.Year), Amount: change.Amount})
		return nil
	}
	require.NoError(t, setBudget(t.Context(), set, list, write, nil))
	check(3, 2025, "450")
	check(4, 2025, "400")
	check(5, 2025, "400")
	check(6, 2025, "800")
	check(1, 2026, "1000")

	set.Amount = decimal.NewFromInt(550)
	set.OverwriteFutureMonths = true
	deleteFuture := func(_ context.Context) error {
		var kept []*bobgen.Budget
		for _, row := range rows {
			if row.CategoryID != id || !row.Month.After(monthYearToTime(3, 2025)) {
				kept = append(kept, row)
			}
		}
		rows = kept
		return nil
	}
	require.NoError(t, setBudget(t.Context(), set, list, write, deleteFuture))
	check(2, 2025, "400")
	check(3, 2025, "550")
	check(6, 2025, "550")
	check(1, 2040, "550")
	for _, budget := range budgetAt(1, 2040) {
		if budget.CategoryID == other {
			assert.Equal(t, "200", budget.Amount.String())
		}
	}
}
