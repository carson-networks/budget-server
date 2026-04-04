package connecthandlers

import (
	"github.com/carson-networks/budget-server/internal/operator"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	budgetsync "github.com/carson-networks/budget-server/internal/sync"
)

// Deps are shared dependencies for all Connect service handlers.
type Deps struct {
	Storage      *storage.Storage
	Operator     operator.IProcessor
	PlaidClient  *plaidclient.Client
	Orchestrator *budgetsync.Orchestrator
}
