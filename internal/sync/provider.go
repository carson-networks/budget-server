package sync

import (
	"context"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
	"github.com/gofrs/uuid/v5"
)

// SyncResult is the outcome of a provider sync: per-account actions plus optional actions to run
// only after every per-account batch has committed successfully (e.g. Plaid cursor updates).
type SyncResult struct {
	ByAccount map[uuid.UUID][]actions.IAction
	OnSuccess []actions.IAction
}

type Provider interface {
	Type() syncstore.SyncType
	Sync(ctx context.Context, reader *storage.Reader, accountIDs []uuid.UUID) (*SyncResult, error)
}
