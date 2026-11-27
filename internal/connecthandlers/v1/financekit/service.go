package v1FinanceKit

import (
	financekit "github.com/carson-networks/budget-server/internal/connecthandlers/gen/financekit/v1"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1"
)

// Service implements financekit.v1.FinanceKitService. Every RPC is unimplemented until phase 4.
type Service struct {
	connecthandlers.Deps
	financekit.UnimplementedFinanceKitServiceHandler
}
