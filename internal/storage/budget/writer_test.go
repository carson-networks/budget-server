package budget

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAmountToPreserve(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	other := uuid.Must(uuid.NewV4())
	tests := []struct {
		name   string
		rows   []*Budget
		amount decimal.Decimal
		keep   bool
	}{
		{
			name:   "pin the amount already in effect",
			rows:   []*Budget{{CategoryID: id, Year: 2025, Month: 3, Amount: decimal.NewFromInt(400)}},
			amount: decimal.NewFromInt(400),
			keep:   true,
		},
		{
			name: "leave an existing next month alone",
			rows: []*Budget{{CategoryID: id, Year: 2025, Month: 4, Amount: decimal.NewFromInt(600)}},
			keep: false,
		},
		{
			name:   "pin zero when the category has no budget",
			rows:   []*Budget{{CategoryID: other, Year: 2025, Month: 2, Amount: decimal.NewFromInt(999)}},
			amount: decimal.Zero,
			keep:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, keep := amountToPreserve(tt.rows, id, 4, 2025)
			assert.Equal(t, tt.keep, keep)
			assert.True(t, amount.Equal(tt.amount))
		})
	}
}
