package budget

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetBudget_PreservesFollowingRecord(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	tests := []struct {
		name           string
		month          int
		following      []*Budget
		preserveAmount int64
		alreadyDefined bool
	}{
		{name: "carry forward", month: 3, following: []*Budget{{CategoryID: id, Year: 2025, Month: 2, Amount: decimal.NewFromInt(400)}}, preserveAmount: 400},
		{name: "replace current record", month: 3, following: []*Budget{{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(500)}}, preserveAmount: 500},
		{name: "existing next-month record", month: 3, following: []*Budget{{CategoryID: id, Year: 2025, Month: 4, Amount: decimal.NewFromInt(600)}}, alreadyDefined: true},
		{name: "no previous record", month: 3},
		{name: "zero budget", month: 3, following: []*Budget{{CategoryID: id, Year: 2025, Month: 2, Amount: decimal.Zero}}},
		{name: "other category", month: 3, following: []*Budget{{CategoryID: other, Year: 2025, Month: 4, Amount: decimal.NewFromInt(999)}, {CategoryID: id, Year: 2025, Month: 2, Amount: decimal.NewFromInt(400)}}, preserveAmount: 400},
		{name: "December rollover", month: 12, following: []*Budget{{CategoryID: id, Year: 2025, Month: 10, Amount: decimal.NewFromInt(400)}}, preserveAmount: 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := &BudgetSet{CategoryID: id, Year: 2025, Month: tt.month, Amount: decimal.NewFromInt(450)}
			next := monthYearToTime(tt.month, 2025).AddDate(0, 1, 0)
			var writes []*BudgetSet
			list := func(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]*Budget, error) {
				assert.Equal(t, t.Context(), ctx)
				assert.Equal(t, []int{int(next.Month()), next.Year(), int(next.Month()), next.Year()}, []int{startMonth, startYear, endMonth, endYear})
				return tt.following, nil
			}
			write := func(_ context.Context, change *BudgetSet, replaceExisting bool) error {
				assert.Equal(t, change == set, replaceExisting)
				writes = append(writes, change)
				return nil
			}
			err := setBudget(t.Context(), set, list, write, func(context.Context) error {
				t.Fatal("month-only edits must preserve later records")
				return nil
			})
			require.NoError(t, err)
			if tt.alreadyDefined {
				assert.Equal(t, []*BudgetSet{set}, writes)
			} else {
				require.Len(t, writes, 2)
				assert.Equal(t, id, writes[0].CategoryID)
				assert.Equal(t, next.Year(), writes[0].Year)
				assert.Equal(t, int(next.Month()), writes[0].Month)
				assert.True(t, writes[0].Amount.Equal(decimal.NewFromInt(tt.preserveAmount)))
				assert.Same(t, set, writes[1])
			}
		})
	}
}

func TestSetBudget_OverwriteRemovesLaterRecordsBeforeWriting(t *testing.T) {
	set := &BudgetSet{CategoryID: uuid.Must(uuid.NewV4()), Year: 2025, Month: 3, Amount: decimal.NewFromInt(450), OverwriteFutureMonths: true}
	var operations []string
	list := func(context.Context, int, int, int, int) ([]*Budget, error) {
		t.Fatal("overwrite does not need to preserve a following record")
		return nil, nil
	}
	write := func(ctx context.Context, change *BudgetSet, replaceExisting bool) error {
		assert.True(t, replaceExisting)
		assert.Equal(t, t.Context(), ctx)
		assert.Equal(t, set, change)
		operations = append(operations, "write")
		return nil
	}
	deleteFuture := func(ctx context.Context) error {
		assert.Equal(t, t.Context(), ctx)
		operations = append(operations, "delete future")
		return nil
	}
	require.NoError(t, setBudget(t.Context(), set, list, write, deleteFuture))
	assert.Equal(t, []string{"delete future", "write"}, operations)
}

func TestSetBudget_ReturnsStorageErrors(t *testing.T) {
	storageError := errors.New("storage failed")
	tests := []struct {
		name       string
		overwrite  bool
		failRead   bool
		failWrite  int
		failDelete bool
		writes     int
	}{
		{name: "read", failRead: true},
		{name: "preserve following month", failWrite: 1, writes: 1},
		{name: "write current month", failWrite: 2, writes: 2},
		{name: "delete future", overwrite: true, failDelete: true},
		{name: "write after delete", overwrite: true, failWrite: 1, writes: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := &BudgetSet{CategoryID: uuid.Must(uuid.NewV4()), Year: 2025, Month: 3, Amount: decimal.NewFromInt(450), OverwriteFutureMonths: tt.overwrite}
			writes := 0
			list := func(context.Context, int, int, int, int) ([]*Budget, error) {
				if tt.failRead {
					return nil, storageError
				}
				return nil, nil
			}
			write := func(context.Context, *BudgetSet, bool) error {
				writes++
				if writes == tt.failWrite {
					return storageError
				}
				return nil
			}
			deleteFuture := func(context.Context) error {
				if tt.failDelete {
					return storageError
				}
				return nil
			}
			assert.ErrorIs(t, setBudget(t.Context(), set, list, write, deleteFuture), storageError)
			assert.Equal(t, tt.writes, writes)
		})
	}
}
