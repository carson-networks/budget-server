package account

import connecthandlers "github.com/carson-networks/budget-server/internal/connecthandlers/v1"

// Service implements budget.v1.AccountService.
type Service struct {
	connecthandlers.Deps
}
