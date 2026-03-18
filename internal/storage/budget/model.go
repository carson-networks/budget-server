package budget

import (
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// Budget represents a budget record (composite PK: category_id, month).
type Budget struct {
	CategoryID uuid.UUID
	Month      int // 1-12
	Year       int
	Amount     decimal.Decimal
}

// BudgetSet is used when setting/upserting a budget row.
type BudgetSet struct {
	CategoryID            uuid.UUID
	Month                 int // 1-12
	Year                  int
	Amount                decimal.Decimal
	OverwriteFutureMonths bool
}

func bobBudgetToBudget(row *bobgen.Budget) *Budget {
	return &Budget{
		CategoryID: row.CategoryID,
		Month:      int(row.Month.Month()),
		Year:       row.Month.Year(),
		Amount:     row.Amount,
	}
}
