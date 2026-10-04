package budget

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stephenafamo/bob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func budgetTestWriter(t *testing.T) *Writer {
	t.Helper()
	dsn := os.Getenv("BUDGET_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("BUDGET_TEST_DATABASE_URL is required for budget persistence tests")
	}
	db, err := bob.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	tx, err := db.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tx.Rollback(context.Background())) })
	_, err = tx.ExecContext(t.Context(), `
		CREATE TEMP TABLE categories (id UUID PRIMARY KEY);
		CREATE TEMP TABLE budgets (
			category_id UUID NOT NULL REFERENCES categories(id),
			month DATE NOT NULL,
			amount DECIMAL(100, 4) NOT NULL,
			PRIMARY KEY (category_id, month)
		);
	`)
	require.NoError(t, err)
	return NewWriter(tx)
}

func seedBudgets(t *testing.T, w *Writer, id uuid.UUID, amounts map[string]int64) {
	t.Helper()
	_, err := w.tx.ExecContext(t.Context(), "INSERT INTO categories (id) VALUES ($1)", id)
	require.NoError(t, err)
	for month, amount := range amounts {
		_, err = w.tx.ExecContext(t.Context(), "INSERT INTO budgets (category_id, month, amount) VALUES ($1, $2, $3)", id, month+"-01", amount)
		require.NoError(t, err)
	}
}

func effectiveAmounts(t *testing.T, w *Writer, id uuid.UUID, months ...string) []string {
	t.Helper()
	amounts := make([]string, len(months))
	for i, month := range months {
		yearMonth, err := time.Parse("2006-01", month)
		require.NoError(t, err)
		rows, err := w.ListForRange(t.Context(), int(yearMonth.Month()), yearMonth.Year(), int(yearMonth.Month()), yearMonth.Year())
		require.NoError(t, err)
		amounts[i] = "0"
		for _, row := range rows {
			if row.CategoryID == id {
				amounts[i] = row.Amount.String()
			}
		}
	}
	return amounts
}

func TestSet_OnlyChangesSelectedMonth(t *testing.T) {
	tests := []struct {
		name     string
		seed     map[string]int64
		month    int
		amount   int64
		months   []string
		expected []string
	}{
		{"carry forward", map[string]int64{"2025-02": 400, "2025-06": 800}, 3, 450, []string{"2025-02", "2025-03", "2025-04", "2025-05", "2025-06", "2026-01"}, []string{"400", "450", "400", "400", "800", "800"}},
		{"replace existing month", map[string]int64{"2025-02": 400, "2025-03": 500}, 3, 450, []string{"2025-03", "2025-04", "2026-01"}, []string{"450", "500", "500"}},
		{"explicit following month", map[string]int64{"2025-02": 400, "2025-04": 600, "2025-06": 800}, 3, 450, []string{"2025-03", "2025-04", "2025-05", "2025-06"}, []string{"450", "600", "600", "800"}},
		{"unset budget", map[string]int64{"2026-03": 800}, 12, 450, []string{"2025-11", "2025-12", "2026-01", "2026-02", "2026-03"}, []string{"0", "450", "0", "0", "800"}},
		{"zero carry forward", map[string]int64{"2025-02": 0}, 3, 450, []string{"2025-03", "2025-04", "2026-01"}, []string{"450", "0", "0"}},
		{"clear selected month", map[string]int64{"2025-02": 400}, 3, 0, []string{"2025-03", "2025-04", "2026-01"}, []string{"0", "400", "400"}},
		{"December rollover", map[string]int64{"2025-10": 400}, 12, 450, []string{"2025-11", "2025-12", "2026-01", "2026-02"}, []string{"400", "450", "400", "400"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := budgetTestWriter(t)
			id := uuid.Must(uuid.NewV4())
			seedBudgets(t, w, id, tt.seed)
			require.NoError(t, w.Set(t.Context(), &BudgetSet{CategoryID: id, Year: 2025, Month: tt.month, Amount: decimal.NewFromInt(tt.amount)}))
			assert.Equal(t, tt.expected, effectiveAmounts(t, w, id, tt.months...))
		})
	}
}

func TestSet_OverwritesEveryFollowingMonth(t *testing.T) {
	w := budgetTestWriter(t)
	id := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	seedBudgets(t, w, id, map[string]int64{"2025-02": 400, "2025-04": 600, "2025-06": 800, "2026-01": 1000})
	seedBudgets(t, w, other, map[string]int64{"2025-02": 200, "2025-06": 300})
	require.NoError(t, w.Set(t.Context(), &BudgetSet{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(450), OverwriteFutureMonths: true}))
	assert.Equal(t, []string{"400", "450", "450", "450", "450", "450"}, effectiveAmounts(t, w, id, "2025-02", "2025-03", "2025-04", "2025-06", "2026-01", "2040-01"))
	assert.Equal(t, []string{"200", "300"}, effectiveAmounts(t, w, other, "2025-04", "2026-01"))
}

func TestSet_RepeatedEditsPreserveFollowingMonthsUntilOverwrite(t *testing.T) {
	w := budgetTestWriter(t)
	id := uuid.Must(uuid.NewV4())
	seedBudgets(t, w, id, map[string]int64{"2025-02": 400, "2025-06": 800})
	for _, amount := range []int64{450, 500} {
		require.NoError(t, w.Set(t.Context(), &BudgetSet{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(amount)}))
	}
	assert.Equal(t, []string{"500", "400", "800"}, effectiveAmounts(t, w, id, "2025-03", "2025-04", "2025-06"))
	require.NoError(t, w.Set(t.Context(), &BudgetSet{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(550), OverwriteFutureMonths: true}))
	assert.Equal(t, []string{"550", "550", "550"}, effectiveAmounts(t, w, id, "2025-03", "2025-04", "2025-06"))
}
