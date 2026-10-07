package budget

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPreserveFutureMonths(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	tests := []struct {
		name        string
		rows        []*Budget
		editedMonth int
		editedYear  int
		amount      decimal.Decimal
		keep        bool
	}{
		{
			name: "pin the latest amount on or before the edited month",
			rows: []*Budget{
				{CategoryID: id, Year: 2025, Month: 1, Amount: decimal.NewFromInt(100)},
				{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(400)},
			},
			editedMonth: 3,
			editedYear:  2025,
			amount:      decimal.NewFromInt(400),
			keep:        true,
		},
		{
			name: "pin an earlier month when nothing is adjacent",
			rows: []*Budget{
				{CategoryID: id, Year: 2025, Month: 10, Amount: decimal.NewFromInt(250)},
			},
			editedMonth: 12,
			editedYear:  2025,
			amount:      decimal.NewFromInt(250),
			keep:        true,
		},
		{
			name: "leave an existing next month alone",
			rows: []*Budget{
				{CategoryID: id, Year: 2025, Month: 1, Amount: decimal.NewFromInt(100)},
				{CategoryID: id, Year: 2025, Month: 4, Amount: decimal.NewFromInt(600)},
			},
			editedMonth: 3,
			editedYear:  2025,
			keep:        false,
		},
		{
			name:        "pin zero when the category has no budget",
			rows:        []*Budget{{CategoryID: other, Year: 2025, Month: 2, Amount: decimal.NewFromInt(999)}},
			editedMonth: 3,
			editedYear:  2025,
			amount:      decimal.Zero,
			keep:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, keep := preserveFutureMonths(tt.rows, id, tt.editedMonth, tt.editedYear)
			assert.Equal(t, tt.keep, keep)
			assert.True(t, amount.Equal(tt.amount))
		})
	}
}
