package v1Transaction

import "github.com/carson-networks/budget-server/internal/connecthandlers/v1"

// Service implements transaction.v1.TransactionService.
type Service struct {
	connecthandlers.Deps
}
