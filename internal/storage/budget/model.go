package budget

import (
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// Budget represents a budget record (composite PK: category_id, month).
type Budget struct {
	CategoryID uuid.UUID
	Month      time.Time
	Amount     decimal.Decimal
}

// BudgetSet is used when setting/upserting a budget row.
type BudgetSet struct {
	CategoryID uuid.UUID
	Month      time.Time
	Amount     decimal.Decimal
}

func bobBudgetToBudget(row *bobgen.Budget) *Budget {
	return &Budget{
		CategoryID: row.CategoryID,
		Month:      row.Month,
		Amount:     row.Amount,
	}
}
