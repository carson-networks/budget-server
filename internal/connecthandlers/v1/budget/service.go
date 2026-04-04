package v1Budget

import "github.com/carson-networks/budget-server/internal/connecthandlers/v1"

// Service implements budget.v1.BudgetService.
type Service struct {
	connecthandlers.Deps
}
